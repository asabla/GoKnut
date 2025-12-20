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

// ProfileHandler handles profile-related HTTP requests.
type ProfileHandler struct {
	profiles  *services.ProfileService
	channels  *repository.ChannelRepository
	templates *template.Template
	logger    *observability.Logger
}

// NewProfileHandler creates a new profile handler.
func NewProfileHandler(
	profiles *services.ProfileService,
	channels *repository.ChannelRepository,
	templates *template.Template,
	logger *observability.Logger,
) *ProfileHandler {
	return &ProfileHandler{
		profiles:  profiles,
		channels:  channels,
		templates: templates,
		logger:    logger,
	}
}

// RegisterRoutes registers profile routes on the mux.
func (h *ProfileHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /profiles", h.handleList)
	mux.HandleFunc("GET /profiles/new", h.handleNew)
	mux.HandleFunc("POST /profiles", h.handleCreate)
	mux.HandleFunc("GET /profiles/{id}", h.handleGet)
	mux.HandleFunc("POST /profiles/{id}", h.handleUpdate)
	mux.HandleFunc("POST /profiles/{id}/delete", h.handleDelete)
	mux.HandleFunc("POST /profiles/{id}/channels", h.handleLinkChannel)
	mux.HandleFunc("POST /profiles/{id}/channels/{channel_id}/remove", h.handleUnlinkChannel)
}

func (h *ProfileHandler) handleList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	profiles, err := h.profiles.List(ctx)
	if err != nil {
		h.logger.Error("failed to list profiles", "error", err)
		h.renderError(w, r, "Failed to load profiles", http.StatusInternalServerError)
		return
	}

	profileDTOs := make([]dto.Profile, 0, len(profiles))
	for _, p := range profiles {
		profileDTOs = append(profileDTOs, dto.Profile{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		})
	}

	data := map[string]any{
		"Profiles": profileDTOs,
		"IsEmpty":  len(profileDTOs) == 0,
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "profiles/index", data); err != nil {
		h.logger.Error("failed to execute profiles/index template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *ProfileHandler) handleNew(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "profiles/new", map[string]any{
		"FormName":        "",
		"FormDescription": "",
	}); err != nil {
		h.logger.Error("failed to execute profiles/new template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *ProfileHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dto.CreateProfileRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.renderError(w, r, "Invalid request body", http.StatusBadRequest)
			return
		}
	} else {
		_ = r.ParseForm()
		req.Name = r.FormValue("name")
		req.Description = r.FormValue("description")
	}

	if err := req.Validate(); err != nil {
		h.renderCreateFormError(w, r, err.Error(), req)
		return
	}

	p, err := h.profiles.Create(ctx, req.Name, req.Description)
	if err != nil {
		h.logger.Error("failed to create profile", "error", err)
		h.renderCreateFormError(w, r, "Failed to create profile", req)
		return
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(dto.Profile{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		})
		return
	}

	http.Redirect(w, r, "/profiles/"+strconv.FormatInt(p.ID, 10), http.StatusSeeOther)
}

func (h *ProfileHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	profileID, err := h.pathInt64(r, "id")
	if err != nil {
		h.renderError(w, r, "Invalid profile id", http.StatusBadRequest)
		return
	}

	p, linked, allChannels, err := h.loadProfileDetail(ctx, profileID)
	if err != nil {
		if err == services.ErrProfileNotFound {
			h.renderError(w, r, "Profile not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to load profile detail", "profile_id", profileID, "error", err)
		h.renderError(w, r, "Failed to load profile", http.StatusInternalServerError)
		return
	}

	data := h.profileDetailTemplateData(p, linked, allChannels, "", nil, nil, nil)

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
		return
	}

	h.renderDetailTemplate(w, data, http.StatusOK)
}

func (h *ProfileHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	profileID, err := h.pathInt64(r, "id")
	if err != nil {
		h.renderError(w, r, "Invalid profile id", http.StatusBadRequest)
		return
	}

	var req dto.UpdateProfileRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.renderError(w, r, "Invalid request body", http.StatusBadRequest)
			return
		}
	} else {
		_ = r.ParseForm()
		req.Name = r.FormValue("name")
		req.Description = r.FormValue("description")
	}

	if err := req.Validate(); err != nil {
		if h.wantsJSON(r) {
			h.renderError(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		h.renderDetailError(w, r, profileID, http.StatusBadRequest, err.Error(), &req.Name, &req.Description, nil)
		return
	}

	_, err = h.profiles.Update(ctx, profileID, req.Name, req.Description)
	if err != nil {
		switch {
		case err == services.ErrProfileNotFound:
			h.renderError(w, r, "Profile not found", http.StatusNotFound)
			return
		case err == services.ErrInvalidProfileName:
			if h.wantsJSON(r) {
				h.renderError(w, r, dto.ErrProfileNameRequired.Error(), http.StatusBadRequest)
				return
			}
			h.renderDetailError(w, r, profileID, http.StatusBadRequest, dto.ErrProfileNameRequired.Error(), &req.Name, &req.Description, nil)
			return
		default:
			h.logger.Error("failed to update profile", "profile_id", profileID, "error", err)
			if h.wantsJSON(r) {
				h.renderError(w, r, "Failed to update profile", http.StatusInternalServerError)
				return
			}
			h.renderDetailError(w, r, profileID, http.StatusInternalServerError, "Failed to update profile", &req.Name, &req.Description, nil)
			return
		}
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
		return
	}

	http.Redirect(w, r, "/profiles/"+strconv.FormatInt(profileID, 10), http.StatusSeeOther)
}

func (h *ProfileHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	profileID, err := h.pathInt64(r, "id")
	if err != nil {
		h.renderError(w, r, "Invalid profile id", http.StatusBadRequest)
		return
	}

	if err := h.profiles.Delete(ctx, profileID); err != nil {
		if err == services.ErrProfileNotFound {
			h.renderError(w, r, "Profile not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to delete profile", "profile_id", profileID, "error", err)
		if h.wantsJSON(r) {
			h.renderError(w, r, "Failed to delete profile", http.StatusInternalServerError)
			return
		}

		h.renderDetailError(w, r, profileID, http.StatusInternalServerError, "Failed to delete profile", nil, nil, nil)
		return
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
		return
	}

	http.Redirect(w, r, "/profiles", http.StatusSeeOther)
}

func (h *ProfileHandler) handleLinkChannel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	profileID, err := h.pathInt64(r, "id")
	if err != nil {
		h.renderError(w, r, "Invalid profile id", http.StatusBadRequest)
		return
	}

	var req dto.LinkProfileChannelRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.renderError(w, r, "Invalid request body", http.StatusBadRequest)
			return
		}
	} else {
		_ = r.ParseForm()
		if r.FormValue("channel_id") == "" {
			if h.wantsJSON(r) {
				h.renderError(w, r, dto.ErrProfileChannelRequired.Error(), http.StatusBadRequest)
				return
			}
			h.renderDetailError(w, r, profileID, http.StatusBadRequest, dto.ErrProfileChannelRequired.Error(), nil, nil, nil)
			return
		}
		channelID, err := strconv.ParseInt(r.FormValue("channel_id"), 10, 64)
		if err != nil {
			if h.wantsJSON(r) {
				h.renderError(w, r, dto.ErrProfileChannelIDInvalid.Error(), http.StatusBadRequest)
				return
			}
			h.renderDetailError(w, r, profileID, http.StatusBadRequest, dto.ErrProfileChannelIDInvalid.Error(), nil, nil, nil)
			return
		}
		req.ChannelID = channelID
	}

	if err := req.Validate(); err != nil {
		if h.wantsJSON(r) {
			h.renderError(w, r, err.Error(), http.StatusBadRequest)
			return
		}
		selected := req.ChannelID
		h.renderDetailError(w, r, profileID, http.StatusBadRequest, err.Error(), nil, nil, &selected)
		return
	}

	err = h.profiles.LinkChannel(ctx, profileID, req.ChannelID)
	if err != nil {
		switch {
		case err == services.ErrProfileNotFound:
			h.renderError(w, r, "Profile not found", http.StatusNotFound)
			return
		case err == services.ErrChannelAlreadyLinked:
			if h.wantsJSON(r) {
				h.renderError(w, r, err.Error(), http.StatusConflict)
				return
			}
			selected := req.ChannelID
			h.renderDetailError(w, r, profileID, http.StatusConflict, "Channel is already linked to a profile", nil, nil, &selected)
			return
		case errors.Is(err, repository.ErrNotFound):
			if h.wantsJSON(r) {
				h.renderError(w, r, "Channel not found", http.StatusNotFound)
				return
			}
			selected := req.ChannelID
			h.renderDetailError(w, r, profileID, http.StatusNotFound, "Channel not found", nil, nil, &selected)
			return
		default:
			h.logger.Error("failed to link channel", "profile_id", profileID, "channel_id", req.ChannelID, "error", err)
			if h.wantsJSON(r) {
				h.renderError(w, r, "Failed to link channel", http.StatusInternalServerError)
				return
			}
			selected := req.ChannelID
			h.renderDetailError(w, r, profileID, http.StatusInternalServerError, "Failed to link channel", nil, nil, &selected)
			return
		}
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "linked"})
		return
	}

	http.Redirect(w, r, "/profiles/"+strconv.FormatInt(profileID, 10), http.StatusSeeOther)
}

func (h *ProfileHandler) handleUnlinkChannel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	profileID, err := h.pathInt64(r, "id")
	if err != nil {
		h.renderError(w, r, "Invalid profile id", http.StatusBadRequest)
		return
	}

	channelID, err := h.pathInt64(r, "channel_id")
	if err != nil {
		h.renderError(w, r, "Invalid channel id", http.StatusBadRequest)
		return
	}

	if err := h.profiles.UnlinkChannel(ctx, profileID, channelID); err != nil {
		if err == services.ErrProfileChannelNotFound {
			if h.wantsJSON(r) {
				h.renderError(w, r, "Channel link not found", http.StatusNotFound)
				return
			}
			h.renderDetailError(w, r, profileID, http.StatusNotFound, "Channel link not found", nil, nil, nil)
			return
		}
		h.logger.Error("failed to unlink channel", "profile_id", profileID, "channel_id", channelID, "error", err)
		if h.wantsJSON(r) {
			h.renderError(w, r, "Failed to remove channel link", http.StatusInternalServerError)
			return
		}
		h.renderDetailError(w, r, profileID, http.StatusInternalServerError, "Failed to remove channel link", nil, nil, nil)
		return
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "unlinked"})
		return
	}

	http.Redirect(w, r, "/profiles/"+strconv.FormatInt(profileID, 10), http.StatusSeeOther)
}

func (h *ProfileHandler) renderCreateFormError(w http.ResponseWriter, r *http.Request, message string, req dto.CreateProfileRequest) {
	if h.wantsJSON(r) {
		h.renderError(w, r, message, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	if err := h.templates.ExecuteTemplate(w, "profiles/new", map[string]any{
		"ErrorMessage":    message,
		"FormName":        req.Name,
		"FormDescription": req.Description,
	}); err != nil {
		h.logger.Error("failed to execute profiles/new template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *ProfileHandler) renderError(w http.ResponseWriter, r *http.Request, message string, status int) {
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

func (h *ProfileHandler) wantsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/json")
}

func (h *ProfileHandler) pathInt64(r *http.Request, name string) (int64, error) {
	v := r.PathValue(name)
	return strconv.ParseInt(v, 10, 64)
}

func (h *ProfileHandler) loadProfileDetail(ctx context.Context, profileID int64) (*repository.Profile, []repository.Channel, []repository.Channel, error) {
	p, err := h.profiles.Get(ctx, profileID)
	if err != nil {
		return nil, nil, nil, err
	}

	linked, err := h.profiles.ListLinkedChannels(ctx, profileID)
	if err != nil {
		return nil, nil, nil, err
	}

	allChannels, err := h.channels.List(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	return p, linked, allChannels, nil
}

func (h *ProfileHandler) profileDetailTemplateData(
	p *repository.Profile,
	linked []repository.Channel,
	allChannels []repository.Channel,
	errorMessage string,
	formName *string,
	formDescription *string,
	selectedChannelID *int64,
) map[string]any {
	linkedDTOs := make([]dto.Channel, 0, len(linked))
	for _, ch := range linked {
		linkedDTOs = append(linkedDTOs, dto.Channel{
			ID:                    ch.ID,
			Name:                  ch.Name,
			DisplayName:           ch.DisplayName,
			Enabled:               ch.Enabled,
			RetainHistoryOnDelete: ch.RetainHistoryOnDelete,
			CreatedAt:             ch.CreatedAt,
			UpdatedAt:             ch.UpdatedAt,
			LastMessageAt:         ch.LastMessageAt,
			TotalMessages:         ch.TotalMessages,
		})
	}

	allDTOs := make([]dto.Channel, 0, len(allChannels))
	for _, ch := range allChannels {
		allDTOs = append(allDTOs, dto.Channel{
			ID:                    ch.ID,
			Name:                  ch.Name,
			DisplayName:           ch.DisplayName,
			Enabled:               ch.Enabled,
			RetainHistoryOnDelete: ch.RetainHistoryOnDelete,
			CreatedAt:             ch.CreatedAt,
			UpdatedAt:             ch.UpdatedAt,
			LastMessageAt:         ch.LastMessageAt,
			TotalMessages:         ch.TotalMessages,
		})
	}

	name := p.Name
	if formName != nil {
		name = *formName
	}
	description := p.Description
	if formDescription != nil {
		description = *formDescription
	}

	selected := int64(0)
	if selectedChannelID != nil {
		selected = *selectedChannelID
	}

	return map[string]any{
		"Profile": dto.Profile{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		},
		"LinkedChannels":    linkedDTOs,
		"ChannelsEmpty":     len(linkedDTOs) == 0,
		"AllChannels":       allDTOs,
		"ErrorMessage":      errorMessage,
		"FormName":          name,
		"FormDescription":   description,
		"SelectedChannelID": selected,
	}
}

func (h *ProfileHandler) renderDetailError(
	w http.ResponseWriter,
	r *http.Request,
	profileID int64,
	status int,
	message string,
	formName *string,
	formDescription *string,
	selectedChannelID *int64,
) {
	ctx := r.Context()

	p, linked, allChannels, err := h.loadProfileDetail(ctx, profileID)
	if err != nil {
		if err == services.ErrProfileNotFound {
			h.renderError(w, r, "Profile not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to load profile detail", "profile_id", profileID, "error", err)
		h.renderError(w, r, "Failed to load profile", http.StatusInternalServerError)
		return
	}

	data := h.profileDetailTemplateData(p, linked, allChannels, message, formName, formDescription, selectedChannelID)
	h.renderDetailTemplate(w, data, status)
}

func (h *ProfileHandler) renderDetailTemplate(w http.ResponseWriter, data map[string]any, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := h.templates.ExecuteTemplate(w, "profiles/detail", data); err != nil {
		h.logger.Error("failed to execute profiles/detail template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
