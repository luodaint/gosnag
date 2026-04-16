package conditions

import (
	"context"
	"encoding/json"

	"github.com/darkspock/gosnag/internal/database/db"
	"github.com/google/uuid"
)

// LoadLatestEventData returns the raw JSON payload of the newest event for an issue.
func LoadLatestEventData(ctx context.Context, queries *db.Queries, issueID uuid.UUID) json.RawMessage {
	events, err := queries.ListEventsByIssue(ctx, db.ListEventsByIssueParams{
		IssueID: issueID,
		Limit:   1,
		Offset:  0,
	})
	if err != nil || len(events) == 0 {
		return nil
	}
	return events[0].Data
}
