package ticket

import (
	"encoding/json"
	"net/http"

	"github.com/darkspock/gosnag/internal/auth"
	"github.com/darkspock/gosnag/internal/database/db"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handler) ListAttachments(w http.ResponseWriter, r *http.Request) {
	ticketID, err := uuid.Parse(chi.URLParam(r, "ticket_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	attachments, err := h.queries.ListAttachmentsByTicket(r.Context(), ticketID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list attachments")
		return
	}

	items := make([]map[string]any, len(attachments))
	for i, a := range attachments {
		items[i] = map[string]any{
			"id":            a.ID,
			"ticket_id":     a.TicketID,
			"filename":      a.Filename,
			"url":           a.Url,
			"content_type":  a.ContentType,
			"size_bytes":    a.SizeBytes,
			"uploaded_by":   a.UploadedBy,
			"uploader_name":  a.UploaderName,
			"uploader_email": a.UploaderEmail,
			"created_at":    a.CreatedAt,
		}
	}

	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) AddAttachment(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "project_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}
	ticketID, err := uuid.Parse(chi.URLParam(r, "ticket_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	// Verify ticket belongs to project
	ticket, err := h.queries.GetTicket(r.Context(), ticketID)
	if err != nil || ticket.ProjectID != projectID {
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}

	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Filename    string `json:"filename"`
		URL         string `json:"url"`
		ContentType string `json:"content_type"`
		Size        int64  `json:"size"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}
	if req.Filename == "" {
		req.Filename = "attachment"
	}

	att, err := h.queries.CreateAttachment(r.Context(), db.CreateAttachmentParams{
		TicketID:    ticketID,
		Filename:    req.Filename,
		Url:         req.URL,
		ContentType: req.ContentType,
		SizeBytes:   req.Size,
		UploadedBy:  user.ID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create attachment")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":           att.ID,
		"ticket_id":    att.TicketID,
		"filename":     att.Filename,
		"url":          att.Url,
		"content_type": att.ContentType,
		"size_bytes":   att.SizeBytes,
		"created_at":   att.CreatedAt,
	})
}

func (h *Handler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "project_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}
	ticketID, err := uuid.Parse(chi.URLParam(r, "ticket_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ticket id")
		return
	}
	attachmentID, err := uuid.Parse(chi.URLParam(r, "attachment_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid attachment id")
		return
	}

	// Verify ticket belongs to project
	ticket, err := h.queries.GetTicket(r.Context(), ticketID)
	if err != nil || ticket.ProjectID != projectID {
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}

	if err := h.queries.DeleteAttachment(r.Context(), db.DeleteAttachmentParams{
		ID:       attachmentID,
		TicketID: ticketID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete attachment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
