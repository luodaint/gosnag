package activity

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"

	"github.com/darkspock/gosnag/internal/database/db"
	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
)

// Record logs an activity entry for an issue (and optionally a ticket).
// userID is nil for system-generated actions. ticketID is nil for issue-only activities.
func Record(ctx context.Context, queries *db.Queries, issueID uuid.UUID, ticketID *uuid.UUID, userID *uuid.UUID, action, oldValue, newValue string, metadata any) {
	var metaJSON pqtype.NullRawMessage
	if metadata != nil {
		if raw, err := json.Marshal(metadata); err == nil {
			metaJSON = pqtype.NullRawMessage{RawMessage: raw, Valid: true}
		}
	}

	var uid uuid.NullUUID
	if userID != nil {
		uid = uuid.NullUUID{UUID: *userID, Valid: true}
	}

	var tid uuid.NullUUID
	if ticketID != nil {
		tid = uuid.NullUUID{UUID: *ticketID, Valid: true}
	}

	_, err := queries.InsertActivity(ctx, db.InsertActivityParams{
		IssueID:  issueID,
		TicketID: tid,
		UserID:   uid,
		Action:   action,
		OldValue: toNullString(oldValue),
		NewValue: toNullString(newValue),
		Metadata: metaJSON,
	})
	if err != nil {
		slog.Error("failed to record activity", "error", err, "issue_id", issueID, "action", action)
	}
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
