package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"
)

// Config holds all configurable values needed to run this service
type Config struct {
	// Port specifies the network port to listen on
	Port string `envconfig:"PORT" required:"false" default:"8080"`

	// LogLevel sets the level for leveled logs. Options are debug, info, warning, error
	LogLevel string `envconfig:"LOG_LEVEL" required:"false" default:"debug"`

	// BuildkiteToken is a Buildkite API key with GraphQL access
	BuildkiteToken string `envconfig:"BUILDKITE_TOKEN" required:"true"`

	// Application holds the GitHub app configs
	Applications []*ConfigApplication `yaml:"applications"`

	ContextTimeout time.Duration `envconfig:"DEFAULT_TIMEOUT" required:"false" default:"30s"`
}

// ConfigApplication is a single GitHub app configuration
type ConfigApplication struct {
	// Host is the github host this app is installed on, most commonly, github.com
	Host string `yaml:"host"`

	// AppID is the github app's unique ID
	AppID int64 `yaml:"appID"`

	// PrivateKeyPath points to a file on disk where the private key is located
	PrivateKeyPath string `yaml:"privateKeyPath"`

	// Accounts is the list of app installations
	Accounts []ConfigAccount `yaml:"accounts"`
}

// ConfigAccount refers to a single app installation
type ConfigAccount struct {
	// Name is the org or account on github the app is installed in
	Name string `yaml:"name"`

	// InstallationID is the app's installation ID
	InstallationID int64 `yaml:"installationID"`
}

func NewConfig(configPath string) (*Config, error) {
	filename, _ := filepath.Abs(configPath)
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to read config file %s: %w", filename, err)
	}

	config := &Config{}
	err = yaml.Unmarshal(yamlFile, config)
	if err != nil {
		return nil, fmt.Errorf("unable to parse config file: %w", err)
	}

	err = envconfig.Process("", config)
	if err != nil {
		return nil, fmt.Errorf("unable to process environment config: %w", err)
	}

	return config, nil
}

// AppConfigForHost returns the Application config for a given GitHub instance (github.com or GHES)
func (c *Config) AppConfigForHost(host string) (*ConfigApplication, error) {
	for _, hostConfig := range c.Applications {
		if hostConfig.Host == host {
			return hostConfig, nil
		}
	}

	return nil, fmt.Errorf("unable to find configuration for host '%s'", host)
}

// InstallationID returns the installation ID of an app in a given GitHub account/org
func (c *ConfigApplication) InstallationID(account string) (int64, error) {
	for _, accountConfig := range c.Accounts {
		if accountConfig.Name == account {
			return accountConfig.InstallationID, nil
		}
	}

	return 0, fmt.Errorf("unable to find installation ID for account '%s'", account)
}
