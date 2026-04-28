package conditions

import (
	"encoding/json"
	"strings"
)

// Group represents an AND/OR group of conditions, optionally nested.
type Group struct {
	Operator   string `json:"operator"` // "and" or "or"
	Conditions []Node `json:"conditions"`
}

// Node is either a leaf Condition or a nested Group.
// Discriminated by the presence of Operator (group) vs Type (leaf).
type Node struct {
	// Leaf condition fields
	Type  string          `json:"type,omitempty"`
	Op    string          `json:"op,omitempty"`
	Value json.RawMessage `json:"value,omitempty"`

	// Nested group fields (if Operator is set, this is a group)
	Operator   string `json:"operator,omitempty"`
	Conditions []Node `json:"conditions,omitempty"`
}

// IsGroup returns true if this node is a nested group.
func (n Node) IsGroup() bool {
	return n.Operator != ""
}

// AsGroup converts a Node to a Group for evaluation.
func (n Node) AsGroup() Group {
	return Group{Operator: n.Operator, Conditions: n.Conditions}
}

// StringValue extracts a string value from the JSON.
func (n Node) StringValue() string {
	var s string
	if err := json.Unmarshal(n.Value, &s); err != nil {
		return string(n.Value)
	}
	return s
}

// IntValue extracts an int32 value from the JSON.
func (n Node) IntValue() int32 {
	var v int32
	if err := json.Unmarshal(n.Value, &v); err != nil {
		// Try float
		var f float64
		if err := json.Unmarshal(n.Value, &f); err == nil {
			return int32(f)
		}
		return 0
	}
	return v
}

// StringSliceValue extracts a []string value from the JSON.
func (n Node) StringSliceValue() []string {
	// Try array first
	var arr []string
	if err := json.Unmarshal(n.Value, &arr); err == nil {
		return arr
	}
	// Fall back to single string
	var s string
	if err := json.Unmarshal(n.Value, &s); err == nil {
		return []string{s}
	}
	return nil
}

// BoolValue extracts a bool value from the JSON.
func (n Node) BoolValue() bool {
	var b bool
	if err := json.Unmarshal(n.Value, &b); err == nil {
		return b
	}

	var s string
	if err := json.Unmarshal(n.Value, &s); err == nil {
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "true", "1", "yes":
			return true
		}
	}

	return false
}
