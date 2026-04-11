package workflow

import (
	"testing"
)

func TestValidTransitions(t *testing.T) {
	valid := []struct {
		from, to string
	}{
		{StatusAcknowledged, StatusInProgress},
		{StatusAcknowledged, StatusDone},
		{StatusAcknowledged, StatusWontfix},
		{StatusAcknowledged, StatusEscalated},
		{StatusInProgress, StatusInReview},
		{StatusInProgress, StatusDone},
		{StatusInProgress, StatusWontfix},
		{StatusInProgress, StatusEscalated},
		{StatusInReview, StatusInProgress},
		{StatusInReview, StatusDone},
		{StatusInReview, StatusWontfix},
		{StatusInReview, StatusEscalated},
		{StatusEscalated, StatusInProgress},
		{StatusEscalated, StatusDone},
		{StatusDone, StatusAcknowledged},
		{StatusWontfix, StatusAcknowledged},
	}

	for _, tc := range valid {
		if !IsValidTransition(tc.from, tc.to) {
			t.Errorf("expected %s -> %s to be valid", tc.from, tc.to)
		}
	}
}

func TestInvalidTransitions(t *testing.T) {
	invalid := []struct {
		from, to string
	}{
		{StatusAcknowledged, StatusInReview},
		{StatusInProgress, StatusAcknowledged},
		{StatusDone, StatusInProgress},
		{StatusDone, StatusEscalated},
		{StatusWontfix, StatusInProgress},
		{StatusWontfix, StatusEscalated},
		{StatusEscalated, StatusAcknowledged},
		{"nonexistent", StatusDone},
		{StatusAcknowledged, StatusAcknowledged},
	}

	for _, tc := range invalid {
		if IsValidTransition(tc.from, tc.to) {
			t.Errorf("expected %s -> %s to be invalid", tc.from, tc.to)
		}
	}
}

func TestValidNextStatuses(t *testing.T) {
	next := ValidNextStatuses(StatusAcknowledged)
	if len(next) != 4 {
		t.Errorf("acknowledged should have 4 next statuses, got %d", len(next))
	}

	next = ValidNextStatuses("nonexistent")
	if next != nil {
		t.Error("unknown status should return nil")
	}
}

func TestAllStatuses(t *testing.T) {
	all := AllStatuses()
	if len(all) != 6 {
		t.Errorf("expected 6 statuses, got %d", len(all))
	}
}
