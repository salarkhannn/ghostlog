package secrets

import (
	"regexp"
	"strings"
)

type Rule struct {
	Name    string
	Pattern *regexp.Regexp
}

var rules = []Rule{
	{
		Name:    "PEM Private Key",
		Pattern: regexp.MustCompile(`(?i)-----BEGIN[ A-Z0-9_-]*PRIVATE KEY-----`),
	},
	{
		Name:    "AWS Access Key ID",
		Pattern: regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
	},
	{
		Name:    "Generic Assignment Secret",
		Pattern: regexp.MustCompile(`(?i)\b(db_password|password|pass|passwd|pwd|secret|client_secret|token|auth_token|api_key|apikey|private_key|secret_key)\b\s*(:=|=)\s*["']?([a-zA-Z0-9_\-\.\~\+\/]{16,})["']?`),
	},
}

var placeholders = []string{
	"your_", "placeholder", "env.", "config.", "os.getenv", "dummy", "mock", "test", "example", "replace_me",
}

func ScanLine(line string) (string, bool) {
	for _, rule := range rules {
		if rule.Pattern.MatchString(line) {
			if rule.Name == "Generic Assignment Secret" {
				lowered := strings.ToLower(line)
				hasPlaceholder := false
				for _, pl := range placeholders {
					if strings.Contains(lowered, pl) {
						hasPlaceholder = true
						break
					}
				}
				if hasPlaceholder {
					continue
				}
			}
			return rule.Name, true
		}
	}
	return "", false
}
