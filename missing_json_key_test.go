package di

import (
	goctx "context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// reflectBackedConfig implements Configuration using struct reflection via ConfigurationNodeLookup,
// which is what the production code does — it finds zero-value struct fields regardless of
// whether the corresponding key exists in the original JSON.
type reflectBackedConfig struct {
	data any
}

func (r reflectBackedConfig) LookupNode(path string) (any, error) {
	return ConfigurationNodeLookup(r.data, path)
}

type missingKeyDataLayerConfig struct {
	DSN string `json:"dsn"`
}

// businessLayerWithMissingKey has two service configs as struct fields.
// In the production bug: the Go struct had both fields but the JSON only had one key,
// so the missing key's config was silently zero-valued.
type businessLayerWithMissingKey struct {
	ServiceAConfig missingKeyDataLayerConfig `json:"service_a"`
	ServiceBConfig missingKeyDataLayerConfig `json:"service_b"`
}

// TestConfigurationLookup_MissingJSONKeyReturnsError reproduces the production bug where
// a service config key is absent from the JSON file but exists as a Go struct field.
// Before the fix: ConfigurationNodeLookup used struct reflection to find the field and
// returned a zero-value config — no error, broken service created, panic at first use.
// After the fix: the raw JSON map is checked first; absent key returns an error at bootstrap.
func TestConfigurationLookup_MissingJSONKeyReturnsError(t *testing.T) {
	// Raw map mirrors the JSON file: service_a is present, service_b is intentionally absent.
	raw := ConfigRawData{
		"service_a": map[string]interface{}{"dsn": "postgres://localhost/db"},
	}

	// The Go struct has both fields. ServiceBConfig is zero-valued (same as production:
	// mapstructure decodes the JSON into the struct and the missing key becomes zero value).
	cfg := reflectBackedConfig{data: businessLayerWithMissingKey{
		ServiceAConfig: missingKeyDataLayerConfig{DSN: "postgres://localhost/db"},
	}}

	// Construct context directly so rawCfg (sparse JSON map) and cfg (full struct) are
	// independent — NewContext would overwrite rawCfg from cfg via Decode, masking the bug.
	ctx := &context{
		ctx:    goctx.Background(),
		rawCfg: raw,
		cfg:    cfg,
	}

	t.Run("present key succeeds", func(t *testing.T) {
		result, err := ConfigurationLookup[missingKeyDataLayerConfig](ctx, &RegistryOpts{ConfigNodePath: "service_a"})
		require.NoError(t, err)
		assert.Equal(t, "postgres://localhost/db", result.DSN)
	})

	t.Run("absent key returns error", func(t *testing.T) {
		_, err := ConfigurationLookup[missingKeyDataLayerConfig](ctx, &RegistryOpts{ConfigNodePath: "service_b"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service_b")
	})
}
