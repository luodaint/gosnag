package conditions

import (
	"encoding/json"
	"regexp"
	"strings"
)

type StacktraceRules struct {
	Preset            string   `json:"preset"`
	AppPatterns       []string `json:"app_patterns"`
	FrameworkPatterns []string `json:"framework_patterns"`
	ExternalPatterns  []string `json:"external_patterns"`
}

type stacktraceFrame struct {
	Filename string `json:"filename"`
	InApp    *bool  `json:"in_app"`
}

type stacktraceEvent struct {
	Exception struct {
		Values []struct {
			Stacktrace struct {
				Frames []stacktraceFrame `json:"frames"`
			} `json:"stacktrace"`
		} `json:"values"`
	} `json:"exception"`
}

// HasAppFrame returns true when the latest event contains at least one frame
// that should be treated as application code for this project's rules.
func HasAppFrame(eventData, rulesRaw json.RawMessage) bool {
	if len(eventData) == 0 || string(eventData) == "null" {
		return false
	}

	var evt stacktraceEvent
	if err := json.Unmarshal(eventData, &evt); err != nil {
		return false
	}

	rules := parseStacktraceRules(rulesRaw)
	for _, val := range evt.Exception.Values {
		for _, frame := range val.Stacktrace.Frames {
			inApp := frame.InApp != nil && *frame.InApp
			if classifyStackFrame(frame.Filename, rules, inApp) == "app" {
				return true
			}
		}
	}
	return false
}

func parseStacktraceRules(raw json.RawMessage) StacktraceRules {
	if len(raw) == 0 || string(raw) == "null" {
		return StacktraceRules{}
	}
	var rules StacktraceRules
	if err := json.Unmarshal(raw, &rules); err != nil {
		return StacktraceRules{}
	}
	return StacktraceRules{
		Preset:            strings.TrimSpace(rules.Preset),
		AppPatterns:       normalizePatternList(rules.AppPatterns),
		FrameworkPatterns: normalizePatternList(rules.FrameworkPatterns),
		ExternalPatterns:  normalizePatternList(rules.ExternalPatterns),
	}
}

func normalizePatternList(patterns []string) []string {
	out := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		out = append(out, pattern)
	}
	return out
}

func classifyStackFrame(filename string, rules StacktraceRules, inApp bool) string {
	normalizedPath := strings.ReplaceAll(filename, "\\", "/")

	if matchesRegexPattern(normalizedPath, rules.AppPatterns) {
		return "app"
	}
	if matchesRegexPattern(normalizedPath, rules.FrameworkPatterns) {
		return "framework"
	}
	if matchesRegexPattern(normalizedPath, rules.ExternalPatterns) {
		return "external"
	}

	lower := strings.ToLower(normalizedPath)
	if matchesBuiltInExternalPath(lower) {
		return "external"
	}
	if matchesBuiltInFrameworkPath(lower) {
		return "framework"
	}
	if matchesBuiltInAppPath(lower) {
		return "app"
	}
	if inApp {
		return "app"
	}
	return "external"
}

func matchesRegexPattern(path string, patterns []string) bool {
	for _, pattern := range patterns {
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			continue
		}
		if re.MatchString(path) {
			return true
		}
	}
	return false
}

func matchesBuiltInAppPath(path string) bool {
	return matchesAny(path,
		`(^|/)application/`,
		`(^|/)app/`,
		`(^|/)src/`,
		`(^|/)(backend|service|api|routes|controllers)/`,
		`(^|/)(config|database|db)/`,
		`(^|/)lib/`,
	)
}

func matchesBuiltInFrameworkPath(path string) bool {
	return matchesAny(path,
		`(^|/)(system|framework)/`,
		`(^|/)(fastapi|starlette|pydantic|uvicorn|django)/`,
		`(^|/)(actionpack|activerecord|activesupport)/`,
		`(^|/)org/springframework/`,
		`(^|/)(gems|ruby|\.m2|gradle/caches)/`,
	)
}

func matchesBuiltInExternalPath(path string) bool {
	return matchesAny(path,
		`(^|/)(vendor|node_modules|site-packages|dist-packages)/`,
		`(^|/)vendor/bundle/`,
		`(^|/)boot-inf/lib/`,
	)
}

func matchesAny(path string, patterns ...string) bool {
	for _, pattern := range patterns {
		if regexp.MustCompile(pattern).MatchString(path) {
			return true
		}
	}
	return false
}
