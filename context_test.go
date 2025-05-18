package di

import (
	goctx "context"
	"reflect"
	"testing"
	"time"
)

// Simple map that implements ConfigData for testing
type SimpleConfig map[string]interface{}

func TestContext_NoArgs(t *testing.T) {
	// Test Context() with no arguments
	ctx := Context()

	// Should use background context internally
	if ctx.Inner() == nil {
		t.Error("Expected background context, got nil")
	}

	// Check configuration properties
	if ctx.RawConfiguration() == nil {
		t.Error("Expected empty raw config map, got nil")
	}

	if len(ctx.RawConfiguration()) != 0 {
		t.Errorf("Expected empty raw config, got %v", ctx.RawConfiguration())
	}

	if ctx.Configuration() != nil {
		t.Errorf("Expected nil config, got %v", ctx.Configuration())
	}
}

func TestContext_WithStdContext(t *testing.T) {
	// Test Context with standard context
	stdCtx := goctx.WithValue(goctx.Background(), "key", "value")
	ctx := Context(stdCtx)

	// Verify the inner context is preserved
	if ctx.Value("key") != "value" {
		t.Errorf("Expected context value 'value', got %v", ctx.Value("key"))
	}

	if ctx.Inner() != stdCtx {
		t.Error("Inner context doesn't match provided context")
	}
}

func TestContext_WithRawConfig(t *testing.T) {
	// Test Context with raw configuration
	rawCfg := ConfigRawData{
		"name":  "test",
		"value": 123,
	}

	ctx := Context(rawCfg)

	// Verify configuration is preserved
	if !reflect.DeepEqual(ctx.RawConfiguration(), rawCfg) {
		t.Errorf("Expected raw config %v, got %v", rawCfg, ctx.RawConfiguration())
	}

	// Config should be nil since we only provided raw config
	if ctx.Configuration() != nil {
		t.Errorf("Expected nil config, got %v", ctx.Configuration())
	}
}

func TestContext_WithConfig(t *testing.T) {
	// Use a simple map as config to avoid decoder complexity
	cfg := SimpleConfig{
		"Name":  "test",
		"Value": 123,
	}

	ctx := Context(cfg)

	// Check that configuration was properly stored
	if ctx.Configuration() == nil {
		t.Error("Expected config to be stored, got nil")
	}

	// Raw configuration should contain the same data
	// But we can't directly compare the values because the Decode function
	// might transform them in implementation-specific ways
	if ctx.RawConfiguration() == nil {
		t.Error("Expected raw config to be populated, got nil")
	}

	// Instead, we can check that the context has our config object stored
	if !reflect.DeepEqual(ctx.Configuration(), cfg) {
		t.Errorf("Expected config %v, got %v", cfg, ctx.Configuration())
	}
}

func TestContext_WithParentCtx(t *testing.T) {
	// Create a parent context
	parentRawCfg := ConfigRawData{"parent": true}
	parentCtx := Context(parentRawCfg)

	// Create child context with parent
	ctx := Context(parentCtx)

	// Should inherit parent's config
	if ctx.RawConfiguration()["parent"] != true {
		t.Errorf("Expected to inherit parent config, got %v", ctx.RawConfiguration())
	}
}

func TestContext_WithParentAndOverrides(t *testing.T) {
	// Create a parent context with config
	parentRawCfg := ConfigRawData{"parent": true, "shared": "parent"}
	parentCtx := Context(parentRawCfg)

	// Create child with overridden config
	childRawCfg := ConfigRawData{"child": true, "shared": "child"}
	ctx := Context(parentCtx, childRawCfg)

	// Verify overrides worked
	if ctx.RawConfiguration()["parent"] != nil {
		t.Error("Expected parent's config to be completely overridden")
	}

	if ctx.RawConfiguration()["child"] != true {
		t.Error("Expected child's config to be present")
	}

	if ctx.RawConfiguration()["shared"] != "child" {
		t.Errorf("Expected shared value to be overridden to 'child', got %v", ctx.RawConfiguration()["shared"])
	}
}

func TestContext_WithStdContextAndConfig(t *testing.T) {
	// Test with both standard context and config
	stdCtx := goctx.WithValue(goctx.Background(), "key", "value")
	rawCfg := ConfigRawData{"name": "test"}

	ctx := Context(stdCtx, rawCfg)

	// Verify both were applied
	if ctx.Value("key") != "value" {
		t.Errorf("Expected context value 'value', got %v", ctx.Value("key"))
	}

	if ctx.RawConfiguration()["name"] != "test" {
		t.Errorf("Expected raw config with name 'test', got %v", ctx.RawConfiguration())
	}
}

func TestContext_WithMultipleTypes(t *testing.T) {
	// Test with all types of arguments
	stdCtx := goctx.WithValue(goctx.Background(), "key", "value")

	// Use a simple map as config instead of a struct
	cfg := SimpleConfig{"Name": "typed"}

	// Create a parent context
	parentRawCfg := ConfigRawData{"parent": true}
	parentCtx := Context(parentRawCfg)

	// Create context with all types
	ctx := Context(parentCtx, stdCtx, cfg)

	// Verify all arguments were correctly processed
	if ctx.Value("key") != "value" {
		t.Errorf("Expected context value 'value', got %v", ctx.Value("key"))
	}

	// Since we're using a map, we can reasonably expect the Name field to be preserved
	if val, exists := ctx.RawConfiguration()["Name"]; !exists || val != "typed" {
		t.Errorf("Expected raw config with Name 'typed', got %v", ctx.RawConfiguration())
	}

	if ctx.Inner() != stdCtx {
		t.Error("Inner context doesn't match provided context")
	}
}

func TestContext_PrimitiveConfig(t *testing.T) {
	// Test with a primitive value as config
	// This tests the Decode function indirectly
	cfg := struct {
		Val int
	} {
		Val: 42,
	}

	ctx := Context(cfg)

	// The raw configuration should have a "value" field with 42
	// assuming the Decode function works as expected
	if ctx.RawConfiguration() == nil {
		t.Error("Expected raw config to be populated, got nil")
	}

	// The original config should be preserved
	if !reflect.DeepEqual(ctx.Configuration(), cfg) {
		t.Errorf("Expected config %v, got %v", cfg, ctx.Configuration())
	}
}

func TestContext_StructConfig(t *testing.T) {
	// Define a simple struct for configuration
	type TestConfig struct {
		Name  string
		Value int
	}

	// Create an instance
	cfg := TestConfig{
		Name:  "test",
		Value: 42,
	}

	// Create context with the struct config
	ctx := Context(cfg)

	// Check that the original struct is preserved as Configuration
	if !reflect.DeepEqual(ctx.Configuration(), cfg) {
		t.Errorf("Expected config %v, got %v", cfg, ctx.Configuration())
	}

	// We can't make specific assertions about RawConfiguration
	// without knowing the implementation of Decode
	if ctx.RawConfiguration() == nil {
		t.Error("Expected raw config to be populated, got nil")
	}
}

func TestContext_Deadline(t *testing.T) {
	// Test deadline is properly passed through
	deadline := time.Now().Add(5 * time.Second)
	stdCtx, cancel := goctx.WithDeadline(goctx.Background(), deadline)
	defer cancel()

	ctx := Context(stdCtx)

	gotDeadline, ok := ctx.Deadline()
	if !ok {
		t.Error("Expected deadline to be set")
	}

	if !gotDeadline.Equal(deadline) {
		t.Errorf("Expected deadline %v, got %v", deadline, gotDeadline)
	}
}

func TestContext_Done(t *testing.T) {
	// Test Done channel is properly passed through
	stdCtx, cancel := goctx.WithCancel(goctx.Background())

	ctx := Context(stdCtx)

	// Verify Done channel exists
	if ctx.Done() == nil {
		t.Error("Expected Done channel, got nil")
	}

	// Cancel and check if Done channel is closed
	cancel()
	select {
	case <-ctx.Done():
		// This is expected
	default:
		t.Error("Expected Done channel to be closed after cancel")
	}
}

func TestContext_Err(t *testing.T) {
	// Test Err is properly passed through
	stdCtx, cancel := goctx.WithCancel(goctx.Background())

	ctx := Context(stdCtx)

	// Before cancel, Err should be nil
	if ctx.Err() != nil {
		t.Errorf("Expected nil error before cancel, got %v", ctx.Err())
	}

	// After cancel, Err should be goctx.Canceled
	cancel()
	if ctx.Err() != goctx.Canceled {
		t.Errorf("Expected goctx.Canceled error after cancel, got %v", ctx.Err())
	}
}

func TestContext_Value(t *testing.T) {
	// Test Value is properly passed through
	stdCtx := goctx.WithValue(goctx.Background(), "key1", "value1")
	stdCtx = goctx.WithValue(stdCtx, "key2", 123)

	ctx := Context(stdCtx)

	// Verify values can be retrieved
	if ctx.Value("key1") != "value1" {
		t.Errorf("Expected value 'value1', got %v", ctx.Value("key1"))
	}

	if ctx.Value("key2") != 123 {
		t.Errorf("Expected value 123, got %v", ctx.Value("key2"))
	}

	// Non-existent key should return nil
	if ctx.Value("nonexistent") != nil {
		t.Errorf("Expected nil for nonexistent key, got %v", ctx.Value("nonexistent"))
	}
}