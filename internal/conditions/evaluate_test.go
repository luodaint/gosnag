package conditions

import (
	"encoding/json"
	"testing"
)

func TestEvaluateHasAppFrame(t *testing.T) {
	ctx := NewEvalContext(IssueData{HasAppFrame: true}, "", nil)
	group := Group{
		Operator: "and",
		Conditions: []Node{{
			Type:  "has_app_frame",
			Op:    "eq",
			Value: json.RawMessage(`true`),
		}},
	}

	if !Evaluate(group, ctx) {
		t.Fatal("expected has_app_frame condition to match")
	}
}

func TestHasAppFrameUsesStacktraceRules(t *testing.T) {
	eventData := json.RawMessage(`{
		"exception": {
			"values": [{
				"stacktrace": {
					"frames": [
						{"filename": "/var/app/current/system/core/CodeIgniter.php"},
						{"filename": "/var/app/current/application/controllers/Tables.php"}
					]
				}
			}]
		}
	}`)
	rules := json.RawMessage(`{
		"preset": "codeigniter",
		"app_patterns": ["(^|/)application/"],
		"framework_patterns": ["(^|/)system/"],
		"external_patterns": ["(^|/)vendor/"]
	}`)

	if !HasAppFrame(eventData, rules) {
		t.Fatal("expected application frame to be detected")
	}
}
