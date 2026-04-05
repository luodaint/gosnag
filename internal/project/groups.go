package project

import (
	"encoding/json"
	"net/http"

	"github.com/darkspock/gosnag/internal/database/db"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handler) ListGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.queries.ListProjectGroups(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list groups")
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

type GroupRequest struct {
	Name string `json:"name"`
}

func (h *Handler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req GroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	group, err := h.queries.CreateProjectGroup(r.Context(), req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create group")
		return
	}
	writeJSON(w, http.StatusCreated, group)
}

func (h *Handler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	groupID, err := uuid.Parse(chi.URLParam(r, "group_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid group id")
		return
	}

	var req GroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	group, err := h.queries.UpdateProjectGroup(r.Context(), db.UpdateProjectGroupParams{
		ID:   groupID,
		Name: req.Name,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update group")
		return
	}
	writeJSON(w, http.StatusOK, group)
}

func (h *Handler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	groupID, err := uuid.Parse(chi.URLParam(r, "group_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid group id")
		return
	}

	err = h.queries.DeleteProjectGroup(r.Context(), groupID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete group")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
