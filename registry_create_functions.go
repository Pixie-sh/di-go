package di

import (
	"reflect"

	"github.com/pixie-sh/errors-go"
)

// Create creates a new instance of type T using the provided context and options.
// It accepts generic type T and returns an instance of T along with any error that occurred.
// The options parameter allows customization of the registry options during creation.
func Create[T any](ctx Context, options ...func(opts *RegistryOpts)) (T, error) {
	registryOpts := RegistryOpts{
		Registry:       Instance,
		InjectionToken: "",
	}

	for _, opt := range options {
		if opt != nil {
			opt(&registryOpts)
		}
	}

	injectionCtx := ctx.Clone()
	injectionCtx.AppendBreadcrumb(registryOpts.InjectionToken)
	return createSingleWithToken[T](injectionCtx, &registryOpts)
}

// CreateConfiguration creates a new configuration instance of type T.
// It uses the provided context and options to create a configuration object.
// Returns the created configuration instance and any error that occurred during creation.
func CreateConfiguration[T any](ctx Context, options ...func(opts *RegistryOpts)) (T, error) {
	registryOpts := RegistryOpts{
		Registry:       Instance,
		InjectionToken: "",
	}

	for _, opt := range options {
		if opt != nil {
			opt(&registryOpts)
		}
	}

	injectionCtx := ctx.Clone()
	return createSingleConfigurationWithToken[T](injectionCtx, &registryOpts)
}

// CreatePair creates a pair of instances where T is the main type and CT is the configuration type.
// It accepts a context and optional registry options to customize the creation process.
// Returns an instance of type T and any error that occurred during creation.
func CreatePair[T any, CT any](ctx Context, options ...func(opts *RegistryOpts)) (T, error) {
	registryOpts := RegistryOpts{
		Registry:       Instance,
		InjectionToken: "",
	}

	for _, opt := range options {
		if opt != nil {
			opt(&registryOpts)
		}
	}

	injectionCtx := ctx.Clone()
	injectionCtx.AppendBreadcrumb(registryOpts.InjectionToken)
	return createPairWithToken[T, CT](injectionCtx, &registryOpts)
}

// createPairWithToken is an internal function that creates a pair of instances using a specific token.
// It handles both the creation of the configuration (CT) and the main type (T).
// The CT type can be either a concrete type or NoConfig.
// Returns the created instance of type T and any error that occurred.
func createPairWithToken[T any, CT any | NoConfig](ctx Context, opts *RegistryOpts) (T, error) {
	var (
		f               = Instance
		typedInstance   T
		ct              CT
		unknownInstance any
		unknownConfig   any
		err             error
		ok              bool
		token           = opts.InjectionToken
	)

	if opts.Registry != nil {
		f = opts.Registry
	}

	ctType := TypeName[CT](token)
	tType := TypeName[T](token)

	inputCTType := reflect.TypeOf(ct)
	noConfigType := reflect.TypeOf(NoConfig{})
	noConfigTypePtr := reflect.TypeOf(&NoConfig{})
	if inputCTType != noConfigType && inputCTType != noConfigTypePtr {
		typeName := PairTypeName(ctType, tType)
		unknownConfig, err = f.CreateConfiguration(ctx, typeName, opts)
		if err != nil {
			return typedInstance, errors.Wrap(err, "failed to create configuration dependency for %s", typeName, ErrorCreatingDependencyErrorCode)
		}

		ct, ok = unknownConfig.(CT)
		if !ok {
			panic(errors.New("failed to cast dependency to expected type (%s)", typeName, DependencyTypeMismatchErrorCode))
		}
	}

	unknownInstance, err = f.Create(ctx, PairTypeName(tType, ctType), ct, opts)
	if err != nil {
		return typedInstance, errors.Wrap(err, "failed to create dependency", ErrorCreatingDependencyErrorCode)
	}

	typedInstance, ok = unknownInstance.(T)
	if !ok {
		panic(errors.New("failed to cast dependency to expected type", DependencyTypeMismatchErrorCode))
	}

	return typedInstance, nil
}

// createSingleWithToken is an internal function that creates a single instance of type T using a token.
// It uses the provided context and registry options to create the instance.
// Returns the created instance and any error that occurred during creation.
func createSingleWithToken[T any](ctx Context, opts *RegistryOpts) (T, error) {
	var (
		f               = Instance
		typedInstance   T
		noopCfg         = struct{}{}
		unknownInstance any
		err             error
		ok              bool
		token           = opts.InjectionToken
	)

	if opts.Registry != nil {
		f = opts.Registry
	}

	tType := TypeName[T](token)
	unknownInstance, err = f.Create(ctx, tType, noopCfg, opts)
	_, isMissing := errors.Has(err, DependencyMissingErrorCode)
	if err != nil && !isMissing {
		return typedInstance, errors.Wrap(err, "failed to create dependency first try", ErrorCreatingDependencyErrorCode)
	}

	if isMissing {
		var secErr error
		tType = TypeName[T]()
		unknownInstance, secErr = f.Create(ctx, tType, noopCfg, opts)
		if secErr != nil {
			return typedInstance, errors.Wrap(secErr, "failed to create dependency second try", ErrorCreatingDependencyErrorCode).WithNestedError(err)
		}
	}

	// Try direct type assertion first
	typedInstance, ok = SafeTypeAssert[T](unknownInstance)
	if !ok {
		panic(errors.New("failed to cast dependency to expected type", DependencyTypeMismatchErrorCode))
	}

	return typedInstance, nil
}

// createSingleConfigurationWithToken is an internal function that creates a configuration instance.
// It creates a single configuration of type CT using the provided context and registry options.
// Returns the created configuration instance and any error that occurred.
func createSingleConfigurationWithToken[CT any](ctx Context, opts *RegistryOpts) (CT, error) {
	var (
		f               = Instance
		typedInstance   CT
		unknownInstance any
		err             error
		ok              bool
		token           = opts.InjectionToken
	)

	if opts.Registry != nil {
		f = opts.Registry
	}

	if opts.ConfigNode != nil {
		log := Logger.With("opts.config_node", opts)
		log.Debug("checking opts.ConfigNode for return type")
		typedInstance, ok = SafeTypeAssert[CT](opts.ConfigNode)
		if ok {
			log.Debug("returning opts.ConfigNode as CT")
			return typedInstance, nil
		}

		log.Debug("opts.ConfigNode is not of type CT. proceeding to create configuration")
	}

	tType := TypeName[CT](token)
	unknownInstance, err = f.CreateConfiguration(ctx, tType, opts)
	_, isMissing := errors.Has(err, DependencyMissingErrorCode)
	if err != nil && (!isMissing || len(token) == 0) {
		return typedInstance, errors.Wrap(err, "failed to create configuration dependency first try", ErrorCreatingDependencyErrorCode)
	}

	if isMissing {
		var secErr error
		tType = TypeName[CT]() //trying creation without token
		unknownInstance, secErr = f.CreateConfiguration(ctx, tType, opts)
		if secErr != nil {
			return typedInstance, errors.Wrap(secErr, "failed to create configuration dependency second try", ErrorCreatingDependencyErrorCode).WithNestedError(err)
		}
	}

	typedInstance, ok = SafeTypeAssert[CT](unknownInstance)
	if !ok {
		panic(errors.New("failed to cast dependency to expected type", DependencyTypeMismatchErrorCode))
	}

	return typedInstance, nil
}
