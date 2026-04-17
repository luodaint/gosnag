package routegroup

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/darkspock/gosnag/internal/sourcecode"
)

var (
	ciRouteRe     = regexp.MustCompile(`\$route\[['"]([^'"]+)['"]\]\s*=\s*['"]([^'"]*)['"]\s*;`)
	ciVerbRouteRe = regexp.MustCompile(`\$route\[['"]([^'"]+)['"]\]\[['"]([A-Za-z]+)['"]\]\s*=\s*['"]([^'"]*)['"]\s*;`)
	ciIncludeRe   = regexp.MustCompile(`(?:require|require_once|include|include_once)\s+APPPATH\s*\.\s*['"]([^'"]+)['"]\s*;`)
)

func ImportCodeIgniterRules(ctx context.Context, provider sourcecode.Provider, ref string) ([]ImportRule, error) {
	files := []string{"application/config/routes.php"}
	seenFiles := map[string]bool{}
	rulesByKey := map[string]ImportRule{}

	for len(files) > 0 {
		file := files[0]
		files = files[1:]
		if seenFiles[file] {
			continue
		}
		seenFiles[file] = true

		body, err := provider.GetFile(ctx, file, ref)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", file, err)
		}

		for _, match := range ciIncludeRe.FindAllStringSubmatch(string(body), -1) {
			includePath := strings.TrimSpace(match[1])
			if includePath == "" {
				continue
			}
			files = append(files, path.Join("application", includePath))
		}

		for _, match := range ciRouteRe.FindAllStringSubmatch(string(body), -1) {
			routeKey := strings.TrimSpace(match[1])
			target := strings.TrimSpace(match[2])
			if shouldSkipCodeIgniterRoute(routeKey, target) {
				continue
			}
			rule := ImportRule{
				Method:        "*",
				MatchPattern:  routeKey,
				CanonicalPath: codeIgniterCanonicalPath(routeKey),
				Target:        target,
				Source:        "source_code",
				Confidence:    1,
				Enabled:       true,
				Framework:     "codeigniter",
				SourceFile:    file,
			}
			rulesByKey[rule.Method+"|"+rule.MatchPattern+"|"+rule.Target] = rule
		}

		for _, match := range ciVerbRouteRe.FindAllStringSubmatch(string(body), -1) {
			routeKey := strings.TrimSpace(match[1])
			method := strings.TrimSpace(match[2])
			target := strings.TrimSpace(match[3])
			if shouldSkipCodeIgniterRoute(routeKey, target) {
				continue
			}
			rule := ImportRule{
				Method:        method,
				MatchPattern:  routeKey,
				CanonicalPath: codeIgniterCanonicalPath(routeKey),
				Target:        target,
				Source:        "source_code",
				Confidence:    1,
				Enabled:       true,
				Framework:     "codeigniter",
				SourceFile:    file,
			}
			rulesByKey[normalizeMethod(rule.Method)+"|"+rule.MatchPattern+"|"+rule.Target] = rule
		}
	}

	out := make([]ImportRule, 0, len(rulesByKey))
	for _, rule := range rulesByKey {
		out = append(out, rule)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CanonicalPath == out[j].CanonicalPath {
			return normalizeMethod(out[i].Method) < normalizeMethod(out[j].Method)
		}
		return out[i].CanonicalPath < out[j].CanonicalPath
	})
	return out, nil
}

func shouldSkipCodeIgniterRoute(routeKey, target string) bool {
	switch routeKey {
	case "", "default_controller", "404_override", "translate_uri_dashes":
		return true
	}
	return strings.TrimSpace(target) == ""
}

func codeIgniterCanonicalPath(routeKey string) string {
	routeKey = strings.Trim(strings.TrimSpace(routeKey), "/")
	if routeKey == "" {
		return "/"
	}

	segments := strings.Split(routeKey, "/")
	for i, segment := range segments {
		switch {
		case segment == "(:any)":
			segments[i] = ":any"
		case segment == "(:num)":
			segments[i] = ":num"
		case strings.HasPrefix(segment, "(") && strings.HasSuffix(segment, ")"):
			segments[i] = ":param"
		default:
			segments[i] = segment
		}
	}

	return "/" + strings.Join(segments, "/")
}

func codeIgniterConventionRule(method, requestPath string) (Rule, bool) {
	pathOnly := strings.Trim(strings.TrimSpace(requestPath), "/")
	if pathOnly == "" {
		return Rule{}, false
	}

	segments := strings.Split(pathOnly, "/")
	if len(segments) < 2 {
		return Rule{}, false
	}

	staticSegments := 3
	if len(segments) < staticSegments {
		staticSegments = len(segments)
	}

	canonical := make([]string, 0, len(segments))
	canonical = append(canonical, segments[:staticSegments]...)
	for _, segment := range segments[staticSegments:] {
		canonical = append(canonical, normalizeCodeIgniterConventionSegment(segment))
	}

	target := strings.Join(segments[:staticSegments], "::")
	if staticSegments == 1 {
		target = segments[0]
	}

	return Rule{
		Method:        normalizeMethod(method),
		MatchPattern:  pathOnly,
		CanonicalPath: "/" + strings.Join(canonical, "/"),
		Target:        target,
		Source:        "framework_convention",
		Confidence:    0.8,
		Enabled:       true,
		Framework:     "codeigniter",
		Notes:         "Derived from CodeIgniter controller/method convention",
	}, true
}

func normalizeCodeIgniterConventionSegment(segment string) string {
	if _, err := strconv.Atoi(segment); err == nil {
		if len(segment) == 4 && (strings.HasPrefix(segment, "19") || strings.HasPrefix(segment, "20")) {
			return ":year"
		}
		return ":int"
	}
	return ":any"
}
