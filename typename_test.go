package di

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTypes are used to verify the TypeName and PackageName functions
type TestStruct struct{}
type TestNestedStruct struct {
	Field string
}

func TestTypeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		typeFunc func() string
		slug     InjectionToken
		expected string
	}{
		{
			name: "Basic type without slug",
			typeFunc: func() string {
				return TypeName[TestStruct]()
			},
			expected: "di.TestStruct",
		},
		{
			name: "Basic type with slug",
			typeFunc: func() string {
				return TypeName[TestStruct]("test-slug")
			},
			expected: "test-slug:di.TestStruct",
		},
		{
			name: "Pointer type without slug",
			typeFunc: func() string {
				return TypeName[*TestStruct]()
			},
			expected: "di.TestStruct",
		},
		{
			name: "Nested type without slug",
			typeFunc: func() string {
				return TypeName[TestNestedStruct]()
			},
			expected: "di.TestNestedStruct",
		},
		{
			name: "Built-in type without slug",
			typeFunc: func() string {
				return TypeName[string]()
			},
			expected: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.typeFunc()
			assert.Equal(t, tt.expected, result)
		})
	}
}
