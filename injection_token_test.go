package di

import (
	"github.com/pixie-sh/errors-go"
	"strings"
	"testing"
)

func TestRegisterInjectionToken(t *testing.T) {
	// Reset the global map before tests
	injectionTokenMap = map[InjectionToken]struct{}{}

	tests := []struct {
		name        string
		tokenString string
		wantError   bool
		errorMsg    string
	}{
		{
			name:        "Valid token",
			tokenString: "auth.service",
			wantError:   false,
		},
		{
			name:        "Valid token with multiple segments",
			tokenString: "auth.service.user",
			wantError:   false,
		},
		{
			name:        "Valid token without dots",
			tokenString: "authservice",
			wantError:   false,
		},
		{
			name:        "Empty token",
			tokenString: "",
			wantError:   true,
			errorMsg:    "injection token cannot be empty",
		},
		{
			name:        "Token starting with dot",
			tokenString: ".auth.service",
			wantError:   true,
			errorMsg:    "injection token .auth.service cannot start or end with a dot",
		},
		{
			name:        "Token ending with dot",
			tokenString: "auth.service.",
			wantError:   true,
			errorMsg:    "injection token auth.service. cannot start or end with a dot",
		},
		{
			name:        "Token with consecutive dots",
			tokenString: "auth..service",
			wantError:   true,
			errorMsg:    "injection token auth..service cannot contain consecutive dots",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var token InjectionToken
			var err error

			// Use a recovery function to capture panics from errors.Must()
			func() {
				defer func() {
					if r := recover(); r != nil {
						var errStr string
						// Convert the panic value to an error
						switch v := r.(type) {
						case error:
							errStr = v.Error()
							err = v
						case errors.E:
							errStr = v.Error()
							err = v
						case string:
							errStr = v
							e := errors.New(v)
							_ = e.UnmarshalJSON([]byte(v))
							err = e
						}

						if !tt.wantError {
							t.Errorf("RegisterInjectionToken() panicked unexpectedly: %v", errStr)
						} else if !strings.Contains(errStr, tt.errorMsg) {
							t.Errorf("RegisterInjectionToken() error = %v, wantErr containing %v", errStr, tt.errorMsg)
						}
					}
				}()

				token = RegisterInjectionToken(tt.tokenString)
			}()

			if err == nil {
				if tt.wantError {
					t.Errorf("RegisterInjectionToken() error = nil, wantErr %v", tt.wantError)
					return
				}

				// Check the token was registered correctly
				if string(token) != tt.tokenString {
					t.Errorf("RegisterInjectionToken() token = %v, want %v", token, tt.tokenString)
				}

				// Check it was added to the map
				if _, exists := injectionTokenMap[token]; !exists {
					t.Errorf("RegisterInjectionToken() did not register token in map")
				}
			}
		})
	}

	// Test for duplicate registration
	t.Run("Duplicate token registration", func(t *testing.T) {
		// Reset the map
		injectionTokenMap = map[InjectionToken]struct{}{}

		// Register a token first
		tokenStr := "auth.service"
		_ = RegisterInjectionToken(tokenStr)

		// Try to register it again and expect failure
		var err error
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = r.(error)
				}
			}()
			RegisterInjectionToken(tokenStr)
		}()

		if err == nil {
			t.Errorf("RegisterInjectionToken() did not fail when registering duplicate token")
		} else if !strings.Contains(err.Error(), "injection token auth.service already registered") {
			t.Errorf("RegisterInjectionToken() error = %v, expected error about already registered token", err.Error())
		}
	})
}
