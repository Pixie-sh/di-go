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
	Factory        Registry
	InjectionToken InjectionToken
}

func WithOpts(opt RegistryOpts) func(opts *RegistryOpts) {
	return func(opts *RegistryOpts) {
		*opts = opt
	}
}

func WithFactory(instance Registry) func(opts *RegistryOpts) {
	return func(opts *RegistryOpts) {
		opts.Factory = instance
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
