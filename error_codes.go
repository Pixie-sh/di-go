package di

import "github.com/pixie-sh/errors-go"

var (
	DIErrorCodeBase = 75000
	ErrorCreatingDependencyErrorCode = errors.NewErrorCode("ErrorCreatingDependencyErrorCode", DIErrorCodeBase+ 503)
	DependencyMissingErrorCode = errors.NewErrorCode("DependencyMissingErrorCode", DIErrorCodeBase+ 503)
	DependencyTypeMismatchErrorCode = errors.NewErrorCode("DependencyTypeMismatchErrorCode", DIErrorCodeBase+ 503)
	StructMapTypeMismatchErrorCode = errors.NewErrorCode("StructMapTypeMismatchErrorCode", DIErrorCodeBase+ 503)
)