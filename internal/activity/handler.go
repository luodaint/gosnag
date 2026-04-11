package activity

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/darkspock/gosnag/internal/database/db"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	queries *db.Queries
}

func NewHandler(queries *db.Queries) *Handler {
	return &Handler{queries: queries}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "project_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}
	issueID, err := uuid.Parse(chi.URLParam(r, "issue_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid issue id")
		return
	}

	// Verify issue belongs to this project
	issue, err := h.queries.GetIssue(r.Context(), issueID)
	if err != nil || issue.ProjectID != projectID {
		writeError(w, http.StatusNotFound, "issue not found")
		return
	}

	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32)
	offset, _ := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 32)
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	rows, err := h.queries.ListActivitiesByIssue(r.Context(), db.ListActivitiesByIssueParams{
		IssueID: issueID,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list activities")
		return
	}

	count, _ := h.queries.CountActivitiesByIssue(r.Context(), issueID)

	type ActivityResponse struct {
		ID         string          `json:"id"`
		IssueID    string          `json:"issue_id"`
		UserID     *string         `json:"user_id"`
		UserName   *string         `json:"user_name"`
		UserEmail  *string         `json:"user_email"`
		UserAvatar *string         `json:"user_avatar"`
		Action     string          `json:"action"`
		OldValue   *string         `json:"old_value"`
		NewValue   *string         `json:"new_value"`
		Metadata   json.RawMessage `json:"metadata"`
		CreatedAt  string          `json:"created_at"`
	}

	items := make([]ActivityResponse, len(rows))
	for i, row := range rows {
		item := ActivityResponse{
			ID:        row.ID.String(),
			IssueID:   row.IssueID.String(),
			Action:    row.Action,
			CreatedAt: row.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		if row.UserID.Valid {
			s := row.UserID.UUID.String()
			item.UserID = &s
		}
		if row.UserName.Valid {
			item.UserName = &row.UserName.String
		}
		if row.UserEmail.Valid {
			item.UserEmail = &row.UserEmail.String
		}
		if row.UserAvatar.Valid {
			item.UserAvatar = &row.UserAvatar.String
		}
		if row.OldValue.Valid {
			item.OldValue = &row.OldValue.String
		}
		if row.NewValue.Valid {
			item.NewValue = &row.NewValue.String
		}
		if row.Metadata.Valid {
			item.Metadata = row.Metadata.RawMessage
		}
		items[i] = item
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"activities": items,
		"total":      count,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
