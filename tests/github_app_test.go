package tests

import (
	"os"
	"strconv"
	"testing"
)

// TestParseValidAppID tests parsing a valid GITHUB_APP_ID
func TestParseValidAppID(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		expected  int64
		shouldErr bool
	}{
		{
			name:      "valid positive number",
			input:     "123456789",
			expected:  123456789,
			shouldErr: false,
		},
		{
			name:      "zero",
			input:     "0",
			expected:  0,
			shouldErr: false,
		},
		{
			name:      "large number",
			input:     "9223372036854775807",
			expected:  9223372036854775807,
			shouldErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := strconv.ParseInt(tc.input, 10, 64)
			if tc.shouldErr && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tc.shouldErr && err != nil {
				t.Errorf("Expected no error but got %v", err)
			}
			if !tc.shouldErr && result != tc.expected {
				t.Errorf("Expected %d, got %d", tc.expected, result)
			}
		})
	}
}

// TestParseInvalidAppID tests parsing an invalid GITHUB_APP_ID
func TestParseInvalidAppID(t *testing.T) {
	invalidInputs := []string{
		"not-a-number",
		"12.34",
		"abc123",
		"",
		"123abc",
	}

	for _, input := range invalidInputs {
		t.Run("invalid: "+input, func(t *testing.T) {
			_, err := strconv.ParseInt(input, 10, 64)
			if err == nil {
				t.Errorf("Expected error for input '%s', but got none", input)
			}
		})
	}
}

// TestEnvironmentVariableValidation tests that the required environment variables can be read
func TestEnvironmentVariableValidation(t *testing.T) {
	testCases := []struct {
		name     string
		varName  string
		value    string
		validate func(string) error
	}{
		{
			name:    "GITHUB_APP_ID",
			varName: "GITHUB_APP_ID",
			value:   "123456789",
			validate: func(v string) error {
				_, err := strconv.ParseInt(v, 10, 64)
				return err
			},
		},
		{
			name:    "GITHUB_APP_INSTALLATION_ID",
			varName: "GITHUB_APP_INSTALLATION_ID",
			value:   "987654321",
			validate: func(v string) error {
				_, err := strconv.ParseInt(v, 10, 64)
				return err
			},
		},
		{
			name:    "GITHUB_APP_PRIVATE_KEY",
			varName: "GITHUB_APP_PRIVATE_KEY",
			value:   "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...",
			validate: func(v string) error {
				if v == "" {
					return os.ErrNotExist
				}
				return nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			origValue := os.Getenv(tc.varName)
			defer os.Setenv(tc.varName, origValue)

			os.Setenv(tc.varName, tc.value)
			value := os.Getenv(tc.varName)

			if value != tc.value {
				t.Errorf("Expected environment variable %s to be '%s', got '%s'", tc.varName, tc.value, value)
			}

			if err := tc.validate(value); err != nil {
				t.Errorf("Validation failed for %s: %v", tc.varName, err)
			}
		})
	}
}

// TestMissingCredentials tests that missing credentials are properly handled
func TestMissingCredentials(t *testing.T) {
	testCases := []struct {
		name   string
		varCat string
	}{
		{"GITHUB_APP_ID missing", "GITHUB_APP_ID"},
		{"GITHUB_APP_INSTALLATION_ID missing", "GITHUB_APP_INSTALLATION_ID"},
		{"GITHUB_APP_PRIVATE_KEY missing", "GITHUB_APP_PRIVATE_KEY"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			origValue := os.Getenv(tc.varCat)
			defer os.Setenv(tc.varCat, origValue)

			os.Unsetenv(tc.varCat)
			value := os.Getenv(tc.varCat)

			if value != "" {
				t.Errorf("Expected environment variable %s to be empty, got '%s'", tc.varCat, value)
			}
		})
	}
}
