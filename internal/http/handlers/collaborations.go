// Package handlers provides HTTP handlers for the web UI.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/asabla/goknut/internal/http/dto"
	"github.com/asabla/goknut/internal/observability"
	"github.com/asabla/goknut/internal/repository"
	"github.com/asabla/goknut/internal/services"
)

// CollaborationHandler handles collaboration-related HTTP requests.
type CollaborationHandler struct {
	collabs   *services.CollaborationService
	profiles  *repository.ProfileRepository
	templates *template.Template
	logger    *observability.Logger
	metrics   *observability.Metrics
}

// NewCollaborationHandler creates a new collaboration handler.
func NewCollaborationHandler(
	collabs *services.CollaborationService,
	profiles *repository.ProfileRepository,
	templates *template.Template,
	logger *observability.Logger,
	metrics *observability.Metrics,
) *CollaborationHandler {
	return &CollaborationHandler{collabs: collabs, profiles: profiles, templates: templates, logger: logger, metrics: metrics}
}

// RegisterRoutes registers collaboration routes on the mux.
func (h *CollaborationHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /collaborations", h.handleList)
	mux.HandleFunc("GET /collaborations/new", h.handleNew)
	mux.HandleFunc("POST /collaborations", h.handleCreate)
	mux.HandleFunc("GET /collaborations/{id}", h.handleGet)
	mux.HandleFunc("POST /collaborations/{id}", h.handleUpdate)
	mux.HandleFunc("POST /collaborations/{id}/delete", h.handleDelete)
	mux.HandleFunc("POST /collaborations/{id}/participants", h.handleAddParticipant)
	mux.HandleFunc("POST /collaborations/{id}/participants/{profile_id}/remove", h.handleRemoveParticipant)
}

func (h *CollaborationHandler) handleList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	collabs, err := h.collabs.List(ctx)
	if err != nil {
		h.logger.Error("failed to list collaborations", "error", err)
		h.renderError(w, r, "Failed to load collaborations", http.StatusInternalServerError)
		return
	}

	collabDTOs := make([]dto.Collaboration, 0, len(collabs))
	for _, c := range collabs {
		collabDTOs = append(collabDTOs, dto.Collaboration{
			ID:          c.ID,
			Name:        c.Name,
			Description: c.Description,
			SharedChat:  c.SharedChat,
			CreatedAt:   c.CreatedAt,
			UpdatedAt:   c.UpdatedAt,
		})
	}

	data := map[string]any{
		"Collaborations": collabDTOs,
		"IsEmpty":        len(collabDTOs) == 0,
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "collaborations/index", data); err != nil {
		h.logger.Error("failed to execute collaborations/index template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *CollaborationHandler) handleNew(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "collaborations/new", map[string]any{
		"FormName":        "",
		"FormDescription": "",
		"FormSharedChat":  false,
	}); err != nil {
		h.logger.Error("failed to execute collaborations/new template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *CollaborationHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dto.CreateCollaborationRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if h.metrics != nil {
				h.metrics.RecordCollaborationCreate(false)
			}
			h.renderError(w, r, "Invalid request body", http.StatusBadRequest)
			return
		}
	} else {
		_ = r.ParseForm()
		req.Name = r.FormValue("name")
		req.Description = r.FormValue("description")
		req.SharedChat = r.FormValue("shared_chat") != ""
	}

	if err := req.Validate(); err != nil {
		if h.metrics != nil {
			h.metrics.RecordCollaborationCreate(false)
		}
		h.renderCreateFormError(w, r, err.Error(), req)
		return
	}

	c, err := h.collabs.Create(ctx, req.Name, req.Description, req.SharedChat)
	if err != nil {
		if h.metrics != nil {
			h.metrics.RecordCollaborationCreate(false)
		}
		if errors.Is(err, services.ErrInvalidCollaborationName) {
			h.renderCreateFormError(w, r, dto.ErrCollaborationNameRequired.Error(), req)
			return
		}
		h.logger.Error("failed to create collaboration", "error", err)
		h.renderCreateFormError(w, r, "Failed to create collaboration", req)
		return
	}

	h.logger.Info("collaboration created", "collaboration_id", c.ID)
	if h.metrics != nil {
		h.metrics.RecordCollaborationCreate(true)
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(dto.Collaboration{
			ID:          c.ID,
			Name:        c.Name,
			Description: c.Description,
			SharedChat:  c.SharedChat,
			CreatedAt:   c.CreatedAt,
			UpdatedAt:   c.UpdatedAt,
		})
		return
	}

	http.Redirect(w, r, "/collaborations/"+strconv.FormatInt(c.ID, 10), http.StatusSeeOther)
}

func (h *CollaborationHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	collaborationID, err := h.pathInt64(r, "id")
	if err != nil {
		h.renderError(w, r, "Invalid collaboration id", http.StatusBadRequest)
		return
	}

	c, participants, allProfiles, err := h.loadCollaborationDetail(ctx, collaborationID)
	if err != nil {
		if err == services.ErrCollaborationNotFound {
			h.renderError(w, r, "Collaboration not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to load collaboration detail", "collaboration_id", collaborationID, "error", err)
		h.renderError(w, r, "Failed to load collaboration", http.StatusInternalServerError)
		return
	}

	data := h.collaborationDetailTemplateData(c, participants, allProfiles, "", nil, nil, nil, nil)

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
		return
	}

	h.renderDetailTemplate(w, data, http.StatusOK)
}

func (h *CollaborationHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	collaborationID, err := h.pathInt64(r, "id")
	if err != nil {
		h.renderError(w, r, "Invalid collaboration id", http.StatusBadRequest)
		return
	}

	var req dto.UpdateCollaborationRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if h.metrics != nil {
				h.metrics.RecordCollaborationUpdate(false)
			}
			h.renderError(w, r, "Invalid request body", http.StatusBadRequest)
			return
		}
	} else {
		_ = r.ParseForm()
		req.Name = r.FormValue("name")
		req.Description = r.FormValue("description")
		req.SharedChat = r.FormValue("shared_chat") != ""
	}

	if err := req.Validate(); err != nil {
		if h.metrics != nil {
			h.metrics.RecordCollaborationUpdate(false)
		}
		if h.wantsJSON(r) {
			h.renderError(w, r, err.Error(), http.StatusBadRequest)
			return
		}
		h.renderDetailError(w, r, collaborationID, http.StatusBadRequest, err.Error(), &req.Name, &req.Description, &req.SharedChat, nil)
		return
	}

	_, err = h.collabs.Update(ctx, collaborationID, req.Name, req.Description, req.SharedChat)
	if err != nil {
		if h.metrics != nil {
			h.metrics.RecordCollaborationUpdate(false)
		}
		switch {
		case err == services.ErrCollaborationNotFound:
			h.renderError(w, r, "Collaboration not found", http.StatusNotFound)
			return
		case errors.Is(err, services.ErrInvalidCollaborationName):
			if h.wantsJSON(r) {
				h.renderError(w, r, dto.ErrCollaborationNameRequired.Error(), http.StatusBadRequest)
				return
			}
			h.renderDetailError(w, r, collaborationID, http.StatusBadRequest, dto.ErrCollaborationNameRequired.Error(), &req.Name, &req.Description, &req.SharedChat, nil)
			return
		default:
			h.logger.Error("failed to update collaboration", "collaboration_id", collaborationID, "error", err)
			if h.wantsJSON(r) {
				h.renderError(w, r, "Failed to update collaboration", http.StatusInternalServerError)
				return
			}
			h.renderDetailError(w, r, collaborationID, http.StatusInternalServerError, "Failed to update collaboration", &req.Name, &req.Description, &req.SharedChat, nil)
			return
		}
	}

	h.logger.Info("collaboration updated", "collaboration_id", collaborationID)
	if h.metrics != nil {
		h.metrics.RecordCollaborationUpdate(true)
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
		return
	}

	http.Redirect(w, r, "/collaborations/"+strconv.FormatInt(collaborationID, 10), http.StatusSeeOther)
}

func (h *CollaborationHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	collaborationID, err := h.pathInt64(r, "id")
	if err != nil {
		h.renderError(w, r, "Invalid collaboration id", http.StatusBadRequest)
		return
	}

	if err := h.collabs.Delete(ctx, collaborationID); err != nil {
		if h.metrics != nil {
			h.metrics.RecordCollaborationDelete(false)
		}
		if err == services.ErrCollaborationNotFound {
			h.renderError(w, r, "Collaboration not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to delete collaboration", "collaboration_id", collaborationID, "error", err)
		if h.wantsJSON(r) {
			h.renderError(w, r, "Failed to delete collaboration", http.StatusInternalServerError)
			return
		}
		h.renderDetailError(w, r, collaborationID, http.StatusInternalServerError, "Failed to delete collaboration", nil, nil, nil, nil)
		return
	}

	h.logger.Info("collaboration deleted", "collaboration_id", collaborationID)
	if h.metrics != nil {
		h.metrics.RecordCollaborationDelete(true)
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
		return
	}

	http.Redirect(w, r, "/collaborations", http.StatusSeeOther)
}

func (h *CollaborationHandler) handleAddParticipant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	collaborationID, err := h.pathInt64(r, "id")
	if err != nil {
		h.renderError(w, r, "Invalid collaboration id", http.StatusBadRequest)
		return
	}

	var req dto.AddCollaborationParticipantRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if h.metrics != nil {
				h.metrics.RecordCollaborationLink(false)
			}
			h.renderError(w, r, "Invalid request body", http.StatusBadRequest)
			return
		}
	} else {
		_ = r.ParseForm()
		if r.FormValue("profile_id") == "" {
			if h.wantsJSON(r) {
				h.renderError(w, r, dto.ErrCollaborationParticipantRequired.Error(), http.StatusBadRequest)
				return
			}
			h.renderDetailError(w, r, collaborationID, http.StatusBadRequest, dto.ErrCollaborationParticipantRequired.Error(), nil, nil, nil, nil)
			return
		}
		profileID, err := strconv.ParseInt(r.FormValue("profile_id"), 10, 64)
		if err != nil {
			if h.wantsJSON(r) {
				h.renderError(w, r, dto.ErrCollaborationParticipantIDInvalid.Error(), http.StatusBadRequest)
				return
			}
			h.renderDetailError(w, r, collaborationID, http.StatusBadRequest, dto.ErrCollaborationParticipantIDInvalid.Error(), nil, nil, nil, nil)
			return
		}
		req.ProfileID = profileID
	}

	if err := req.Validate(); err != nil {
		if h.metrics != nil {
			h.metrics.RecordCollaborationLink(false)
		}
		if h.wantsJSON(r) {
			h.renderError(w, r, err.Error(), http.StatusBadRequest)
			return
		}
		selected := req.ProfileID
		h.renderDetailError(w, r, collaborationID, http.StatusBadRequest, err.Error(), nil, nil, nil, &selected)
		return
	}

	if err := h.collabs.AddParticipant(ctx, collaborationID, req.ProfileID); err != nil {
		if h.metrics != nil {
			h.metrics.RecordCollaborationLink(false)
		}
		switch {
		case err == services.ErrCollaborationNotFound:
			h.renderError(w, r, "Collaboration not found", http.StatusNotFound)
			return
		case err == services.ErrProfileNotFound:
			h.renderError(w, r, "Profile not found", http.StatusNotFound)
			return
		case err == services.ErrCollaborationParticipant:
			if h.wantsJSON(r) {
				h.renderError(w, r, err.Error(), http.StatusConflict)
				return
			}
			selected := req.ProfileID
			h.renderDetailError(w, r, collaborationID, http.StatusConflict, "Participant already exists", nil, nil, nil, &selected)
			return
		default:
			h.logger.Error("failed to add collaboration participant", "collaboration_id", collaborationID, "profile_id", req.ProfileID, "error", err)
			if h.wantsJSON(r) {
				h.renderError(w, r, "Failed to add participant", http.StatusInternalServerError)
				return
			}
			selected := req.ProfileID
			h.renderDetailError(w, r, collaborationID, http.StatusInternalServerError, "Failed to add participant", nil, nil, nil, &selected)
			return
		}
	}

	h.logger.Info("collaboration participant added", "collaboration_id", collaborationID, "profile_id", req.ProfileID)
	if h.metrics != nil {
		h.metrics.RecordCollaborationLink(true)
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "added"})
		return
	}

	http.Redirect(w, r, "/collaborations/"+strconv.FormatInt(collaborationID, 10), http.StatusSeeOther)
}

func (h *CollaborationHandler) handleRemoveParticipant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	collaborationID, err := h.pathInt64(r, "id")
	if err != nil {
		h.renderError(w, r, "Invalid collaboration id", http.StatusBadRequest)
		return
	}

	profileID, err := h.pathInt64(r, "profile_id")
	if err != nil {
		h.renderError(w, r, "Invalid profile id", http.StatusBadRequest)
		return
	}

	if err := h.collabs.RemoveParticipant(ctx, collaborationID, profileID); err != nil {
		if h.metrics != nil {
			h.metrics.RecordCollaborationUnlink(false)
		}
		if err == services.ErrCollaborationNoParticipant {
			if h.wantsJSON(r) {
				h.renderError(w, r, "Participant not found", http.StatusNotFound)
				return
			}
			h.renderDetailError(w, r, collaborationID, http.StatusNotFound, "Participant not found", nil, nil, nil, nil)
			return
		}
		h.logger.Error("failed to remove collaboration participant", "collaboration_id", collaborationID, "profile_id", profileID, "error", err)
		if h.wantsJSON(r) {
			h.renderError(w, r, "Failed to remove participant", http.StatusInternalServerError)
			return
		}
		h.renderDetailError(w, r, collaborationID, http.StatusInternalServerError, "Failed to remove participant", nil, nil, nil, nil)
		return
	}

	h.logger.Info("collaboration participant removed", "collaboration_id", collaborationID, "profile_id", profileID)
	if h.metrics != nil {
		h.metrics.RecordCollaborationUnlink(true)
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "removed"})
		return
	}

	http.Redirect(w, r, "/collaborations/"+strconv.FormatInt(collaborationID, 10), http.StatusSeeOther)
}

func (h *CollaborationHandler) loadCollaborationDetail(ctx context.Context, collaborationID int64) (*repository.Collaboration, []repository.Profile, []repository.Profile, error) {
	c, err := h.collabs.Get(ctx, collaborationID)
	if err != nil {
		return nil, nil, nil, err
	}

	participants, err := h.collabs.ListParticipants(ctx, collaborationID)
	if err != nil {
		return nil, nil, nil, err
	}

	allProfiles, err := h.profiles.List(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	return c, participants, allProfiles, nil
}

func (h *CollaborationHandler) collaborationDetailTemplateData(
	c *repository.Collaboration,
	participants []repository.Profile,
	allProfiles []repository.Profile,
	errorMessage string,
	formName *string,
	formDescription *string,
	formSharedChat *bool,
	selectedProfileID *int64,
) map[string]any {
	participantDTOs := make([]dto.Profile, 0, len(participants))
	for _, p := range participants {
		participantDTOs = append(participantDTOs, dto.Profile{ID: p.ID, Name: p.Name, Description: p.Description, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt})
	}

	allProfileDTOs := make([]dto.Profile, 0, len(allProfiles))
	for _, p := range allProfiles {
		allProfileDTOs = append(allProfileDTOs, dto.Profile{ID: p.ID, Name: p.Name, Description: p.Description, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt})
	}

	name := c.Name
	if formName != nil {
		name = *formName
	}
	description := c.Description
	if formDescription != nil {
		description = *formDescription
	}
	sharedChat := c.SharedChat
	if formSharedChat != nil {
		sharedChat = *formSharedChat
	}

	selected := int64(0)
	if selectedProfileID != nil {
		selected = *selectedProfileID
	}

	return map[string]any{
		"Collaboration": dto.Collaboration{
			ID:          c.ID,
			Name:        c.Name,
			Description: c.Description,
			SharedChat:  c.SharedChat,
			CreatedAt:   c.CreatedAt,
			UpdatedAt:   c.UpdatedAt,
		},
		"Participants":      participantDTOs,
		"ParticipantsEmpty": len(participantDTOs) == 0,
		"AllProfiles":       allProfileDTOs,
		"ErrorMessage":      errorMessage,
		"FormName":          name,
		"FormDescription":   description,
		"FormSharedChat":    sharedChat,
		"SelectedProfileID": selected,
	}
}

func (h *CollaborationHandler) renderDetailTemplate(w http.ResponseWriter, data map[string]any, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := h.templates.ExecuteTemplate(w, "collaborations/detail", data); err != nil {
		h.logger.Error("failed to execute collaborations/detail template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *CollaborationHandler) renderCreateFormError(w http.ResponseWriter, r *http.Request, message string, req dto.CreateCollaborationRequest) {
	if h.wantsJSON(r) {
		h.renderError(w, r, message, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	if err := h.templates.ExecuteTemplate(w, "collaborations/new", map[string]any{
		"ErrorMessage":    message,
		"FormName":        req.Name,
		"FormDescription": req.Description,
		"FormSharedChat":  req.SharedChat,
	}); err != nil {
		h.logger.Error("failed to execute collaborations/new template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *CollaborationHandler) renderDetailError(
	w http.ResponseWriter,
	r *http.Request,
	collaborationID int64,
	status int,
	message string,
	formName *string,
	formDescription *string,
	formSharedChat *bool,
	selectedProfileID *int64,
) {
	ctx := r.Context()

	c, participants, allProfiles, err := h.loadCollaborationDetail(ctx, collaborationID)
	if err != nil {
		if err == services.ErrCollaborationNotFound {
			h.renderError(w, r, "Collaboration not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to load collaboration detail", "collaboration_id", collaborationID, "error", err)
		h.renderError(w, r, "Failed to load collaboration", http.StatusInternalServerError)
		return
	}

	data := h.collaborationDetailTemplateData(c, participants, allProfiles, message, formName, formDescription, formSharedChat, selectedProfileID)
	h.renderDetailTemplate(w, data, status)
}

func (h *CollaborationHandler) renderError(w http.ResponseWriter, r *http.Request, message string, status int) {
	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]string{"error": message})
		return
	}

	w.WriteHeader(status)
	if err := h.templates.ExecuteTemplate(w, "error.html", map[string]any{
		"Title":   http.StatusText(status),
		"Message": message,
	}); err != nil {
		h.logger.Error("failed to execute error template", "error", err)
	}
}

func (h *CollaborationHandler) wantsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/json")
}

func (h *CollaborationHandler) pathInt64(r *http.Request, name string) (int64, error) {
	v := r.PathValue(name)
	return strconv.ParseInt(v, 10, 64)
}
