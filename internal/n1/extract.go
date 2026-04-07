// Package n1 handles N+1 query detection: extracting query summaries from events
// and detecting repeated query patterns across requests.
package n1

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/darkspock/gosnag/internal/database/db"
	"github.com/google/uuid"
)

// QuerySummaryItem matches the format injected by gosnag-agent.
type QuerySummaryItem struct {
	Query   string  `json:"query"`
	Table   string  `json:"table"`
	Count   int     `json:"count"`
	TotalMs float64 `json:"total_ms"`
	AvgMs   float64 `json:"avg_ms"`
	Hash    string  `json:"hash"`
}

// ExtractAndStore reads gosnag_query_summary from event data and upserts
// each entry into the query_patterns table. Should be called asynchronously
// after event creation.
func ExtractAndStore(ctx context.Context, queries *db.Queries, projectID uuid.UUID, eventData json.RawMessage) {
	var raw map[string]any
	if err := json.Unmarshal(eventData, &raw); err != nil {
		return
	}

	extra, ok := raw["extra"].(map[string]any)
	if !ok {
		return
	}

	summaryRaw, ok := extra["gosnag_query_summary"]
	if !ok {
		return
	}

	// Re-marshal and unmarshal to get typed items
	summaryJSON, err := json.Marshal(summaryRaw)
	if err != nil {
		return
	}

	var items []QuerySummaryItem
	if err := json.Unmarshal(summaryJSON, &items); err != nil {
		return
	}

	if len(items) == 0 {
		return
	}

	// Get transaction name from event
	transaction, _ := raw["transaction"].(string)

	for _, item := range items {
		if item.Hash == "" || item.Query == "" {
			continue
		}
		if err := queries.UpsertQueryPattern(ctx, db.UpsertQueryPatternParams{
			ProjectID:       projectID,
			Transaction:     transaction,
			QueryHash:       item.Hash,
			NormalizedQuery: item.Query,
			TableName:       item.Table,
			EventCount:      int32(item.Count),
			TotalExecMs:     item.TotalMs,
		}); err != nil {
			slog.Error("failed to upsert query pattern", "error", err, "project_id", projectID, "hash", item.Hash)
		}
	}
}
