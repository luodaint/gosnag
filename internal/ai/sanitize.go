package ai

import (
	"regexp"
	"strings"
)

// allowedTags is the set of HTML tags allowed in AI-generated content.
var allowedTags = map[string]bool{
	"p": true, "h1": true, "h2": true, "h3": true,
	"ul": true, "ol": true, "li": true,
	"strong": true, "em": true, "b": true, "i": true,
	"code": true, "pre": true,
	"a": true, "br": true, "blockquote": true,
	"table": true, "thead": true, "tbody": true, "tr": true, "th": true, "td": true,
	"hr": true, "span": true,
}

// allowedAttrs is the set of attributes allowed per tag.
var allowedAttrs = map[string]map[string]bool{
	"a": {"href": true, "title": true},
}

var (
	tagPattern   = regexp.MustCompile(`<(/?)([a-zA-Z][a-zA-Z0-9]*)\b([^>]*)(/?)>`)
	eventPattern = regexp.MustCompile(`(?i)\bon[a-z]+\s*=`)
	stylePattern = regexp.MustCompile(`(?i)\bstyle\s*=`)
)

var codeFencePattern = regexp.MustCompile("(?s)^\\s*```(?:html)?\\s*\n?(.*?)\\s*```\\s*$")

// cleanCodeFences strips markdown code fences wrapping the response.
func cleanCodeFences(s string) string {
	if m := codeFencePattern.FindStringSubmatch(s); m != nil {
		return strings.TrimSpace(m[1])
	}
	return strings.TrimSpace(s)
}

// hasHTMLTags checks if the string contains any HTML tags.
func hasHTMLTags(s string) bool {
	return tagPattern.MatchString(s)
}

// markdownToBasicHTML converts simple markdown to HTML for cases where
// the model ignores the HTML instruction.
func markdownToBasicHTML(md string) string {
	lines := strings.Split(md, "\n")
	var sb strings.Builder
	inList := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if inList {
				sb.WriteString("</ul>")
				inList = false
			}
			continue
		}

		// Headers
		if strings.HasPrefix(trimmed, "### ") {
			sb.WriteString("<h3>" + convertInline(strings.TrimPrefix(trimmed, "### ")) + "</h3>")
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			sb.WriteString("<h3>" + convertInline(strings.TrimPrefix(trimmed, "## ")) + "</h3>")
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			sb.WriteString("<h3>" + convertInline(strings.TrimPrefix(trimmed, "# ")) + "</h3>")
			continue
		}

		// List items
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			if !inList {
				sb.WriteString("<ul>")
				inList = true
			}
			sb.WriteString("<li>" + convertInline(trimmed[2:]) + "</li>")
			continue
		}

		// Numbered list
		if len(trimmed) > 2 && trimmed[0] >= '0' && trimmed[0] <= '9' && (trimmed[1] == '.' || (len(trimmed) > 2 && trimmed[2] == '.')) {
			idx := strings.Index(trimmed, ". ")
			if idx > 0 && idx <= 3 {
				if !inList {
					sb.WriteString("<ul>")
					inList = true
				}
				sb.WriteString("<li>" + convertInline(trimmed[idx+2:]) + "</li>")
				continue
			}
		}

		if inList {
			sb.WriteString("</ul>")
			inList = false
		}
		sb.WriteString("<p>" + convertInline(trimmed) + "</p>")
	}
	if inList {
		sb.WriteString("</ul>")
	}
	return sb.String()
}

var (
	boldPattern   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	italicPattern = regexp.MustCompile(`\*(.+?)\*`)
	codePattern   = regexp.MustCompile("`([^`]+)`")
)

func convertInline(s string) string {
	s = boldPattern.ReplaceAllString(s, "<strong>$1</strong>")
	s = codePattern.ReplaceAllString(s, "<code>$1</code>")
	s = italicPattern.ReplaceAllString(s, "<em>$1</em>")
	return s
}

// SanitizeHTML strips disallowed tags and attributes from HTML content.
// If the input is markdown (no HTML tags), it converts to basic HTML first.
func SanitizeHTML(input string) string {
	html := cleanCodeFences(input)
	if !hasHTMLTags(html) {
		html = markdownToBasicHTML(html)
	}
	return sanitizeTags(html)
}

// sanitizeTags strips disallowed tags and attributes.
func sanitizeTags(html string) string {
	return tagPattern.ReplaceAllStringFunc(html, func(tag string) string {
		m := tagPattern.FindStringSubmatch(tag)
		if m == nil {
			return ""
		}
		tagName := strings.ToLower(m[2])
		if !allowedTags[tagName] {
			return ""
		}

		attrs := m[3]
		// Strip event handlers and style attributes
		if eventPattern.MatchString(attrs) || stylePattern.MatchString(attrs) {
			attrs = eventPattern.ReplaceAllString(attrs, "")
			attrs = stylePattern.ReplaceAllString(attrs, "")
		}

		// For tags with allowed attrs, keep only those
		if allowed, ok := allowedAttrs[tagName]; ok {
			attrs = filterAttrs(attrs, allowed)
		} else {
			attrs = "" // no attrs allowed
		}

		closing := m[1]
		selfClosing := m[4]
		if attrs != "" {
			return "<" + closing + tagName + " " + strings.TrimSpace(attrs) + selfClosing + ">"
		}
		return "<" + closing + tagName + selfClosing + ">"
	})
}

func filterAttrs(attrs string, allowed map[string]bool) string {
	attrPattern := regexp.MustCompile(`([a-zA-Z-]+)\s*=\s*(?:"([^"]*)"|'([^']*)')`)
	matches := attrPattern.FindAllStringSubmatch(attrs, -1)
	var kept []string
	for _, m := range matches {
		name := strings.ToLower(m[1])
		if allowed[name] {
			value := m[2]
			if value == "" {
				value = m[3]
			}
			// Block javascript: URLs
			if name == "href" && strings.HasPrefix(strings.TrimSpace(strings.ToLower(value)), "javascript:") {
				continue
			}
			kept = append(kept, name+`="`+value+`"`)
		}
	}
	return strings.Join(kept, " ")
}
