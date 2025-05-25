package di

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pixie-sh/errors-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AppConfig represents an application configuration that would typically be loaded from a JSON file
type AppConfig struct {
	DatabaseURL       string `json:"database_url"`
	ServerPort        int    `json:"server_port"`
	LogLevel          string `json:"log_level"`
	EnableMetrics     bool   `json:"enable_metrics"`
	MaxConnections    int    `json:"max_connections"`
	ConnectionTimeout int    `json:"connection_timeout"`
}

// DatabaseService demonstrates a service that requires configuration
type DatabaseService struct {
	Config      AppConfig
	IsConnected bool
}

func (d *DatabaseService) Connect() error {
	// In a real implementation, this would connect to the database using the config
	if d.Config.DatabaseURL == "" {
		return errors.New("database URL is not set", "db_connection_error")
	}
	d.IsConnected = true
	return nil
}

// Helper function to create a temporary config file for testing
func createTempConfigFile(t *testing.T, config AppConfig) string {
	// Create a temporary directory for our test
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Marshal our config to JSON
	configData, err := json.Marshal(config)
	require.NoError(t, err, "Failed to marshal config to JSON")

	// Write the config to a temporary file
	err = os.WriteFile(configPath, configData, 0644)
	require.NoError(t, err, "Failed to write config file")

	return configPath
}

// Test that demonstrates how to register and inject configuration
func TestConfigurationInjection(t *testing.T) {
	// 1. Create a test registry instead of using the global one
	registry := NewRegistry()

	// 2. Define our test configuration
	expectedConfig := AppConfig{
		DatabaseURL:       "postgres://user:pass@localhost:5432/testdb",
		ServerPort:        8080,
		LogLevel:          "debug",
		EnableMetrics:     true,
		MaxConnections:    100,
		ConnectionTimeout: 30,
	}

	// 3. Create a temp config file with our test data
	configPath := createTempConfigFile(t, expectedConfig)

	// 4. Register configuration provider that loads from JSON file
	err := RegisterConfiguration[AppConfig](
		func(ctx Context, opts RegistryOpts) (AppConfig, error) {
			// This function demonstrates how to load config from a file
			// In a real application, the path might come from ctx or environment
			data, err := os.ReadFile(configPath)
			if err != nil {
				return AppConfig{}, errors.Wrap(err, "failed to read config file", "config_read_error")
			}

			var config AppConfig
			if err := json.Unmarshal(data, &config); err != nil {
				return AppConfig{}, errors.Wrap(err, "failed to parse config file", "config_parse_error")
			}

			return config, nil
		},
		WithRegistry(registry),
	)
	require.NoError(t, err, "Failed to register configuration")

	// 5. Register the database service that requires this configuration
	err = Register[*DatabaseService](
		func(ctx Context, opts RegistryOpts) (*DatabaseService, error) {
			// Get the configuration
			config, err := CreateConfiguration[AppConfig](ctx, WithRegistry(registry))
			if err != nil {
				return nil, errors.Wrap(err, "failed to get database configuration", "config_error")
			}

			// Create and return the service with the injected configuration
			return &DatabaseService{
				Config: config,
			}, nil
		},
		func(opts *RegistryOpts) {
			opts.Registry = registry
		},
	)
	require.NoError(t, err, "Failed to register database service")

	// 6. Create a DI context
	ctx := NewContext()

	// 7. Create the database service through DI
	dbService, err := Create[*DatabaseService](ctx,
		func(opts *RegistryOpts) {
			opts.Registry = registry
		},
	)
	require.NoError(t, err, "Failed to create database service")

	// 8. Validate that the configuration was properly injected
	assert.Equal(t, expectedConfig.DatabaseURL, dbService.Config.DatabaseURL)
	assert.Equal(t, expectedConfig.ServerPort, dbService.Config.ServerPort)
	assert.Equal(t, expectedConfig.LogLevel, dbService.Config.LogLevel)
	assert.Equal(t, expectedConfig.EnableMetrics, dbService.Config.EnableMetrics)
	assert.Equal(t, expectedConfig.MaxConnections, dbService.Config.MaxConnections)
	assert.Equal(t, expectedConfig.ConnectionTimeout, dbService.Config.ConnectionTimeout)

	// 9. Verify the service works with the injected configuration
	err = dbService.Connect()
	assert.NoError(t, err)
	assert.True(t, dbService.IsConnected)
}

// Alternative test showing how to use RegisterPair for services with configuration
func TestConfigurationInjectionWithPair(t *testing.T) {
	// 1. Create a test registry
	registry := NewRegistry()

	// 2. Define our test configuration
	expectedConfig := AppConfig{
		DatabaseURL:       "postgres://user:pass@localhost:5432/testdb",
		ServerPort:        8080,
		LogLevel:          "debug",
		EnableMetrics:     true,
		MaxConnections:    100,
		ConnectionTimeout: 30,
	}

	// 3. Create a temp config file with our test data
	configPath := createTempConfigFile(t, expectedConfig)

	// 4. Register both the config provider and service with RegisterPair
	err := RegisterPair[*DatabaseService, AppConfig](
		// Service creation function that takes config
		func(ctx Context, opts RegistryOpts, config AppConfig) (*DatabaseService, error) {
			return &DatabaseService{
				Config: config,
			}, nil
		},
		// Configuration creation function
		func(ctx Context, opts RegistryOpts) (AppConfig, error) {
			data, err := os.ReadFile(configPath)
			if err != nil {
				return AppConfig{}, errors.Wrap(err, "failed to read config file", "config_read_error")
			}

			var config AppConfig
			if err := json.Unmarshal(data, &config); err != nil {
				return AppConfig{}, errors.Wrap(err, "failed to parse config file", "config_parse_error")
			}

			return config, nil
		},
		func(opts *RegistryOpts) {
			opts.Registry = registry
		},
	)
	require.NoError(t, err, "Failed to register database service pair")

	// 5. Create a DI context
	ctx := NewContext()

	// 6. Create the service using CreatePair which handles the config injection
	dbService, err := CreatePair[*DatabaseService, AppConfig](ctx, WithRegistry(registry))
	require.NoError(t, err, "Failed to create database service")

	// 7. Validate that the configuration was properly injected
	assert.Equal(t, expectedConfig.DatabaseURL, dbService.Config.DatabaseURL)
	assert.Equal(t, expectedConfig.ServerPort, dbService.Config.ServerPort)
	assert.Equal(t, expectedConfig.LogLevel, dbService.Config.LogLevel)
	assert.Equal(t, expectedConfig.EnableMetrics, dbService.Config.EnableMetrics)
	assert.Equal(t, expectedConfig.MaxConnections, dbService.Config.MaxConnections)
	assert.Equal(t, expectedConfig.ConnectionTimeout, dbService.Config.ConnectionTimeout)

	// 8. Verify the service works with the injected configuration
	err = dbService.Connect()
	assert.NoError(t, err)
	assert.True(t, dbService.IsConnected)
}
