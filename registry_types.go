package di

import (
	"fmt"
	"reflect"
)

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

type RegistryOpts struct {
	Registry       Registry
	InjectionToken InjectionToken
}

func WithOpts(opt RegistryOpts) func(opts *RegistryOpts) {
	return func(opts *RegistryOpts) {
		*opts = opt
	}
}

func WithRegistry(instance Registry) func(opts *RegistryOpts) {
	return func(opts *RegistryOpts) {
		opts.Registry = instance
	}
}

func WithInjectionToken(token InjectionToken) func(opts *RegistryOpts) {
	return func(opts *RegistryOpts) {
		opts.InjectionToken = token
	}
}

type NoConfig struct{}
type InjectionToken string

type CreateInstanceHandler func(Ctx, RegistryOpts, any) (any, error)
type CreateConfigurationHandler func(Ctx, RegistryOpts) (any, error)
type TypedCreateInstanceHandler[T any, CT any] func(Ctx, RegistryOpts, CT) (T, error)
type TypedCreateInstanceNoConfigHandler[T any] func(Ctx, RegistryOpts) (T, error)

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