package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PromMetrics is the global metrics instance to be accessed throughout the code
var PromMetrics Metrics

func init() {
	// initializing with an empty config for unit tests
	InitializeMetrics(Config{})
}

// Metrics holds the prometheus registry and instances of individual metrics
type Metrics struct {
	// HTTP metrics
	HTTPLatency              *prometheus.HistogramVec
	HTTPCount                *prometheus.CounterVec
	HTTPInflightRequestCount *prometheus.GaugeVec

	// DB metrics
	DBLatency *prometheus.HistogramVec

	CircuitTrips *prometheus.GaugeVec
	PanicCount   prometheus.Counter
	httpHandler  http.Handler
}

// Config provides parameters for instantiating the Metrics
type Config struct {
	// Prefix will be applied to all app-specific metrics, but not to Go runtime
	// metrics or process metrics
	Prefix string

	// Labels will be applied to all metrics
	Labels map[string]string

	// ErrorLogger will be used to log Prometheus errors
	ErrorLogger func(v ...interface{})

	// MaxRequestsInFlight is the number of maximum concurrent scrape
	// requests. See prometheus.HandlerOpts for more info.
	MaxRequestsInFlight int
}

// InitializeMetrics initializes all the Prometheus metrics
func InitializeMetrics(cfg Config) {
	reg := prometheus.NewRegistry()

	// create a wrapper registerer so we can add constant labels to these collectors
	labelRegistry := prometheus.WrapRegistererWith(prometheus.Labels(cfg.Labels), reg)
	labelRegistry.MustRegister(prometheus.NewGoCollector())

	/*labelRegistry.MustRegister(
	prometheus.NewProcessCollector(
		prometheus.ProcessCollectorOpts{
			ReportErrors: true,
		}))*/

	// Make sure metric name prefix (usually the app name) ends with `_`
	if cfg.Prefix != "" && cfg.Prefix[len(cfg.Prefix)-1] != '_' {
		cfg.Prefix = cfg.Prefix + "_"
	}

	promMetric := Metrics{}

	// Register the metrics with prefix and the constant labels
	promMetric.RegisterMetrics(
		prometheus.WrapRegistererWithPrefix(cfg.Prefix, labelRegistry),
	)

	promMetric.httpHandler = promhttp.InstrumentMetricHandler(
		labelRegistry, // handler metrics will be written to this registry
		promhttp.HandlerFor(
			reg, // handler will expose this registry

			promhttp.HandlerOpts{
				ErrorLog: promLogger(cfg.ErrorLogger),

				// If we get an error while collecting, then return an HTTP error
				ErrorHandling: promhttp.HTTPErrorOnError,

				MaxRequestsInFlight: cfg.MaxRequestsInFlight,

				// Not setting timeout because it doesn't prevent all the collector
				// work from being done anyways
			}))

	PromMetrics = promMetric
}

// RegisterMetrics initializes and registers all the metrics
func (p *Metrics) RegisterMetrics(reg prometheus.Registerer) {
	p.HTTPLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_latency",
		Help:    "The amount of time it takes to process http requests",
		Buckets: []float64{0.001, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 0.75, 1, 5, 10, 30, 60},
	}, []string{"code", "endpoint", "method"})
	reg.MustRegister(p.HTTPLatency)

	p.HTTPInflightRequestCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "http_in_flight_request",
		Help: "The amount of in flight http requests",
	}, []string{"endpoint"})
	reg.MustRegister(p.HTTPInflightRequestCount)

	p.HTTPCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_count_total",
		Help: "The total count of http request",
	}, []string{"code", "method", "handler"})
	reg.MustRegister(p.HTTPCount)

	p.PanicCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "panic_count_total",
		Help: "The total count of middleware caught panics",
	})
	reg.MustRegister(p.PanicCount)

	p.CircuitTrips = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "circuit_trips",
		Help: "The total count of circuit trips",
	}, []string{"circuit"})
	reg.MustRegister(p.CircuitTrips)

	p.DBLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "database_latency_seconds",
		Help:    "The number of seconds it takes to execute a database call",
		Buckets: []float64{0.0001, 0.001, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 0.75, 1, 5, 10, 30, 60},
	}, []string{"table_name", "result"})
	reg.MustRegister(p.DBLatency)
}

// ServeHTTP forwards the request onto the Prometheus http.Handler
func (p *Metrics) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	p.httpHandler.ServeHTTP(rw, req)
}

// promLogger is an adapter that satisfies the promhttp.Logger interface for
// logging errors
type promLogger func(v ...interface{})

func (p promLogger) Println(v ...interface{}) {
	p(v...)
}
