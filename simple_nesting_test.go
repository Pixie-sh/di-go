package di

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// Example types for nested dependency
type A struct {
	B *B
}
type B struct {
	C *C
}
type C struct {
	Value int
}

// Custom test factory that delegates to di.Instance but counts calls
var regCalls []string

// Create a struct that implements the Registry interface
type TestFactory struct{}

// Implement the Registry interface methods

// Register tracks the registration and delegates to di.Instance
func (f *TestFactory) Register(typeNameOf string, createFn func(ctx Context, opts RegistryOpts, c any) (any, error), opts RegistryOpts) error {
	regCalls = append(regCalls, "Register:"+typeNameOf)
	return Instance.Register(typeNameOf, createFn, opts)
}

// RegisterConfiguration tracks configuration registration and delegates to di.Instance
func (f *TestFactory) RegisterConfiguration(typeNameOf string, createCfgFn func(ctx Context, opts RegistryOpts) (any, error), opts RegistryOpts) error {
	regCalls = append(regCalls, "RegisterConf:"+typeNameOf)
	return Instance.RegisterConfiguration(typeNameOf, createCfgFn, opts)
}

// Create delegates to di.Instance
func (f *TestFactory) Create(ctx Context, typeNameOf string, c any, opts RegistryOpts) (any, error) {
	return Instance.Create(ctx, typeNameOf, c, opts)
}

// CreateConfiguration delegates to di.Instance
func (f *TestFactory) CreateConfiguration(ctx Context, typeNameOf string, opts RegistryOpts) (any, error) {
	return Instance.CreateConfiguration(ctx, typeNameOf, opts)
}

// Create an instance of our TestFactory

func Test_NestedRegistrationsWithFactory(t *testing.T) {
	customFactory := &TestFactory{}

	// Register the deepest first: C
	require.NoError(t, Register[*C](func(ctx Context, opts RegistryOpts) (*C, error) {
		return &C{Value: 42}, nil
	}, WithRegistry(customFactory)))

	// B depends on C
	require.NoError(t, Register[*B](func(ctx Context, opts RegistryOpts) (*B, error) {
		c, err := Create[*C](ctx, func(opts *RegistryOpts) { opts.Registry = customFactory })
		if err != nil {
			return nil, err
		}
		return &B{C: c}, nil
	}, WithRegistry(customFactory)))

	// A depends on B
	require.NoError(t, Register[*A](func(ctx Context, opts RegistryOpts) (*A, error) {
		b, err := Create[*B](ctx, func(opts *RegistryOpts) { opts.Registry = customFactory })
		if err != nil {
			return nil, err
		}
		return &A{B: b}, nil
	}, WithRegistry(customFactory)))

	// Act: Resolve top-level dependency (A), traversing dependencies via factory
	a, err := Create[*A](NewContext(), func(opts *RegistryOpts) { opts.Registry = customFactory })
	require.NoError(t, err)
	require.NotNil(t, a)
	require.NotNil(t, a.B)
	require.NotNil(t, a.B.C)
	require.Equal(t, 42, a.B.C.Value)

	// Optional: Check that our factory was used
	require.NotEmpty(t, regCalls)
}
