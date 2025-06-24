package di

import (
	"fmt"
	gojson "github.com/goccy/go-json"
	"github.com/pixie-sh/errors-go"
	"reflect"
	"regexp"
	"strings"
)

func ConfigurationLookup[T any](ctx Context, opts *RegistryOpts) (T, error) {
	var result T

	if ctx == nil {
		return result, errors.New("di.Context cannot be nil", ConfigurationLookupErrorCode)
	}

	if ctx.Configuration() == nil {
		return result, errors.New("di.Context.Configuration() cannot be nil", ConfigurationLookupErrorCode)
	}

	lookupPath, err := assembleConfigurationLookupPath(ctx, opts)
	if err != nil {
		return result, errors.Wrap(err, "assembleConfigurationLookupPath error", ConfigurationLookupErrorCode)
	}

	abstractNode, err := ctx.Configuration().LookupNode(lookupPath)
	if err != nil || abstractNode == nil {
		return result, errors.Wrap(err, "di.Context.Configuration().LookupNode() failed", ConfigurationLookupErrorCode)
	}

	typed, good := SafeTypeAssert[T](abstractNode)
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

func assembleConfigurationLookupPath(_ Context, opts *RegistryOpts) (string, error) {
	if len(opts.InjectionToken) == 0 && len(opts.ConfigNodePath) == 0{
		return "", errors.New("di.*RegistryOpts.InjectionToken and di.*RegistryOpts.ConfigNodePath cannot be both empty", ConfigurationLookupErrorCode)
	}

	lp := opts.ConfigNodePath
	if len(opts.InjectionToken) > 0 {
		lp = opts.InjectionToken.String() + "." + opts.ConfigNodePath
	}

	if len(lp) == 0 {
		return "", errors.New("lookup path cannot be empty", ConfigurationLookupErrorCode)
	}

	return lp, nil
}

// ResolveDIReferences processes a JSON string and replaces "${di.XXXXX}" references
// with the actual JSON nodes they point to. This function can be used independently
// of any specific struct type.
func ResolveDIReferences(jsonStr string) (string, error) {
	// Regular expression to match both quoted and unquoted ${di.path.to.node} patterns
	// This will match: "session_cache": ${di.singleton} or "session_cache": "${di.singleton}"
	re := regexp.MustCompile(`["']?(\$\{di\.([^}]+)\})["']?`)

	// First, we need to make the JSON valid by quoting unquoted DI references
	validJSON := makeJSONValid(jsonStr)

	// Parse the JSON to get the base structure
	var rawData map[string]interface{}
	tempJSON := re.ReplaceAllString(validJSON, `null`)
	if err := gojson.Unmarshal([]byte(tempJSON), &rawData); err != nil {
		return "", fmt.Errorf("failed to parse JSON for DI resolution: %w", err)
	}

	// Find all DI references (both quoted and unquoted)
	matches := re.FindAllStringSubmatch(jsonStr, -1)
	replacements := make(map[string]string)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		fullMatch := match[1] // ${di.singleton}
		diPath := match[2]    // singleton

		// Skip if we already processed this reference
		if _, exists := replacements[fullMatch]; exists {
			continue
		}

		// Extract the referenced node from the raw data
		referencedNode, err := ExtractNodeFromJSONPath(rawData, diPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve DI reference %s: %w", fullMatch, err)
		}

		// Convert the referenced node back to JSON
		nodeJSON, err := gojson.Marshal(referencedNode)
		if err != nil {
			return "", fmt.Errorf("failed to marshal referenced node %s: %w", fullMatch, err)
		}

		replacements[fullMatch] = string(nodeJSON)
	}

	// Apply all replacements to the valid JSON
	result := validJSON
	for placeholder, replacement := range replacements {
		// Replace both quoted and unquoted versions
		result = strings.ReplaceAll(result, `"`+placeholder+`"`, replacement)
		result = strings.ReplaceAll(result, placeholder, replacement)
	}

	return result, nil
}

// makeJSONValid converts unquoted DI references to quoted strings to make valid JSON
func makeJSONValid(jsonStr string) string {
	// Regular expression to find unquoted ${di.xxx} patterns
	re := regexp.MustCompile(`:\s*(\$\{di\.[^}]+\})([,\s\}])`)

	// Replace unquoted DI references with quoted versions
	result := re.ReplaceAllString(jsonStr, `: "$1"$2`)

	return result
}

// ExtractNodeFromJSONPath navigates through a map[string]interface{} structure
// to find the node at the given dot-separated path.
func ExtractNodeFromJSONPath(data map[string]interface{}, path string) (interface{}, error) {
	if path == "" {
		return data, nil
	}

	parts := strings.Split(path, ".")
	current := data

	for i, part := range parts {
		value, exists := current[part]
		if !exists {
			return nil, fmt.Errorf("path component '%s' not found in path '%s'", part, path)
		}

		// If this is the last part, return the value
		if i == len(parts)-1 {
			return value, nil
		}

		// Otherwise, ensure the value is a map for the next iteration
		nextMap, ok := value.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("path component '%s' is not an object, cannot navigate further in path '%s'", part, path)
		}

		current = nextMap
	}

	return current, nil
}

// UnmarshalJSONWithDIResolution is a helper function that can be used by any struct
// to unmarshal JSON with DI reference resolution. It takes the raw JSON bytes,
// resolves DI references, and unmarshals into the provided destination.
func UnmarshalJSONWithDIResolution(data []byte, dest interface{}) error {
	// Resolve DI references in the JSON string
	resolvedJSON, err := ResolveDIReferences(string(data))
	if err != nil {
		return fmt.Errorf("failed to resolve DI references: %w", err)
	}

	// Unmarshal the resolved JSON into the destination
	if err := gojson.Unmarshal([]byte(resolvedJSON), dest); err != nil {
		return fmt.Errorf("failed to unmarshal resolved JSON: %w", err)
	}

	return nil
}

// FindDIReferences scans a JSON string and returns all DI references found.
// This can be useful for validation or preprocessing.
func FindDIReferences(jsonStr string) []string {
	re := regexp.MustCompile(`\$\{di\.([^}]+)\}`)
	matches := re.FindAllString(jsonStr, -1)

	// Remove duplicates
	seen := make(map[string]bool)
	var unique []string
	for _, match := range matches {
		if !seen[match] {
			seen[match] = true
			unique = append(unique, match)
		}
	}

	return unique
}

// ValidateDIReferences checks if all DI references in a JSON string can be resolved
// against the provided data structure. Returns an error if any reference is invalid.
func ValidateDIReferences(jsonStr string, data map[string]interface{}) error {
	re := regexp.MustCompile(`\$\{di\.([^}]+)\}`)
	matches := re.FindAllStringSubmatch(jsonStr, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		diPath := match[1] // singleton or singleton.cache
		_, err := ExtractNodeFromJSONPath(data, diPath)
		if err != nil {
			return fmt.Errorf("invalid DI reference ${di.%s}: %w", diPath, err)
		}
	}

	return nil
}