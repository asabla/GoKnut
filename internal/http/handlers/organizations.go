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

// OrganizationHandler handles organization-related HTTP requests.
type OrganizationHandler struct {
	orgs      *services.OrganizationService
	profiles  *repository.ProfileRepository
	templates *template.Template
	logger    *observability.Logger
	metrics   *observability.Metrics
}

// NewOrganizationHandler creates a new organization handler.
func NewOrganizationHandler(
	orgs *services.OrganizationService,
	profiles *repository.ProfileRepository,
	templates *template.Template,
	logger *observability.Logger,
	metrics *observability.Metrics,
) *OrganizationHandler {
	return &OrganizationHandler{orgs: orgs, profiles: profiles, templates: templates, logger: logger, metrics: metrics}
}

// RegisterRoutes registers organization routes on the mux.
func (h *OrganizationHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /organizations", h.handleList)
	mux.HandleFunc("GET /organizations/new", h.handleNew)
	mux.HandleFunc("POST /organizations", h.handleCreate)
	mux.HandleFunc("GET /organizations/{id}", h.handleGet)
	mux.HandleFunc("POST /organizations/{id}", h.handleUpdate)
	mux.HandleFunc("POST /organizations/{id}/delete", h.handleDelete)
	mux.HandleFunc("POST /organizations/{id}/members", h.handleAddMember)
	mux.HandleFunc("POST /organizations/{id}/members/{profile_id}/remove", h.handleRemoveMember)
}

func (h *OrganizationHandler) handleList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgs, err := h.orgs.List(ctx)
	if err != nil {
		h.logger.Error("failed to list organizations", "error", err)
		h.renderError(w, r, "Failed to load organizations", http.StatusInternalServerError)
		return
	}

	orgDTOs := make([]dto.Organization, 0, len(orgs))
	for _, org := range orgs {
		orgDTOs = append(orgDTOs, dto.Organization{
			ID:          org.ID,
			Name:        org.Name,
			Description: org.Description,
			CreatedAt:   org.CreatedAt,
			UpdatedAt:   org.UpdatedAt,
		})
	}

	data := map[string]any{
		"Organizations": orgDTOs,
		"IsEmpty":       len(orgDTOs) == 0,
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "organizations/index", data); err != nil {
		h.logger.Error("failed to execute organizations/index template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *OrganizationHandler) handleNew(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "organizations/new", map[string]any{
		"FormName":        "",
		"FormDescription": "",
	}); err != nil {
		h.logger.Error("failed to execute organizations/new template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *OrganizationHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dto.CreateOrganizationRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if h.metrics != nil {
				h.metrics.RecordOrganizationCreate(false)
			}
			h.renderError(w, r, "Invalid request body", http.StatusBadRequest)
			return
		}
	} else {
		_ = r.ParseForm()
		req.Name = r.FormValue("name")
		req.Description = r.FormValue("description")
	}

	if err := req.Validate(); err != nil {
		if h.metrics != nil {
			h.metrics.RecordOrganizationCreate(false)
		}
		h.renderCreateFormError(w, r, err.Error(), req)
		return
	}

	org, err := h.orgs.Create(ctx, req.Name, req.Description)
	if err != nil {
		if h.metrics != nil {
			h.metrics.RecordOrganizationCreate(false)
		}
		if errors.Is(err, services.ErrInvalidOrganizationName) {
			h.renderCreateFormError(w, r, dto.ErrOrganizationNameRequired.Error(), req)
			return
		}
		h.logger.Error("failed to create organization", "error", err)
		h.renderCreateFormError(w, r, "Failed to create organization", req)
		return
	}

	h.logger.Info("organization created", "organization_id", org.ID)
	if h.metrics != nil {
		h.metrics.RecordOrganizationCreate(true)
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(dto.Organization{
			ID:          org.ID,
			Name:        org.Name,
			Description: org.Description,
			CreatedAt:   org.CreatedAt,
			UpdatedAt:   org.UpdatedAt,
		})
		return
	}

	http.Redirect(w, r, "/organizations/"+strconv.FormatInt(org.ID, 10), http.StatusSeeOther)
}

func (h *OrganizationHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgID, err := h.pathInt64(r, "id")
	if err != nil {
		h.renderError(w, r, "Invalid organization id", http.StatusBadRequest)
		return
	}

	org, members, allProfiles, err := h.loadOrganizationDetail(ctx, orgID)
	if err != nil {
		switch {
		case err == services.ErrOrganizationNotFound:
			h.renderError(w, r, "Organization not found", http.StatusNotFound)
			return
		default:
			h.logger.Error("failed to load organization detail", "organization_id", orgID, "error", err)
			h.renderError(w, r, "Failed to load organization", http.StatusInternalServerError)
			return
		}
	}

	data := h.organizationDetailTemplateData(org, members, allProfiles, "", nil, nil, nil)

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
		return
	}

	h.renderDetailTemplate(w, data, http.StatusOK)
}

func (h *OrganizationHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgID, err := h.pathInt64(r, "id")
	if err != nil {
		h.renderError(w, r, "Invalid organization id", http.StatusBadRequest)
		return
	}

	var req dto.UpdateOrganizationRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if h.metrics != nil {
				h.metrics.RecordOrganizationUpdate(false)
			}
			h.renderError(w, r, "Invalid request body", http.StatusBadRequest)
			return
		}
	} else {
		_ = r.ParseForm()
		req.Name = r.FormValue("name")
		req.Description = r.FormValue("description")
	}

	if err := req.Validate(); err != nil {
		if h.metrics != nil {
			h.metrics.RecordOrganizationUpdate(false)
		}
		if h.wantsJSON(r) {
			h.renderError(w, r, err.Error(), http.StatusBadRequest)
			return
		}
		h.renderDetailError(w, r, orgID, http.StatusBadRequest, err.Error(), &req.Name, &req.Description, nil)
		return
	}

	_, err = h.orgs.Update(ctx, orgID, req.Name, req.Description)
	if err != nil {
		if h.metrics != nil {
			h.metrics.RecordOrganizationUpdate(false)
		}
		switch {
		case err == services.ErrOrganizationNotFound:
			h.renderError(w, r, "Organization not found", http.StatusNotFound)
			return
		case err == services.ErrInvalidOrganizationName:
			if h.wantsJSON(r) {
				h.renderError(w, r, dto.ErrOrganizationNameRequired.Error(), http.StatusBadRequest)
				return
			}
			h.renderDetailError(w, r, orgID, http.StatusBadRequest, dto.ErrOrganizationNameRequired.Error(), &req.Name, &req.Description, nil)
			return
		default:
			h.logger.Error("failed to update organization", "organization_id", orgID, "error", err)
			if h.wantsJSON(r) {
				h.renderError(w, r, "Failed to update organization", http.StatusInternalServerError)
				return
			}
			h.renderDetailError(w, r, orgID, http.StatusInternalServerError, "Failed to update organization", &req.Name, &req.Description, nil)
			return
		}
	}

	h.logger.Info("organization updated", "organization_id", orgID)
	if h.metrics != nil {
		h.metrics.RecordOrganizationUpdate(true)
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
		return
	}

	http.Redirect(w, r, "/organizations/"+strconv.FormatInt(orgID, 10), http.StatusSeeOther)
}

func (h *OrganizationHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgID, err := h.pathInt64(r, "id")
	if err != nil {
		h.renderError(w, r, "Invalid organization id", http.StatusBadRequest)
		return
	}

	if err := h.orgs.Delete(ctx, orgID); err != nil {
		if h.metrics != nil {
			h.metrics.RecordOrganizationDelete(false)
		}
		if err == services.ErrOrganizationNotFound {
			h.renderError(w, r, "Organization not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to delete organization", "organization_id", orgID, "error", err)
		if h.wantsJSON(r) {
			h.renderError(w, r, "Failed to delete organization", http.StatusInternalServerError)
			return
		}
		h.renderDetailError(w, r, orgID, http.StatusInternalServerError, "Failed to delete organization", nil, nil, nil)
		return
	}

	h.logger.Info("organization deleted", "organization_id", orgID)
	if h.metrics != nil {
		h.metrics.RecordOrganizationDelete(true)
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
		return
	}

	http.Redirect(w, r, "/organizations", http.StatusSeeOther)
}

func (h *OrganizationHandler) handleAddMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgID, err := h.pathInt64(r, "id")
	if err != nil {
		h.renderError(w, r, "Invalid organization id", http.StatusBadRequest)
		return
	}

	var req dto.AddOrganizationMemberRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if h.metrics != nil {
				h.metrics.RecordOrganizationLink(false)
			}
			h.renderError(w, r, "Invalid request body", http.StatusBadRequest)
			return
		}
	} else {
		_ = r.ParseForm()
		if r.FormValue("profile_id") == "" {
			if h.wantsJSON(r) {
				h.renderError(w, r, dto.ErrOrganizationMemberRequired.Error(), http.StatusBadRequest)
				return
			}
			h.renderDetailError(w, r, orgID, http.StatusBadRequest, dto.ErrOrganizationMemberRequired.Error(), nil, nil, nil)
			return
		}
		profileID, err := strconv.ParseInt(r.FormValue("profile_id"), 10, 64)
		if err != nil {
			if h.wantsJSON(r) {
				h.renderError(w, r, dto.ErrOrganizationMemberIDInvalid.Error(), http.StatusBadRequest)
				return
			}
			h.renderDetailError(w, r, orgID, http.StatusBadRequest, dto.ErrOrganizationMemberIDInvalid.Error(), nil, nil, nil)
			return
		}
		req.ProfileID = profileID
	}

	if err := req.Validate(); err != nil {
		if h.metrics != nil {
			h.metrics.RecordOrganizationLink(false)
		}
		if h.wantsJSON(r) {
			h.renderError(w, r, err.Error(), http.StatusBadRequest)
			return
		}
		selected := req.ProfileID
		h.renderDetailError(w, r, orgID, http.StatusBadRequest, err.Error(), nil, nil, &selected)
		return
	}

	err = h.orgs.AddMember(ctx, orgID, req.ProfileID)
	if err != nil {
		if h.metrics != nil {
			h.metrics.RecordOrganizationLink(false)
		}
		switch {
		case err == services.ErrOrganizationNotFound:
			h.renderError(w, r, "Organization not found", http.StatusNotFound)
			return
		case err == services.ErrProfileNotFound:
			h.renderError(w, r, "Profile not found", http.StatusNotFound)
			return
		case err == services.ErrMembershipAlreadyExists:
			if h.wantsJSON(r) {
				h.renderError(w, r, err.Error(), http.StatusConflict)
				return
			}
			selected := req.ProfileID
			h.renderDetailError(w, r, orgID, http.StatusConflict, "Profile is already a member", nil, nil, &selected)
			return
		default:
			h.logger.Error("failed to add organization member", "organization_id", orgID, "profile_id", req.ProfileID, "error", err)
			if h.wantsJSON(r) {
				h.renderError(w, r, "Failed to add member", http.StatusInternalServerError)
				return
			}
			selected := req.ProfileID
			h.renderDetailError(w, r, orgID, http.StatusInternalServerError, "Failed to add member", nil, nil, &selected)
			return
		}
	}

	h.logger.Info("organization member added", "organization_id", orgID, "profile_id", req.ProfileID)
	if h.metrics != nil {
		h.metrics.RecordOrganizationLink(true)
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "added"})
		return
	}

	http.Redirect(w, r, "/organizations/"+strconv.FormatInt(orgID, 10), http.StatusSeeOther)
}

func (h *OrganizationHandler) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgID, err := h.pathInt64(r, "id")
	if err != nil {
		h.renderError(w, r, "Invalid organization id", http.StatusBadRequest)
		return
	}
	profileID, err := h.pathInt64(r, "profile_id")
	if err != nil {
		h.renderError(w, r, "Invalid profile id", http.StatusBadRequest)
		return
	}

	if err := h.orgs.RemoveMember(ctx, orgID, profileID); err != nil {
		if h.metrics != nil {
			h.metrics.RecordOrganizationUnlink(false)
		}
		if err == services.ErrMembershipNotFound {
			h.renderError(w, r, "Membership not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to remove organization member", "organization_id", orgID, "profile_id", profileID, "error", err)
		if h.wantsJSON(r) {
			h.renderError(w, r, "Failed to remove member", http.StatusInternalServerError)
			return
		}
		h.renderDetailError(w, r, orgID, http.StatusInternalServerError, "Failed to remove member", nil, nil, nil)
		return
	}

	h.logger.Info("organization member removed", "organization_id", orgID, "profile_id", profileID)
	if h.metrics != nil {
		h.metrics.RecordOrganizationUnlink(true)
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "removed"})
		return
	}

	http.Redirect(w, r, "/organizations/"+strconv.FormatInt(orgID, 10), http.StatusSeeOther)
}

func (h *OrganizationHandler) loadOrganizationDetail(
	ctx context.Context,
	organizationID int64,
) (*repository.Organization, []repository.Profile, []repository.Profile, error) {
	org, err := h.orgs.Get(ctx, organizationID)
	if err != nil {
		return nil, nil, nil, err
	}

	members, err := h.orgs.ListMembers(ctx, organizationID)
	if err != nil {
		return nil, nil, nil, err
	}

	allProfiles, err := h.profiles.List(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	return org, members, allProfiles, nil
}

func (h *OrganizationHandler) organizationDetailTemplateData(
	org *repository.Organization,
	members []repository.Profile,
	allProfiles []repository.Profile,
	errorMessage string,
	formName *string,
	formDescription *string,
	selectedProfileID *int64,
) map[string]any {
	memberDTOs := make([]dto.Profile, 0, len(members))
	for _, p := range members {
		memberDTOs = append(memberDTOs, dto.Profile{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		})
	}

	profileDTOs := make([]dto.Profile, 0, len(allProfiles))
	for _, p := range allProfiles {
		profileDTOs = append(profileDTOs, dto.Profile{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		})
	}

	name := org.Name
	if formName != nil {
		name = *formName
	}
	description := org.Description
	if formDescription != nil {
		description = *formDescription
	}

	selected := int64(0)
	if selectedProfileID != nil {
		selected = *selectedProfileID
	}

	return map[string]any{
		"Organization": dto.Organization{
			ID:          org.ID,
			Name:        org.Name,
			Description: org.Description,
			CreatedAt:   org.CreatedAt,
			UpdatedAt:   org.UpdatedAt,
		},
		"Members":           memberDTOs,
		"MembersEmpty":      len(memberDTOs) == 0,
		"AllProfiles":       profileDTOs,
		"ErrorMessage":      errorMessage,
		"FormName":          name,
		"FormDescription":   description,
		"SelectedProfileID": selected,
	}
}

func (h *OrganizationHandler) renderCreateFormError(w http.ResponseWriter, r *http.Request, message string, req dto.CreateOrganizationRequest) {
	if h.wantsJSON(r) {
		h.renderError(w, r, message, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	if err := h.templates.ExecuteTemplate(w, "organizations/new", map[string]any{
		"ErrorMessage":    message,
		"FormName":        req.Name,
		"FormDescription": req.Description,
	}); err != nil {
		h.logger.Error("failed to execute organizations/new template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *OrganizationHandler) renderDetailError(
	w http.ResponseWriter,
	r *http.Request,
	organizationID int64,
	status int,
	message string,
	formName *string,
	formDescription *string,
	selectedProfileID *int64,
) {
	ctx := r.Context()

	org, members, allProfiles, err := h.loadOrganizationDetail(ctx, organizationID)
	if err != nil {
		if err == services.ErrOrganizationNotFound {
			h.renderError(w, r, "Organization not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to load organization detail", "organization_id", organizationID, "error", err)
		h.renderError(w, r, "Failed to load organization", http.StatusInternalServerError)
		return
	}

	data := h.organizationDetailTemplateData(org, members, allProfiles, message, formName, formDescription, selectedProfileID)
	h.renderDetailTemplate(w, data, status)
}

func (h *OrganizationHandler) renderDetailTemplate(w http.ResponseWriter, data map[string]any, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := h.templates.ExecuteTemplate(w, "organizations/detail", data); err != nil {
		h.logger.Error("failed to execute organizations/detail template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *OrganizationHandler) renderError(w http.ResponseWriter, r *http.Request, message string, status int) {
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

func (h *OrganizationHandler) wantsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/json")
}

func (h *OrganizationHandler) pathInt64(r *http.Request, name string) (int64, error) {
	v := r.PathValue(name)
	return strconv.ParseInt(v, 10, 64)
}
