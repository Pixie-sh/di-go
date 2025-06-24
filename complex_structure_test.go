package di

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test for singleton pattern
type databaseTest struct {
	ConnectionString string
	initialized      bool
}

// Configuration for databaseTest
type databaseConfigTest struct {
	ConnectionString string
}

func (d databaseConfigTest) LookupNode(lookupPath string) (any, error) {
	panic("implement me")
}

// Test for complex object with multiple dependencies
type serviceTest struct {
	DB               *databaseTest
	Logger           *loggerTest
	MetricsCollector *metricsCollectorTest
}

type loggerTest struct {
	Level string
}

type metricsCollectorTest struct {
	Endpoint string
}

// Test for singleton implementation
func Test_SingletonPattern(t *testing.T) {
	// Reset the test registry before test
	Instance = NewRegistry()

	// Our singleton instance counter
	instanceCounter := 0
	var singletonInstance *databaseTest
	var mu sync.Mutex

	// Custom test factory for tracking registrations
	customFactory := &TestFactory{}

	// Register databaseTest as a singleton with configuration
	require.NoError(t, RegisterPair[*databaseTest, *databaseConfigTest](
		// Instance creator function
		func(ctx Context, opts *RegistryOpts, config *databaseConfigTest) (*databaseTest, error) {
			mu.Lock()
			defer mu.Unlock()

			// Only create one instance no matter how many times we're called
			if singletonInstance == nil {
				singletonInstance = &databaseTest{
					ConnectionString: config.ConnectionString,
					initialized:      true,
				}
				instanceCounter++
			}

			return singletonInstance, nil
		},
		// Configuration creator function
		func(ctx Context, opts *RegistryOpts) (*databaseConfigTest, error) {
			return &databaseConfigTest{
				ConnectionString: "mongodb://localhost:27017",
			}, nil
		},
		WithRegistry(customFactory),
	))

	// Create multiple instances - they should all be the same object
	db1, err := CreatePair[*databaseTest, *databaseConfigTest](NewContext(), WithRegistry(customFactory))
	require.NoError(t, err)
	require.NotNil(t, db1)

	db2, err := CreatePair[*databaseTest, *databaseConfigTest](NewContext(), WithRegistry(customFactory))
	require.NoError(t, err)
	require.NotNil(t, db2)

	db3, err := CreatePair[*databaseTest, *databaseConfigTest](NewContext(), WithRegistry(customFactory))
	require.NoError(t, err)
	require.NotNil(t, db3)

	// Verify all instances are the same object (pointer comparison)
	assert.Equal(t, db1, db2)
	assert.Equal(t, db2, db3)

	// Verify only one instance was created
	assert.Equal(t, 1, instanceCounter)

	// Verify our factory was called
	require.Contains(t, regCalls, "Register:di.databaseTest;di.databaseConfigTest")
	require.Contains(t, regCalls, "RegisterConf:di.databaseConfigTest;di.databaseTest")
}

// Test for complex object with multiple dependencies
func Test_ComplexObjectWithDependencies(t *testing.T) {
	// Reset the test registry
	Instance = NewRegistry()
	regCalls = []string{}

	// Our test factory for tracking registrations
	customFactory := &TestFactory{}

	// Register all dependencies
	require.NoError(t, Register[*loggerTest](
		func(ctx Context, opts *RegistryOpts) (*loggerTest, error) {
			return &loggerTest{Level: "INFO"}, nil
		},
		WithRegistry(customFactory),
	))

	require.NoError(t, Register[*metricsCollectorTest](
		func(ctx Context, opts *RegistryOpts) (*metricsCollectorTest, error) {
			return &metricsCollectorTest{Endpoint: "http://metrics.example.com"}, nil
		},
		WithRegistry(customFactory),
	))

	// Register databaseTest with config
	require.NoError(t, RegisterPair[*databaseTest, *databaseConfigTest](
		func(ctx Context, opts *RegistryOpts, config *databaseConfigTest) (*databaseTest, error) {
			return &databaseTest{
				ConnectionString: config.ConnectionString,
				initialized:      true,
			}, nil
		},
		func(ctx Context, opts *RegistryOpts) (*databaseConfigTest, error) {
			return &databaseConfigTest{ConnectionString: "mongodb://localhost:27017"}, nil
		},
		WithRegistry(customFactory),
	))

	// Register serviceTest that depends on all other components
	require.NoError(t, Register[*serviceTest](
		func(ctx Context, opts *RegistryOpts) (*serviceTest, error) {
			// Create all dependencies through the DI container
			db, err := CreatePair[*databaseTest, *databaseConfigTest](ctx, WithRegistry(customFactory))
			if err != nil {
				return nil, err
			}

			logger, err := Create[*loggerTest](ctx, WithRegistry(customFactory))
			if err != nil {
				return nil, err
			}

			metrics, err := Create[*metricsCollectorTest](ctx, WithRegistry(customFactory))
			if err != nil {
				return nil, err
			}

			return &serviceTest{
				DB:               db,
				Logger:           logger,
				MetricsCollector: metrics,
			}, nil
		},
		WithRegistry(customFactory),
	))

	// Create the complex service with all its dependencies
	service, err := Create[*serviceTest](NewContext(), WithRegistry(customFactory))
	require.NoError(t, err)
	require.NotNil(t, service)

	// Verify all dependencies were injected
	require.NotNil(t, service.DB)
	require.NotNil(t, service.Logger)
	require.NotNil(t, service.MetricsCollector)

	// Verify specific values
	assert.Equal(t, "mongodb://localhost:27017", service.DB.ConnectionString)
	assert.Equal(t, "INFO", service.Logger.Level)
	assert.Equal(t, "http://metrics.example.com", service.MetricsCollector.Endpoint)

	// Verify our factory was used for all registrations
	require.Len(t, regCalls, 5)
}

// Test for named instances with injection tokens
func Test_NamedInstances(t *testing.T) {
	// Reset the test registry
	Instance = NewRegistry()

	regCalls = []string{}
	customFactory := &TestFactory{}

	// Register multiple database instances with different tokens
	require.NoError(t, RegisterPair[*databaseTest, *databaseConfigTest](
		func(ctx Context, opts *RegistryOpts, config *databaseConfigTest) (*databaseTest, error) {
			return &databaseTest{
				ConnectionString: config.ConnectionString,
				initialized:      true,
			}, nil
		},
		func(ctx Context, opts *RegistryOpts) (*databaseConfigTest, error) {
			return &databaseConfigTest{ConnectionString: "mongodb://primary:27017"}, nil
		},
		WithRegistry(customFactory),
		WithToken("primary"),
	))

	require.NoError(t, RegisterPair[*databaseTest, *databaseConfigTest](
		func(ctx Context, opts *RegistryOpts, config *databaseConfigTest) (*databaseTest, error) {
			return &databaseTest{
				ConnectionString: config.ConnectionString,
				initialized:      true,
			}, nil
		},
		func(ctx Context, opts *RegistryOpts) (*databaseConfigTest, error) {
			return &databaseConfigTest{ConnectionString: "mongodb://replica:27017"}, nil
		},
		WithRegistry(customFactory),
		WithToken("replica"),
	))

	// Create the named instances
	primaryDB, err := CreatePair[*databaseTest, *databaseConfigTest](
		NewContext(),
		WithRegistry(customFactory),
		WithToken("primary"),
	)
	require.NoError(t, err)
	require.NotNil(t, primaryDB)

	replicaDB, err := CreatePair[*databaseTest, *databaseConfigTest](
		NewContext(),
		WithRegistry(customFactory),
		WithToken("replica"),
	)
	require.NoError(t, err)
	require.NotNil(t, replicaDB)

	// Verify they are different instances with different connection strings
	assert.NotEqual(t, primaryDB, replicaDB)
	assert.Equal(t, "mongodb://primary:27017", primaryDB.ConnectionString)
	assert.Equal(t, "mongodb://replica:27017", replicaDB.ConnectionString)

	// Verify our factory was called for all registrations with correct tokens
	require.Contains(t, regCalls, "Register:primary:di.databaseTest;primary:di.databaseConfigTest")
	require.Contains(t, regCalls, "Register:replica:di.databaseTest;replica:di.databaseConfigTest")
}
