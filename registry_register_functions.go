package di

import (
	"github.com/pixie-sh/errors-go"
)

// RegisterPair registers a pair of types T and CT where T depends on CT for configuration.
// It takes a creation function that requires configuration and a configuration creation function.
// Options can be provided to customize the registration behavior.
func RegisterPair[T any, CT any](
	fn TypedCreateInstanceHandler[T, CT],
	fnCT TypedCreateInstanceNoConfigHandler[CT],
	options ...func(opts *RegistryOpts)) error {
	registryOpts := RegistryOpts{
		Registry:       Instance,
		InjectionToken: "",
	}

	for _, opt := range options {
		if opt != nil {
			opt(&registryOpts)
		}
	}

	return registerPairWithToken[T, CT](fn, fnCT, &registryOpts)
}

// Register adds a single type T to the registry without configuration dependencies.
// It takes a creation function that doesn't require configuration.
// Options can be provided to customize the registration behavior.
func Register[T any](fn TypedCreateInstanceNoConfigHandler[T], options ...func(*RegistryOpts)) error {
	registryOpts := RegistryOpts{
		Registry:       Instance,
		InjectionToken: "",
	}

	for _, opt := range options {
		if opt != nil {
			opt(&registryOpts)
		}
	}

	return registerSingleWithToken[T](fn, &registryOpts)
}

// RegisterConfiguration registers a configuration type T in the registry.
// It takes a creation function that generates configuration instances.
// Options can be provided to customize the registration behavior.
func RegisterConfiguration[T any](fn TypedCreateInstanceNoConfigHandler[T], options ...func(*RegistryOpts)) error {
	registryOpts := RegistryOpts{
		Registry:       Instance,
		InjectionToken: "",
	}

	for _, opt := range options {
		if opt != nil {
			opt(&registryOpts)
		}
	}

	return registerSingleConfigurationWithToken[T](fn, &registryOpts)
}

// registerPairWithToken is an internal function that handles the registration of a type pair with specific tokens.
// It registers both the configuration type CT and the dependent type T with their respective creation functions.
func registerPairWithToken[T any, CT any](fn TypedCreateInstanceHandler[T, CT], fnCT TypedCreateInstanceNoConfigHandler[CT], opts *RegistryOpts) error {
	var (
		f     = Instance
		err   error
		token = opts.InjectionToken
	)

	if opts.Registry != nil {
		f = opts.Registry
	}

	ctType := TypeName[CT](token)
	tType := TypeName[T](token)
	pairTypeName := PairTypeName(ctType, tType)
	err = f.RegisterConfiguration(pairTypeName, fromHotMemoryRegisterNoConfig(f, fnCT, pairTypeName), opts)
	if err != nil {
		return errors.Wrap(err, "failed to RegisterPair configuration creator", ErrorCreatingDependencyErrorCode)
	}

	pairTypeName = PairTypeName(tType, ctType)
	err = f.Register(pairTypeName, fromHotMemoryRegisterWithConfig(f, fn, pairTypeName), opts)
	if err != nil {
		return errors.Wrap(err, "failed to RegisterPair creator", ErrorCreatingDependencyErrorCode)
	}

	return nil
}

// registerSingleWithToken is an internal function that registers a single type T with a specific token.
// It handles the registration of types that don't require configuration.
func registerSingleWithToken[T any](fn TypedCreateInstanceNoConfigHandler[T], opts *RegistryOpts) error {
	var (
		f     = Instance
		err   error
		token = opts.InjectionToken
	)

	if opts.Registry != nil {
		f = opts.Registry
	}

	tType := TypeName[T](token)
	fromHotFn := fromHotMemoryRegisterNoConfig(f, fn, tType)
	err = f.Register(tType, func(ctx Context, opts *RegistryOpts, _ any) (any, error) {
		return fromHotFn(ctx, opts)
	}, opts)
	if err != nil {
		return errors.Wrap(err, "failed to RegisterPair creator", ErrorCreatingDependencyErrorCode)
	}

	return nil
}

// registerSingleConfigurationWithToken is an internal function that registers a configuration type T with a specific token.
// It handles the registration of configuration types in the dependency injection system.
func registerSingleConfigurationWithToken[T any](fn TypedCreateInstanceNoConfigHandler[T], opts *RegistryOpts) error {
	var (
		f     = Instance
		err   error
		token = opts.InjectionToken
	)

	if opts.Registry != nil {
		f = opts.Registry
	}

	tType := TypeName[T](token)
	err = f.RegisterConfiguration(tType, fromHotMemoryRegisterNoConfig(f, fn, tType), opts)
	if err != nil {
		return errors.Wrap(err, "failed to RegisterPair creator", ErrorCreatingDependencyErrorCode)
	}

	return nil
}


func fromHotMemoryRegisterWithConfig[T any, CT any](f Registry, fn TypedCreateInstanceHandler[T, CT], typeName string) func(ctx Context, opts *RegistryOpts, c any) (any, error) {
	return func(ctx Context, opts *RegistryOpts, c any) (any, error) {
		resultInstance, err := f.GetHotInstance(ctx, opts, typeName)
		_, isMissing := errors.Has(err, DependencyMissingErrorCode)
		if err != nil && !isMissing {
			return resultInstance, err
		}

		if err == nil {
			return resultInstance, nil
		}

		resultInstance, err = fn(ctx, opts, c.(CT))
		if err != nil {
			return nil, err
		}

		err = f.SetHotInstance(ctx, opts, typeName, resultInstance)
		if err != nil {
			return nil, err
		}

		return resultInstance, nil
	}
}

func fromHotMemoryRegisterNoConfig[T any](f Registry, fn TypedCreateInstanceNoConfigHandler[T], typeName string) func(ctx Context, opts *RegistryOpts) (any, error) {
	return func(ctx Context, opts *RegistryOpts) (any, error) {
		resultInstance, err := f.GetHotInstance(ctx, opts, typeName)
		_, isMissing := errors.Has(err, DependencyMissingErrorCode)
		if err != nil && !isMissing {
			return resultInstance, err
		}

		if err == nil {
			return resultInstance, nil
		}

		resultInstance, err = fn(ctx, opts)
		if err != nil {
			return nil, err
		}

		err = f.SetHotInstance(ctx, opts, typeName, resultInstance)
		if err != nil {
			return nil, err
		}

		return resultInstance, nil
	}
}
