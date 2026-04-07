package ingest

import "testing"

func boolPtr(v bool) *bool { return &v }

func TestFingerprintVendorException(t *testing.T) {
	// Same vendor exception reached via different app call paths should produce the same fingerprint
	eventA := &SentryEvent{
		Exception: &ExceptionData{Values: []ExceptionValue{{
			Type: "UnexpectedValueException",
			Stacktrace: &Stacktrace{Frames: []Frame{
				{Filename: "/public/index.php", InApp: boolPtr(true)},
				{Filename: "/vendor/monolog/StreamHandler.php", Function: "write", InApp: boolPtr(false)},
			}},
		}}},
	}

	eventB := &SentryEvent{
		Exception: &ExceptionData{Values: []ExceptionValue{{
			Type: "UnexpectedValueException",
			Stacktrace: &Stacktrace{Frames: []Frame{
				{Filename: "/public/index.php", InApp: boolPtr(true)},
				{Filename: "/app/Middleware/Auth.php", Function: "handle", InApp: boolPtr(true)},
				{Filename: "/app/Controllers/FooController.php", Function: "index", InApp: boolPtr(true)},
				{Filename: "/vendor/monolog/StreamHandler.php", Function: "write", InApp: boolPtr(false)},
			}},
		}}},
	}

	fpA := eventA.ComputeFingerprint()
	fpB := eventB.ComputeFingerprint()

	if fpA != fpB {
		t.Fatalf("vendor exception with different callers should have same fingerprint, got %s vs %s", fpA, fpB)
	}
}

func TestFingerprintAppException(t *testing.T) {
	// Same exception thrown from different in_app locations should produce different fingerprints
	eventA := &SentryEvent{
		Exception: &ExceptionData{Values: []ExceptionValue{{
			Type: "RuntimeException",
			Stacktrace: &Stacktrace{Frames: []Frame{
				{Filename: "/app/ServiceA.php", Function: "doWork", InApp: boolPtr(true)},
			}},
		}}},
	}

	eventB := &SentryEvent{
		Exception: &ExceptionData{Values: []ExceptionValue{{
			Type: "RuntimeException",
			Stacktrace: &Stacktrace{Frames: []Frame{
				{Filename: "/app/ServiceB.php", Function: "doOtherWork", InApp: boolPtr(true)},
			}},
		}}},
	}

	fpA := eventA.ComputeFingerprint()
	fpB := eventB.ComputeFingerprint()

	if fpA == fpB {
		t.Fatalf("different in_app throw locations should have different fingerprints")
	}
}

func TestFingerprintCustom(t *testing.T) {
	event := &SentryEvent{
		Fingerprint: []string{"my-custom-group"},
		Message:     "some message",
	}

	fp := event.ComputeFingerprint()
	if fp == "" {
		t.Fatal("expected non-empty fingerprint")
	}

	// Same custom fingerprint should produce same hash
	event2 := &SentryEvent{
		Fingerprint: []string{"my-custom-group"},
		Message:     "different message",
	}
	if event.ComputeFingerprint() != event2.ComputeFingerprint() {
		t.Fatal("same custom fingerprint should produce same hash")
	}
}

func TestFingerprintMessageFallback(t *testing.T) {
	eventA := &SentryEvent{Message: "connection timeout"}
	eventB := &SentryEvent{Message: "connection timeout"}
	eventC := &SentryEvent{Message: "disk full"}

	if eventA.ComputeFingerprint() != eventB.ComputeFingerprint() {
		t.Fatal("same message should produce same fingerprint")
	}
	if eventA.ComputeFingerprint() == eventC.ComputeFingerprint() {
		t.Fatal("different messages should produce different fingerprints")
	}
}
