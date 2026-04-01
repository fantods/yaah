package provider

import (
	"encoding/json"
	"strings"
)

type PartialJSONParser struct {
	buf strings.Builder
}

func NewPartialJSONParser() *PartialJSONParser {
	return &PartialJSONParser{}
}

func (p *PartialJSONParser) Parse(chunk string) (map[string]any, bool) {
	p.buf.WriteString(chunk)
	raw := p.buf.String()

	result, err := parseJSONObject(raw)
	if err == nil {
		return result, true
	}

	completed := tryCompleteJSON(raw)
	if completed != nil {
		result, err = parseJSONObject(*completed)
		if err == nil {
			return result, true
		}
	}

	return nil, false
}

func (p *PartialJSONParser) Reset() {
	p.buf.Reset()
}

func (p *PartialJSONParser) Buffer() string {
	return p.buf.String()
}

func parseJSONObject(s string) (map[string]any, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return map[string]any{}, nil
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil, err
	}
	if result == nil {
		return map[string]any{}, nil
	}
	return result, nil
}

func tryCompleteJSON(s string) *string {
	s = strings.TrimSpace(s)

	if len(s) == 0 || s[0] != '{' {
		return nil
	}

	completed := s

	if !isBalanced(completed) {
		completed += strings.Repeat("}", depthDiff(completed))
	}

	completed = stripTrailingComma(completed)

	return &completed
}

func isBalanced(s string) bool {
	depth := 0
	inStr := false
	escape := false

	for _, c := range s {
		if escape {
			escape = false
			continue
		}
		if c == '\\' && inStr {
			escape = true
			continue
		}
		if c == '"' {
			inStr = !inStr
			continue
		}
		if inStr {
			continue
		}
		switch c {
		case '{', '[':
			depth++
		case '}', ']':
			depth--
		}
	}

	return depth == 0
}

func depthDiff(s string) int {
	depth := 0
	inStr := false
	escape := false

	for _, c := range s {
		if escape {
			escape = false
			continue
		}
		if c == '\\' && inStr {
			escape = true
			continue
		}
		if c == '"' {
			inStr = !inStr
			continue
		}
		if inStr {
			continue
		}
		switch c {
		case '{', '[':
			depth++
		case '}', ']':
			depth--
		}
	}

	if depth <= 0 {
		return 0
	}
	return depth
}

func stripTrailingComma(s string) string {
	trimmed := strings.TrimRight(s, " \t\n\r")
	if strings.HasSuffix(trimmed, ",") {
		return trimmed[:len(trimmed)-1]
	}
	if strings.HasSuffix(trimmed, ",}") {
		return trimmed[:len(trimmed)-2] + "}"
	}
	return s
}
