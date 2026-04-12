package workflow

// Ticket statuses
const (
	StatusAcknowledged = "acknowledged"
	StatusInProgress   = "in_progress"
	StatusInReview     = "in_review"
	StatusDone         = "done"
	StatusWontfix      = "wontfix"
	StatusEscalated    = "escalated"
)

// Project workflow modes
const (
	ModeSimple  = "simple"
	ModeManaged = "managed"
)

// validTransitions defines the ticket state machine.
var validTransitions = map[string][]string{
	StatusAcknowledged: {StatusInProgress, StatusDone, StatusWontfix, StatusEscalated},
	StatusInProgress:   {StatusInReview, StatusDone, StatusWontfix, StatusEscalated},
	StatusInReview:     {StatusInProgress, StatusDone, StatusWontfix, StatusEscalated},
	StatusEscalated:    {StatusInProgress, StatusDone},
	StatusDone:         {StatusAcknowledged}, // reopen
	StatusWontfix:      {StatusAcknowledged}, // reopen
}

// AllStatuses returns all valid ticket statuses.
func AllStatuses() []string {
	return []string{StatusAcknowledged, StatusInProgress, StatusInReview, StatusDone, StatusWontfix, StatusEscalated}
}

// IsValidTransition checks if a ticket transition is allowed.
func IsValidTransition(from, to string) bool {
	targets, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == to {
			return true
		}
	}
	return false
}

// ValidNextStatuses returns all statuses reachable from the given status.
func ValidNextStatuses(from string) []string {
	return validTransitions[from]
}
