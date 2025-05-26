package di

import (
	"reflect"
	"strings"

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

	return createSingleWithToken[T](ctx, registryOpts)
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

	return createSingleConfigurationWithToken[T](ctx, registryOpts)
}

func ConfigurationLookup[T any](ctx Context, opts RegistryOpts) (T, error) {
	var result T

	if ctx == nil {
		return result, errors.New("di.Context cannot be nil", ConfigurationLookupErrorCode)
	}

	if ctx.Configuration() == nil {
		return result, errors.New("di.Context.Configuration() cannot be nil", ConfigurationLookupErrorCode)
	}


	lookupPath, err := lookupPath(ctx, opts)
	if err != nil {
		return result, errors.Wrap(err, "lookupPath cannot be empty", ConfigurationLookupErrorCode)
	}

	abstractNode, err := ctx.Configuration().LookupNode(lookupPath)
	if err != nil || abstractNode == nil {
		return result, errors.Wrap(err, "di.Context.Configuration().LookupNode() failed", ConfigurationLookupErrorCode)
	}

	typed, good := safeTypeAssert[T](abstractNode)
	if !good {
		return result, errors.New("di.Context.Configuration().LookupNode() returned an invalid type", ConfigurationLookupErrorCode)
	}

	return typed, nil
}

func ConfigurationNodeLookup(c any, path string) (any, error) {
	if path == "" {
		return c, nil
	}

	parts := strings.Split(path, ".")
	current := reflect.ValueOf(c)

	for _, part := range parts {
		// If current value is a pointer, dereference it
		if current.Kind() == reflect.Ptr {
			if current.IsNil() {
				return nil, errors.New("nil pointer encountered in path")
			}
			current = current.Elem()
		}

		// Only struct types can have fields
		if current.Kind() != reflect.Struct {
			return nil, errors.New("cannot access field '" + part + "' on non-struct type")
		}

		// Get the field by name
		field := current.FieldByName(part)
		if !field.IsValid() {
			// Try to find a JSON tag that matches the part
			foundField := false
			t := current.Type()
			for i := 0; i < t.NumField(); i++ {
				structField := t.Field(i)
				jsonTag := structField.Tag.Get("json")
				if jsonTag == part || strings.Split(jsonTag, ",")[0] == part {
					field = current.Field(i)
					foundField = true
					break
				}
			}
			if !foundField {
				return nil, errors.New("field '" + part + "' not found")
			}
		}

		current = field
	}

	// Return the interface value
	return current.Interface(), nil
}

func lookupPath(_ Context, opts RegistryOpts) (string, error) {
	if len(opts.InjectionToken) == 0 && len(opts.ConfigNode) == 0{
		return "", errors.New("di.RegistryOpts.InjectionToken and di.RegistryOpts.ConfigNode cannot be both empty", ConfigurationLookupErrorCode)
	}

	lp := opts.ConfigNode
	if len(opts.InjectionToken) > 0 {
		lp = opts.InjectionToken.String() + "." + opts.ConfigNode
	}

	if len(lp) == 0 {
		return "", errors.New("lookup path cannot be empty", ConfigurationLookupErrorCode)
	}

	return lp, nil
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

	return createPairWithToken[T, CT](ctx, registryOpts)
}

// createPairWithToken is an internal function that creates a pair of instances using a specific token.
// It handles both the creation of the configuration (CT) and the main type (T).
// The CT type can be either a concrete type or NoConfig.
// Returns the created instance of type T and any error that occurred.
func createPairWithToken[T any, CT any | NoConfig](ctx Context, opts RegistryOpts) (T, error) {
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
func createSingleWithToken[T any](ctx Context, opts RegistryOpts) (T, error) {
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
	if err != nil && (!isMissing || len(token) == 0) {
		return typedInstance, errors.Wrap(err, "failed to create dependency first try", ErrorCreatingDependencyErrorCode)
	}

	if isMissing {
		tType = TypeName[T]()
		unknownInstance, err = f.Create(ctx, tType, noopCfg, opts)
		if err != nil {
			return typedInstance, errors.Wrap(err, "failed to create dependency second try", ErrorCreatingDependencyErrorCode)
		}
	}

	// Try direct type assertion first
	typedInstance, ok = safeTypeAssert[T](unknownInstance)
	if !ok {
		panic(errors.New("failed to cast dependency to expected type", DependencyTypeMismatchErrorCode))
	}

	return typedInstance, nil
}

// createSingleConfigurationWithToken is an internal function that creates a configuration instance.
// It creates a single configuration of type CT using the provided context and registry options.
// Returns the created configuration instance and any error that occurred.
func createSingleConfigurationWithToken[CT any](ctx Context, opts RegistryOpts) (CT, error) {
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

	tType := TypeName[CT](token)
	unknownInstance, err = f.CreateConfiguration(ctx, tType, opts)
	_, isMissing := errors.Has(err, DependencyMissingErrorCode)
	if err != nil && (!isMissing || len(token) == 0) {
		return typedInstance, errors.Wrap(err, "failed to create configuration dependency first try", ErrorCreatingDependencyErrorCode)
	}

	if isMissing {
		tType = TypeName[CT]() //trying creation without token
		unknownInstance, err = f.CreateConfiguration(ctx, tType, opts)
		if err != nil {
			return typedInstance, errors.Wrap(err, "failed to create configuration dependency second try", ErrorCreatingDependencyErrorCode)
		}
	}

	typedInstance, ok = safeTypeAssert[CT](unknownInstance)
	if !ok {
		panic(errors.New("failed to cast dependency to expected type", DependencyTypeMismatchErrorCode))
	}

	return typedInstance, nil
}
