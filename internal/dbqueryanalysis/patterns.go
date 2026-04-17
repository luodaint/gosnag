package dbqueryanalysis

import "regexp"

var (
	basicAuthDSNPattern = regexp.MustCompile(`://[^/\s:@]+:[^/\s@]+@`)
	passwordPairPattern = regexp.MustCompile(`(?i)\b(password|pwd)\s*=\s*[^ \t\n\r;]+`)
	quotedLiteralRegex  = regexp.MustCompile(`'(?:''|[^'])*'`)
	numericLiteralRegex = regexp.MustCompile(`(?i)\b-?\d+(?:\.\d+)?\b`)
	whitespaceRegex     = regexp.MustCompile(`\s+`)
)
