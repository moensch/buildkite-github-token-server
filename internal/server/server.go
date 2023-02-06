package server

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"go.uber.org/zap"

	"github.com/moensch/buildkite-github-token-server/internal/buildkite"
	"github.com/moensch/buildkite-github-token-server/internal/config"
	"github.com/moensch/buildkite-github-token-server/internal/github"
	"github.com/moensch/buildkite-github-token-server/internal/jwks"
	"github.com/moensch/buildkite-github-token-server/internal/metrics"
)

// Version holds the app version (Set at compile time)
var Version = "dev"

// appName holds the application name used for things such as log lines and metrics labels
var appName = "buildkite-github-token-server"

const (
	jwksURL     = "https://agent.buildkite.com/.well-known/jwks"
	jwtAudience = "https://buildkite.com/twilio-sandbox"
	jwtIssuer   = "https://agent.buildkite.com"
)

type Server struct {
	jwkCache         *jwks.JWKS
	httpServer       *http.Server
	config           config.Config
	log              *zap.Logger
	buildkite        *buildkite.Client
	github           *github.Client
	githubAppClients map[string]*github.Client
	port             string
}

func New(c config.Config) *Server {
	logger, err := zap.NewProduction()

	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	return &Server{
		log:       logger.With(zap.String("application", appName), zap.String("version", Version)),
		port:      c.Port,
		config:    c,
		buildkite: buildkite.NewClient(c.BuildkiteToken),
	}
}

func (srv *Server) Initialize() error {
	metrics.InitializeMetrics(metrics.Config{
		Prefix: strings.ReplaceAll(appName, "-", "_"),
		Labels: map[string]string{
			"app":         appName,
			"app_version": Version,
		},
		MaxRequestsInFlight: 2, // 2 for a High-Availability Prometheus setup

		// Log an error event if prometheus logs an error
		ErrorLogger: func(v ...interface{}) {
			srv.log.Error("prometheus error",
				zap.String("error", "prometheus error"),
				zap.String("message", fmt.Sprint(v...)),
			)
		},
	})

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		srv.log.Info("signal",
			zap.String("sig", fmt.Sprint(sig)),
		)
		srv.Close()
	}()

	// initialize additional clients
	cache, err := jwks.New(srv.log, jwksURL)
	if err != nil {
		return err
	}
	srv.jwkCache = cache

	srv.githubAppClients = make(map[string]*github.Client)
	for _, app := range srv.config.Applications {
		client, err := github.NewClientForHost(&srv.config, app.Host)
		if err != nil {
			return fmt.Errorf("cannot initialize github client for %s: %w", app.Host, err)
		}
		srv.githubAppClients[app.Host] = client
	}

	return nil
}

// Serve starts http server running on the port set in srv
func (srv *Server) Serve() {
	defer srv.Close()
	address := fmt.Sprintf(":%s", srv.port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		srv.log.Error("unable to serve",
			zap.Error(err),
			zap.String("address", address),
			zap.String("port", srv.port),
		)
		os.Exit(1)
	}
	srv.listen(listener)
}

// listen starts a server on the given listener. It allows for easier testability of the server.
func (srv *Server) listen(listener net.Listener) {
	router := chi.NewRouter()
	router.Use(srv.middlewareRecoverer)       // ensure server does not crash on panic
	router.Use(srv.jsonContentTypeMiddleware) // always set application/json content type
	router.Use(srv.logMiddleware)             // log all requests

	router.Post("/token", metricsMiddleware("token", srv.handleToken))
	router.Get("/metrics", metricsMiddleware("metrics", metrics.PromMetrics.ServeHTTP))

	// re-discover what port we are on. If config was to port :0, this will allow us to know what port we bound to
	port := listener.Addr().(*net.TCPAddr).Port
	srv.port = fmt.Sprintf("%d", port)
	srv.log.Info("server listening", zap.String("port", srv.port))

	srv.httpServer = &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: router}
	// TODO add timeouts to config
	srv.httpServer.WriteTimeout = 1 * time.Minute
	srv.httpServer.ReadTimeout = 1 * time.Minute

	if err := srv.httpServer.Serve(listener); err != nil {
		if err != http.ErrServerClosed {
			srv.log.Error("server crash", zap.Error(err))
			os.Exit(1)
		}
	}
}

// Close performs any remaining cleanup and shuts down
func (srv *Server) Close() error {
	// potentially doing many things that could error. Keep all errors and return at the end.
	var errs error
	// close socket to stop new requests from coming in
	if srv.httpServer != nil {
		err := srv.httpServer.Close()
		if err != nil {
			errs = fmt.Errorf("error closing server: %w", err)
		}
	}

	return errs
}
