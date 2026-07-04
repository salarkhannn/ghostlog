package secrets

import (
	"testing"
)

func TestScanLine(t *testing.T) {
	tests := []struct {
		line     string
		expected string
		matched  bool
	}{
		{"-----BEGIN RSA PRIVATE KEY-----", "PEM Private Key", true},
		{"-----BEGIN EC PRIVATE KEY-----", "PEM Private Key", true},
		{"const AWS_KEY = \"AKIAIOSFODNN7EXAMPLE\"", "AWS Access Key ID", true},
		{"secret_key = \"a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6\"", "Generic Assignment Secret", true},
		{"db_password := \"super_secure_password_12345\"", "Generic Assignment Secret", true},
		{"password = \"YOUR_PASSWORD\"", "", false},
		{"api_key = os.Getenv(\"API_KEY\")", "", false},
		{"dummy_secret = \"placeholder_token\"", "", false},
		{"fmt.Println(\"Hello World\")", "", false},
	}

	for _, tc := range tests {
		name, ok := ScanLine(tc.line)
		if ok != tc.matched {
			t.Errorf("ScanLine(%q) matched = %t, expected %t", tc.line, ok, tc.matched)
		}
		if name != tc.expected {
			t.Errorf("ScanLine(%q) name = %q, expected %q", tc.line, name, tc.expected)
		}
	}
}
