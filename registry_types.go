package di

import (
	"fmt"
	"github.com/pixie-sh/errors-go"
	"reflect"
)

var injectionTokenMap = map[InjectionToken]struct{}{}

type NoConfig struct{}
type InjectionToken string

func (t InjectionToken) String() string {
	return string(t)
}

const injectionTokenSeparator = "."

// RegisterInjectionToken creates and returns a new InjectionToken from the provided string.
// This function is used to create typed tokens for dependency injection registration and resolution.
// The token string must not:
// - Be empty
// - Start or end with a dot
// - Contain consecutive dots
func RegisterInjectionToken(tkn string) InjectionToken {
	_, existing := injectionTokenMap[InjectionToken(tkn)]
	if existing {
		errors.Must(errors.New("injection token %s already registered", tkn))
	}

	if tkn == "" {
		errors.Must(errors.New("injection token cannot be empty"))
	}

	for i, r := range tkn {
		if r == '.' {
			if i == 0 || i == len(tkn)-1 {
				errors.Must(errors.New("injection token %s cannot start or end with a dot", tkn))
			}

			if tkn[i-1] == '.' {
				errors.Must(errors.New("injection token %s cannot contain consecutive dots", tkn))
			}
		}
	}

	injectionTokenMap[InjectionToken(tkn)] = struct{}{}
	return InjectionToken(tkn)
}


func TypeName[T any](tokens ...InjectionToken) string {
	var typeName string
	var t *T

	typeOfT := reflect.TypeOf(t).Elem()
	if typeOfT.Kind() == reflect.Ptr {
		typeName = typeOfT.Elem().String()
	} else {
		typeName = typeOfT.String()
	}

	if len(tokens) > 0 && len(tokens[0]) > 0 {
		return fmt.Sprintf("%s:%s", tokens[0], typeName)
	}

	return typeName
}

func PairTypeName(first, second string) string {
	return fmt.Sprintf("%s;%s", first, second)
}

// RegistryOpts defines the configuration options for dependency injection registry operations.
// It contains the registry instance to use, an optional injection token for type identification,
// and a configuration node path for structured configuration handling.
//
// InjectionToken + ConfigNode should return the correct go struct extracted form
type RegistryOpts struct {
	Registry       Registry       // The registry instance to use for dependency management
	InjectionToken InjectionToken // Optional token to identify specific type registrations
	ConfigNode     string         // Path to configuration node in structured config
}

// WithOpts returns a function that replaces all registry options with the provided options.
// This is useful when you want to completely override the default options with a new set.
func WithOpts(opt *RegistryOpts) func(opts *RegistryOpts) {
	return func(opts *RegistryOpts) {
		*opts = *opt
	}
}

// WithRegistry returns a function that sets the registry instance in the options.
// This allows specifying which registry should be used for dependency management.
func WithRegistry(instance Registry) func(opts *RegistryOpts) {
	return func(opts *RegistryOpts) {
		opts.Registry = instance
	}
}

// WithToken returns a function that sets the injection token in the options.
// This enables type-specific registration and resolution in the dependency injection system.
func WithToken(token InjectionToken) func(opts *RegistryOpts) {
	return func(opts *RegistryOpts) {
		opts.InjectionToken = token
	}
}

// WithConfigNode returns a function that sets the configuration node path in the options.
// This allows specifying which configuration path should be used for dependency management.
func WithConfigNode(path string) func(opts *RegistryOpts) {
	return func(opts *RegistryOpts) {
		if len(opts.ConfigNode) > 0 {
			opts.ConfigNode = opts.ConfigNode + "." + path
			return
		}

		opts.ConfigNode = path
	}
}

type CreateInstanceHandler func(Context, *RegistryOpts, any) (any, error)
type CreateConfigurationHandler func(Context, *RegistryOpts) (any, error)
type TypedCreateInstanceHandler[T any, CT any] func(Context, *RegistryOpts, CT) (T, error)
type TypedCreateInstanceNoConfigHandler[T any] func(Context, *RegistryOpts) (T, error)

// Here's the implementation of the fix in a function that can be added to your codebase
// This would need to replace or augment the existing type assertion in createSingleWithToken
func safeTypeAssert[T any](unknownInstance any) (T, bool) {
	var typedInstance T

	// Try direct type assertion first
	typedInstance, ok := unknownInstance.(T)
	if ok {
		return typedInstance, true
	}

	// Get the type information
	targetType := reflect.TypeOf((*T)(nil)).Elem()
	sourceType := reflect.TypeOf(unknownInstance)

	// If both are nil, we can't do much
	if sourceType == nil {
		return typedInstance, false
	}

	// Check if source is pointer but target is not
	if sourceType.Kind() == reflect.Ptr && targetType.Kind() != reflect.Ptr {
		// If source is *X and target is X, dereference the pointer
		if sourceType.Elem() == targetType {
			elemValue := reflect.ValueOf(unknownInstance).Elem().Interface()
			typedInstance, ok = elemValue.(T)
			return typedInstance, ok
		}
	}

	// Check if target is pointer but source is not
	if targetType.Kind() == reflect.Ptr && sourceType.Kind() != reflect.Ptr {
		// If target is *X and source is X, get a pointer to the value
		if targetType.Elem() == sourceType {
			// Create a new pointer to source type
			ptrValue := reflect.New(sourceType)
			// Set the pointer's value to our source
			ptrValue.Elem().Set(reflect.ValueOf(unknownInstance))
			// Try the cast
			typedInstance, ok = ptrValue.Interface().(T)
			return typedInstance, ok
		}
	}

	return typedInstance, false
}
