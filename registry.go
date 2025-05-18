package di

import (
	"github.com/pixie-sh/errors-go"
)

var Instance Registry = newRegistry()

// Registry provides a dependency injection container interface for managing
// dependencies and their configurations. It allows registration and creation
// of both regular dependencies and their configurations.
//
// The interface supports:
// - Registering dependencies with their creation functions
// - Registering configuration providers
// - Creating instances of registered dependencies
// - Creating configuration objects for registered dependencies
//
// All methods accept a context and optional registry options to control
// the dependency creation process.
type Registry interface {
	Create(ctx Ctx, typeNameOf string, c any, opts RegistryOpts) (any, error)
	CreateConfiguration(ctx Ctx, typeNameOf string, opts RegistryOpts) (any, error)

	Register(typeNameOf string, createFn func(ctx Ctx, opts RegistryOpts, c any) (any, error), opts RegistryOpts) error
	RegisterConfiguration(typeNameOf string, createCfgFn func(ctx Ctx, opts RegistryOpts) (any, error), opts RegistryOpts) error
}

type registration struct {
	creator CreateInstanceHandler
	opts    RegistryOpts
}

type configurationRegistration struct {
	creator CreateConfigurationHandler
	opts    RegistryOpts
}

// diRegistry implements the Registry interface and serves as a dependency injection container.
// It manages two types of registrations:
// - registrations: stores regular dependency creators mapped by their type names
// - configurationRegistrations: stores configuration creators mapped by their type names
//
// The struct provides methods to register and create both regular dependencies
// and their configurations, maintaining them in separate maps for clear separation
// of concerns and easier management.
type diRegistry struct {
	registrations              map[string]registration
	configurationRegistrations map[string]configurationRegistration
}

func newRegistry() diRegistry {
	return diRegistry{registrations: map[string]registration{}, configurationRegistrations: map[string]configurationRegistration{}}
}

func (dif diRegistry) Register(typeNameOf string, createFn func(ctx Ctx, opts RegistryOpts, config any) (any, error), opts RegistryOpts) error {
	dif.registrations[typeNameOf] = registration{creator: createFn, opts: opts}
	return nil
}

func (dif diRegistry) RegisterConfiguration(typeNameOf string, createCfgFn func(ctx Ctx, opts RegistryOpts) (any, error), opts RegistryOpts) error {
	dif.configurationRegistrations[typeNameOf] = configurationRegistration{creator: createCfgFn, opts: opts}
	return nil
}

func (dif diRegistry) Create(ctx Ctx, typeNameOf string, config any, opts RegistryOpts) (any, error) {
	reg, ok := dif.registrations[typeNameOf]
	if !ok {
		return nil, errors.New("dependency not registered: %s", typeNameOf, DependencyMissingErrorCode)
	}

	return reg.creator(ctx, opts, config)
}

func (dif diRegistry) CreateConfiguration(ctx Ctx, typeNameOf string, opts RegistryOpts) (any, error) {
	reg, ok := dif.configurationRegistrations[typeNameOf]
	if !ok {
		return nil, errors.New("configuration dependency not registered: %s", typeNameOf, DependencyMissingErrorCode)
	}

	return reg.creator(ctx, opts)
}
