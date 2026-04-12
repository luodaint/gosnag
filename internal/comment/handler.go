package comment

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/darkspock/gosnag/internal/activity"
	"github.com/darkspock/gosnag/internal/auth"
	"github.com/darkspock/gosnag/internal/database/db"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type NotifyFunc func(issueID, projectID uuid.UUID, issueTitle, action string, excludeUserID *uuid.UUID)

type Handler struct {
	queries  *db.Queries
	notifyFn NotifyFunc
}

func NewHandler(queries *db.Queries, notifyFn ...NotifyFunc) *Handler {
	h := &Handler{queries: queries}
	if len(notifyFn) > 0 {
		h.notifyFn = notifyFn[0]
	}
	return h
}

type CommentResponse struct {
	ID         string `json:"id"`
	IssueID    string `json:"issue_id"`
	UserID     string `json:"user_id"`
	UserName   string `json:"user_name"`
	UserEmail  string `json:"user_email"`
	UserAvatar string `json:"user_avatar"`
	Body       string `json:"body"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

func toResponse(r db.ListIssueCommentsRow) CommentResponse {
	return CommentResponse{
		ID:         r.ID.String(),
		IssueID:    r.IssueID.String(),
		UserID:     r.UserID.String(),
		UserName:   r.UserName,
		UserEmail:  r.UserEmail,
		UserAvatar: r.UserAvatar,
		Body:       r.Body,
		CreatedAt:  r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  r.UpdatedAt.Format(time.RFC3339),
	}
}

func (h *Handler) validateIssueScope(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	projectID, err := uuid.Parse(chi.URLParam(r, "project_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return uuid.Nil, false
	}
	issueID, err := uuid.Parse(chi.URLParam(r, "issue_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid issue id")
		return uuid.Nil, false
	}
	issue, err := h.queries.GetIssue(r.Context(), issueID)
	if err != nil {
		writeError(w, http.StatusNotFound, "issue not found")
		return uuid.Nil, false
	}
	if issue.ProjectID != projectID {
		writeError(w, http.StatusNotFound, "issue not found")
		return uuid.Nil, false
	}
	return issueID, true
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	issueID, ok := h.validateIssueScope(w, r)
	if !ok {
		return
	}
	rows, err := h.queries.ListIssueComments(r.Context(), issueID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list comments")
		return
	}
	out := make([]CommentResponse, len(rows))
	for i, row := range rows {
		out[i] = toResponse(row)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	issueID, ok := h.validateIssueScope(w, r)
	if !ok {
		return
	}
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Body == "" {
		writeError(w, http.StatusBadRequest, "body is required")
		return
	}

	comment, err := h.queries.CreateIssueComment(r.Context(), db.CreateIssueCommentParams{
		IssueID: issueID,
		UserID:  user.ID,
		Body:    req.Body,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create comment")
		return
	}

	activity.Record(r.Context(), h.queries, issueID, nil, &user.ID, "commented", "", "", map[string]string{"comment_id": comment.ID.String()})

	if h.notifyFn != nil {
		issue, _ := h.queries.GetIssue(r.Context(), issueID)
		projectID, _ := uuid.Parse(chi.URLParam(r, "project_id"))
		go h.notifyFn(issueID, projectID, issue.Title, fmt.Sprintf("New comment by %s", user.Name), &user.ID)

		// Notify @mentioned users who may not be following
		mentionRe := regexp.MustCompile(`@([\w.@+-]+)`)
		matches := mentionRe.FindAllStringSubmatch(req.Body, -1)
		for _, m := range matches {
			if mentioned, err := h.queries.GetUserByNameOrEmail(r.Context(), m[1]); err == nil && mentioned.ID != user.ID {
				// Auto-follow mentioned users so they see future updates
				h.queries.FollowIssue(r.Context(), db.FollowIssueParams{UserID: mentioned.ID, IssueID: issueID})
			}
		}
	}

	// Return full response with user info
	writeJSON(w, http.StatusCreated, CommentResponse{
		ID:         comment.ID.String(),
		IssueID:    comment.IssueID.String(),
		UserID:     comment.UserID.String(),
		UserName:   user.Name,
		UserEmail:  user.Email,
		UserAvatar: user.AvatarUrl,
		Body:       comment.Body,
		CreatedAt:  comment.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  comment.UpdatedAt.Format(time.RFC3339),
	})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	_, ok := h.validateIssueScope(w, r)
	if !ok {
		return
	}
	commentID, err := uuid.Parse(chi.URLParam(r, "comment_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid comment id")
		return
	}

	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	existing, err := h.queries.GetIssueComment(r.Context(), commentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "comment not found")
		return
	}
	if existing.UserID != user.ID && user.Role != "admin" {
		writeError(w, http.StatusForbidden, "can only edit your own comments")
		return
	}

	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Body == "" {
		writeError(w, http.StatusBadRequest, "body is required")
		return
	}

	updated, err := h.queries.UpdateIssueComment(r.Context(), db.UpdateIssueCommentParams{
		ID:   commentID,
		Body: req.Body,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update comment")
		return
	}

	writeJSON(w, http.StatusOK, CommentResponse{
		ID:        updated.ID.String(),
		IssueID:   updated.IssueID.String(),
		UserID:    updated.UserID.String(),
		UserName:  user.Name,
		UserEmail: user.Email,
		Body:      updated.Body,
		CreatedAt: updated.CreatedAt.Format(time.RFC3339),
		UpdatedAt: updated.UpdatedAt.Format(time.RFC3339),
	})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	_, ok := h.validateIssueScope(w, r)
	if !ok {
		return
	}
	commentID, err := uuid.Parse(chi.URLParam(r, "comment_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid comment id")
		return
	}

	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	existing, err := h.queries.GetIssueComment(r.Context(), commentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "comment not found")
		return
	}
	if existing.UserID != user.ID && user.Role != "admin" {
		writeError(w, http.StatusForbidden, "can only delete your own comments")
		return
	}

	if err := h.queries.DeleteIssueComment(r.Context(), commentID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete comment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
