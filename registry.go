package di

import (
	"github.com/pixie-sh/errors-go"
	"github.com/pixie-sh/logger-go/logger"
)

var Logger logger.Interface
var Instance Registry

func init() {
	Logger = logger.Logger
	Instance = NewRegistry()
}

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
	Create(ctx Context, typeNameOf string, c any, opts *RegistryOpts) (any, error)
	CreateConfiguration(ctx Context, typeNameOf string, opts *RegistryOpts) (any, error)
	GetHotInstance(ctx Context, opts *RegistryOpts, name string) (any, error)
	SetHotInstance(ctx Context, opts *RegistryOpts, name string, instance any) error

	Register(typeNameOf string, createFn func(ctx Context, opts *RegistryOpts, c any) (any, error), opts *RegistryOpts) error
	RegisterConfiguration(typeNameOf string, createCfgFn func(ctx Context, opts *RegistryOpts) (any, error), opts *RegistryOpts) error

}

type registration struct {
	creator CreateInstanceHandler
	opts    *RegistryOpts
}

type configurationRegistration struct {
	creator CreateConfigurationHandler
	opts    *RegistryOpts
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
	hotInstances               map[string]any
}

func NewRegistry() diRegistry {
	return diRegistry{registrations: map[string]registration{}, configurationRegistrations: map[string]configurationRegistration{}, hotInstances: map[string]any{}}
}

func (dif diRegistry) Register(typeNameOf string, createFn func(ctx Context, opts *RegistryOpts, config any) (any, error), opts *RegistryOpts) error {
	dif.registrations[typeNameOf] = registration{creator: createFn, opts: opts}
	return nil
}

func (dif diRegistry) RegisterConfiguration(typeNameOf string, createCfgFn func(ctx Context, opts *RegistryOpts) (any, error), opts *RegistryOpts) error {
	dif.configurationRegistrations[typeNameOf] = configurationRegistration{creator: createCfgFn, opts: opts}
	return nil
}

func (dif diRegistry) Create(ctx Context, typeNameOf string, config any, opts *RegistryOpts) (any, error) {
	reg, ok := dif.registrations[typeNameOf]
	if !ok {
		return nil, errors.New("dependency not registered: %s", typeNameOf, DependencyMissingErrorCode)
	}

	return reg.creator(ctx, opts, config)
}

func (dif diRegistry) CreateConfiguration(ctx Context, typeNameOf string, opts *RegistryOpts) (any, error) {
	reg, ok := dif.configurationRegistrations[typeNameOf]
	if !ok {
		return nil, errors.New("configuration dependency not registered: %s", typeNameOf, DependencyMissingErrorCode)
	}

	return reg.creator(ctx, opts)
}

func (dif diRegistry) GetHotInstance(ctx Context, opts *RegistryOpts, typeName string) (any, error) {
	key := typeName
	if opts != nil && opts.InjectionToken != "" {
		key = opts.InjectionToken.String() + ":" + typeName
	}

	instance, ok := dif.hotInstances[key]
	if !ok {
		return nil, errors.New("no hot instance found for: %s", key, DependencyMissingErrorCode)
	}

	return instance, nil
}

func (dif diRegistry) SetHotInstance(ctx Context, opts *RegistryOpts, typeName string, instance any) error {
	key := typeName
	if opts != nil && opts.InjectionToken != "" {
		key = opts.InjectionToken.String() + ":" + typeName
	}

	dif.hotInstances[key] = instance
	return nil
}