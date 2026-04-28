package routegroup

import (
	"context"
	"testing"
	"time"

	"github.com/darkspock/gosnag/internal/sourcecode"
)

type fakeProvider struct {
	files map[string]string
}

func (f fakeProvider) FileURL(path string, line int, commitOrBranch string) string { return "" }
func (f fakeProvider) GetCommitsForFiles(ctx context.Context, files []string, since time.Time) ([]sourcecode.Commit, error) {
	return nil, nil
}
func (f fakeProvider) ResolveRef(ctx context.Context, ref string) (string, error) { return "", nil }
func (f fakeProvider) TestConnection(ctx context.Context) error                   { return nil }
func (f fakeProvider) GetFile(ctx context.Context, filePath string, ref string) ([]byte, error) {
	return []byte(f.files[filePath]), nil
}

func TestCodeIgniterCanonicalPath(t *testing.T) {
	got := codeIgniterCanonicalPath("payments/getPaymentReceipt/(:any)/(:any)")
	want := "/payments/getPaymentReceipt/:any/:any"
	if got != want {
		t.Fatalf("unexpected canonical path: got %q want %q", got, want)
	}
}

func TestRoutePatternToRegex(t *testing.T) {
	pattern := routePatternToRegex("api/v4/internal/payment/id/(:any)")
	if pattern == "" {
		t.Fatal("expected regex")
	}
	rule := Rule{Method: "GET", MatchPattern: "api/v4/internal/payment/id/(:any)"}
	if !MatchRule("GET", "/api/v4/internal/payment/id/12345", rule) {
		t.Fatal("expected rule to match request path")
	}
}

func TestCodeIgniterConventionRule(t *testing.T) {
	rule, ok := codeIgniterConventionRule("GET", "/coverApp/Reserv/getCalendar/4/2026")
	if !ok {
		t.Fatal("expected convention rule")
	}
	if rule.CanonicalPath != "/coverApp/Reserv/getCalendar/:int/:year" {
		t.Fatalf("unexpected canonical path: %q", rule.CanonicalPath)
	}
	if rule.Target != "coverApp::Reserv::getCalendar" {
		t.Fatalf("unexpected target: %q", rule.Target)
	}
	if rule.Source != "framework_convention" {
		t.Fatalf("unexpected source: %q", rule.Source)
	}
}
