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
	"time"

	"github.com/asabla/goknut/internal/http/dto"
	"github.com/asabla/goknut/internal/observability"
	"github.com/asabla/goknut/internal/repository"
	"github.com/asabla/goknut/internal/services"
)

const datetimeLocalLayout = "2006-01-02T15:04"

// EventHandler handles event-related HTTP requests.
type EventHandler struct {
	events    *services.EventService
	profiles  *repository.ProfileRepository
	templates *template.Template
	logger    *observability.Logger
}

// NewEventHandler creates a new event handler.
func NewEventHandler(
	events *services.EventService,
	profiles *repository.ProfileRepository,
	templates *template.Template,
	logger *observability.Logger,
) *EventHandler {
	return &EventHandler{events: events, profiles: profiles, templates: templates, logger: logger}
}

// RegisterRoutes registers event routes on the mux.
func (h *EventHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /events", h.handleList)
	mux.HandleFunc("GET /events/new", h.handleNew)
	mux.HandleFunc("POST /events", h.handleCreate)
	mux.HandleFunc("GET /events/{id}", h.handleGet)
	mux.HandleFunc("POST /events/{id}", h.handleUpdate)
	mux.HandleFunc("POST /events/{id}/delete", h.handleDelete)
	mux.HandleFunc("POST /events/{id}/participants", h.handleAddParticipant)
	mux.HandleFunc("POST /events/{id}/participants/{profile_id}/remove", h.handleRemoveParticipant)
}

func (h *EventHandler) handleList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	events, err := h.events.List(ctx)
	if err != nil {
		h.logger.Error("failed to list events", "error", err)
		h.renderError(w, r, "Failed to load events", http.StatusInternalServerError)
		return
	}

	eventDTOs := make([]dto.Event, 0, len(events))
	for _, evt := range events {
		eventDTOs = append(eventDTOs, dto.Event{
			ID:          evt.ID,
			Title:       evt.Title,
			Description: evt.Description,
			StartAt:     evt.StartAt,
			EndAt:       evt.EndAt,
			CreatedAt:   evt.CreatedAt,
			UpdatedAt:   evt.UpdatedAt,
		})
	}

	data := map[string]any{
		"Events":  eventDTOs,
		"IsEmpty": len(eventDTOs) == 0,
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "events/index", data); err != nil {
		h.logger.Error("failed to execute events/index template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *EventHandler) handleNew(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "events/new", map[string]any{
		"FormTitle":       "",
		"FormDescription": "",
		"FormStartAt":     "",
		"FormEndAt":       "",
	}); err != nil {
		h.logger.Error("failed to execute events/new template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *EventHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dto.CreateEventRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.renderError(w, r, "Invalid request body", http.StatusBadRequest)
			return
		}
	} else {
		_ = r.ParseForm()
		req.Title = r.FormValue("title")
		req.Description = r.FormValue("description")

		startAt, endAt, err := parseEventDatesFromForm(r)
		if err != nil {
			h.renderCreateFormError(w, r, err.Error(), req, r.FormValue("start_at"), r.FormValue("end_at"))
			return
		}
		req.StartAt = startAt
		req.EndAt = endAt
	}

	if err := req.Validate(); err != nil {
		h.renderCreateFormError(w, r, err.Error(), req, formatDateTimeLocal(req.StartAt), formatDateTimeLocalPtr(req.EndAt))
		return
	}

	evt, err := h.events.Create(ctx, req.Title, req.Description, req.StartAt, req.EndAt)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidEventTitle):
			h.renderCreateFormError(w, r, dto.ErrEventTitleRequired.Error(), req, formatDateTimeLocal(req.StartAt), formatDateTimeLocalPtr(req.EndAt))
			return
		case errors.Is(err, services.ErrInvalidEventDates):
			h.renderCreateFormError(w, r, dto.ErrEventDatesInvalid.Error(), req, formatDateTimeLocal(req.StartAt), formatDateTimeLocalPtr(req.EndAt))
			return
		default:
			h.logger.Error("failed to create event", "error", err)
			h.renderCreateFormError(w, r, "Failed to create event", req, formatDateTimeLocal(req.StartAt), formatDateTimeLocalPtr(req.EndAt))
			return
		}
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(dto.Event{
			ID:          evt.ID,
			Title:       evt.Title,
			Description: evt.Description,
			StartAt:     evt.StartAt,
			EndAt:       evt.EndAt,
			CreatedAt:   evt.CreatedAt,
			UpdatedAt:   evt.UpdatedAt,
		})
		return
	}

	http.Redirect(w, r, "/events/"+strconv.FormatInt(evt.ID, 10), http.StatusSeeOther)
}

func (h *EventHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	eventID, err := h.pathInt64(r, "id")
	if err != nil {
		h.renderError(w, r, "Invalid event id", http.StatusBadRequest)
		return
	}

	evt, participants, allProfiles, err := h.loadEventDetail(ctx, eventID)
	if err != nil {
		switch {
		case err == services.ErrEventNotFound:
			h.renderError(w, r, "Event not found", http.StatusNotFound)
			return
		default:
			h.logger.Error("failed to load event detail", "event_id", eventID, "error", err)
			h.renderError(w, r, "Failed to load event", http.StatusInternalServerError)
			return
		}
	}

	data := h.eventDetailTemplateData(evt, participants, allProfiles, "", nil, nil, nil, nil, nil)

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
		return
	}

	h.renderDetailTemplate(w, data, http.StatusOK)
}

func (h *EventHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	eventID, err := h.pathInt64(r, "id")
	if err != nil {
		h.renderError(w, r, "Invalid event id", http.StatusBadRequest)
		return
	}

	var req dto.UpdateEventRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.renderError(w, r, "Invalid request body", http.StatusBadRequest)
			return
		}
	} else {
		_ = r.ParseForm()
		req.Title = r.FormValue("title")
		req.Description = r.FormValue("description")

		startAt, endAt, err := parseEventDatesFromForm(r)
		if err != nil {
			h.renderDetailError(w, r, eventID, http.StatusBadRequest, err.Error(), &req.Title, &req.Description, ptrString(r.FormValue("start_at")), ptrString(r.FormValue("end_at")), nil)
			return
		}
		req.StartAt = startAt
		req.EndAt = endAt
	}

	if err := req.Validate(); err != nil {
		if h.wantsJSON(r) {
			h.renderError(w, r, err.Error(), http.StatusBadRequest)
			return
		}
		start := formatDateTimeLocal(req.StartAt)
		end := formatDateTimeLocalPtr(req.EndAt)
		h.renderDetailError(w, r, eventID, http.StatusBadRequest, err.Error(), &req.Title, &req.Description, &start, &end, nil)
		return
	}

	_, err = h.events.Update(ctx, eventID, req.Title, req.Description, req.StartAt, req.EndAt)
	if err != nil {
		switch {
		case err == services.ErrEventNotFound:
			h.renderError(w, r, "Event not found", http.StatusNotFound)
			return
		case errors.Is(err, services.ErrInvalidEventTitle):
			if h.wantsJSON(r) {
				h.renderError(w, r, dto.ErrEventTitleRequired.Error(), http.StatusBadRequest)
				return
			}
			start := formatDateTimeLocal(req.StartAt)
			end := formatDateTimeLocalPtr(req.EndAt)
			h.renderDetailError(w, r, eventID, http.StatusBadRequest, dto.ErrEventTitleRequired.Error(), &req.Title, &req.Description, &start, &end, nil)
			return
		case errors.Is(err, services.ErrInvalidEventDates):
			if h.wantsJSON(r) {
				h.renderError(w, r, dto.ErrEventDatesInvalid.Error(), http.StatusBadRequest)
				return
			}
			start := formatDateTimeLocal(req.StartAt)
			end := formatDateTimeLocalPtr(req.EndAt)
			h.renderDetailError(w, r, eventID, http.StatusBadRequest, dto.ErrEventDatesInvalid.Error(), &req.Title, &req.Description, &start, &end, nil)
			return
		default:
			h.logger.Error("failed to update event", "event_id", eventID, "error", err)
			if h.wantsJSON(r) {
				h.renderError(w, r, "Failed to update event", http.StatusInternalServerError)
				return
			}
			start := formatDateTimeLocal(req.StartAt)
			end := formatDateTimeLocalPtr(req.EndAt)
			h.renderDetailError(w, r, eventID, http.StatusInternalServerError, "Failed to update event", &req.Title, &req.Description, &start, &end, nil)
			return
		}
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
		return
	}

	http.Redirect(w, r, "/events/"+strconv.FormatInt(eventID, 10), http.StatusSeeOther)
}

func (h *EventHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	eventID, err := h.pathInt64(r, "id")
	if err != nil {
		h.renderError(w, r, "Invalid event id", http.StatusBadRequest)
		return
	}

	if err := h.events.Delete(ctx, eventID); err != nil {
		if err == services.ErrEventNotFound {
			h.renderError(w, r, "Event not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to delete event", "event_id", eventID, "error", err)
		if h.wantsJSON(r) {
			h.renderError(w, r, "Failed to delete event", http.StatusInternalServerError)
			return
		}
		h.renderDetailError(w, r, eventID, http.StatusInternalServerError, "Failed to delete event", nil, nil, nil, nil, nil)
		return
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
		return
	}

	http.Redirect(w, r, "/events", http.StatusSeeOther)
}

func (h *EventHandler) handleAddParticipant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	eventID, err := h.pathInt64(r, "id")
	if err != nil {
		h.renderError(w, r, "Invalid event id", http.StatusBadRequest)
		return
	}

	var req dto.AddEventParticipantRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.renderError(w, r, "Invalid request body", http.StatusBadRequest)
			return
		}
	} else {
		_ = r.ParseForm()
		if r.FormValue("profile_id") == "" {
			if h.wantsJSON(r) {
				h.renderError(w, r, dto.ErrEventParticipantRequired.Error(), http.StatusBadRequest)
				return
			}
			h.renderDetailError(w, r, eventID, http.StatusBadRequest, dto.ErrEventParticipantRequired.Error(), nil, nil, nil, nil, nil)
			return
		}
		profileID, err := strconv.ParseInt(r.FormValue("profile_id"), 10, 64)
		if err != nil {
			if h.wantsJSON(r) {
				h.renderError(w, r, dto.ErrEventParticipantIDInvalid.Error(), http.StatusBadRequest)
				return
			}
			h.renderDetailError(w, r, eventID, http.StatusBadRequest, dto.ErrEventParticipantIDInvalid.Error(), nil, nil, nil, nil, nil)
			return
		}
		req.ProfileID = profileID
	}

	if err := req.Validate(); err != nil {
		if h.wantsJSON(r) {
			h.renderError(w, r, err.Error(), http.StatusBadRequest)
			return
		}
		selected := req.ProfileID
		h.renderDetailError(w, r, eventID, http.StatusBadRequest, err.Error(), nil, nil, nil, nil, &selected)
		return
	}

	err = h.events.AddParticipant(ctx, eventID, req.ProfileID)
	if err != nil {
		switch {
		case err == services.ErrEventNotFound:
			h.renderError(w, r, "Event not found", http.StatusNotFound)
			return
		case err == services.ErrProfileNotFound:
			h.renderError(w, r, "Profile not found", http.StatusNotFound)
			return
		case err == services.ErrParticipantExists:
			if h.wantsJSON(r) {
				h.renderError(w, r, err.Error(), http.StatusConflict)
				return
			}
			selected := req.ProfileID
			h.renderDetailError(w, r, eventID, http.StatusConflict, "Profile is already a participant", nil, nil, nil, nil, &selected)
			return
		default:
			h.logger.Error("failed to add event participant", "event_id", eventID, "profile_id", req.ProfileID, "error", err)
			if h.wantsJSON(r) {
				h.renderError(w, r, "Failed to add participant", http.StatusInternalServerError)
				return
			}
			selected := req.ProfileID
			h.renderDetailError(w, r, eventID, http.StatusInternalServerError, "Failed to add participant", nil, nil, nil, nil, &selected)
			return
		}
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "added"})
		return
	}

	http.Redirect(w, r, "/events/"+strconv.FormatInt(eventID, 10), http.StatusSeeOther)
}

func (h *EventHandler) handleRemoveParticipant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	eventID, err := h.pathInt64(r, "id")
	if err != nil {
		h.renderError(w, r, "Invalid event id", http.StatusBadRequest)
		return
	}
	profileID, err := h.pathInt64(r, "profile_id")
	if err != nil {
		h.renderError(w, r, "Invalid profile id", http.StatusBadRequest)
		return
	}

	if err := h.events.RemoveParticipant(ctx, eventID, profileID); err != nil {
		if err == services.ErrParticipantMissing {
			h.renderError(w, r, "Participant not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to remove event participant", "event_id", eventID, "profile_id", profileID, "error", err)
		if h.wantsJSON(r) {
			h.renderError(w, r, "Failed to remove participant", http.StatusInternalServerError)
			return
		}
		h.renderDetailError(w, r, eventID, http.StatusInternalServerError, "Failed to remove participant", nil, nil, nil, nil, nil)
		return
	}

	if h.wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "removed"})
		return
	}

	http.Redirect(w, r, "/events/"+strconv.FormatInt(eventID, 10), http.StatusSeeOther)
}

func (h *EventHandler) loadEventDetail(
	ctx context.Context,
	eventID int64,
) (*repository.Event, []repository.Profile, []repository.Profile, error) {
	evt, err := h.events.Get(ctx, eventID)
	if err != nil {
		return nil, nil, nil, err
	}

	participants, err := h.events.ListParticipants(ctx, eventID)
	if err != nil {
		return nil, nil, nil, err
	}

	allProfiles, err := h.profiles.List(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	return evt, participants, allProfiles, nil
}

func (h *EventHandler) eventDetailTemplateData(
	evt *repository.Event,
	participants []repository.Profile,
	allProfiles []repository.Profile,
	errorMessage string,
	formTitle *string,
	formDescription *string,
	formStartAt *string,
	formEndAt *string,
	selectedProfileID *int64,
) map[string]any {
	participantDTOs := make([]dto.Profile, 0, len(participants))
	for _, p := range participants {
		participantDTOs = append(participantDTOs, dto.Profile{
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

	title := evt.Title
	if formTitle != nil {
		title = *formTitle
	}
	description := evt.Description
	if formDescription != nil {
		description = *formDescription
	}

	start := formatDateTimeLocal(evt.StartAt)
	if formStartAt != nil {
		start = *formStartAt
	}

	end := formatDateTimeLocalPtr(evt.EndAt)
	if formEndAt != nil {
		end = *formEndAt
	}

	selected := int64(0)
	if selectedProfileID != nil {
		selected = *selectedProfileID
	}

	return map[string]any{
		"Event": dto.Event{
			ID:          evt.ID,
			Title:       evt.Title,
			Description: evt.Description,
			StartAt:     evt.StartAt,
			EndAt:       evt.EndAt,
			CreatedAt:   evt.CreatedAt,
			UpdatedAt:   evt.UpdatedAt,
		},
		"Participants":      participantDTOs,
		"ParticipantsEmpty": len(participantDTOs) == 0,
		"AllProfiles":       profileDTOs,
		"ErrorMessage":      errorMessage,
		"FormTitle":         title,
		"FormDescription":   description,
		"FormStartAt":       start,
		"FormEndAt":         end,
		"SelectedProfileID": selected,
	}
}

func (h *EventHandler) renderCreateFormError(w http.ResponseWriter, r *http.Request, message string, req dto.CreateEventRequest, startAtStr, endAtStr string) {
	if h.wantsJSON(r) {
		h.renderError(w, r, message, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	if err := h.templates.ExecuteTemplate(w, "events/new", map[string]any{
		"ErrorMessage":    message,
		"FormTitle":       req.Title,
		"FormDescription": req.Description,
		"FormStartAt":     startAtStr,
		"FormEndAt":       endAtStr,
	}); err != nil {
		h.logger.Error("failed to execute events/new template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *EventHandler) renderDetailError(
	w http.ResponseWriter,
	r *http.Request,
	eventID int64,
	status int,
	message string,
	formTitle *string,
	formDescription *string,
	formStartAt *string,
	formEndAt *string,
	selectedProfileID *int64,
) {
	ctx := r.Context()

	evt, participants, allProfiles, err := h.loadEventDetail(ctx, eventID)
	if err != nil {
		h.logger.Error("failed to reload event detail", "event_id", eventID, "error", err)
		h.renderError(w, r, "Failed to load event", http.StatusInternalServerError)
		return
	}

	data := h.eventDetailTemplateData(evt, participants, allProfiles, message, formTitle, formDescription, formStartAt, formEndAt, selectedProfileID)

	if h.wantsJSON(r) {
		h.renderError(w, r, message, status)
		return
	}

	h.renderDetailTemplate(w, data, status)
}

func (h *EventHandler) renderDetailTemplate(w http.ResponseWriter, data map[string]any, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := h.templates.ExecuteTemplate(w, "events/detail", data); err != nil {
		h.logger.Error("failed to execute events/detail template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *EventHandler) wantsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/json")
}

func (h *EventHandler) pathInt64(r *http.Request, name string) (int64, error) {
	value := r.PathValue(name)
	return strconv.ParseInt(value, 10, 64)
}

func (h *EventHandler) renderError(w http.ResponseWriter, r *http.Request, message string, status int) {
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

func parseEventDatesFromForm(r *http.Request) (time.Time, *time.Time, error) {
	startAtStr := strings.TrimSpace(r.FormValue("start_at"))
	if startAtStr == "" {
		return time.Time{}, nil, dto.ErrEventStartAtRequired
	}
	startAt, err := time.ParseInLocation(datetimeLocalLayout, startAtStr, time.Local)
	if err != nil {
		return time.Time{}, nil, dto.ErrEventStartAtRequired
	}

	endAtStr := strings.TrimSpace(r.FormValue("end_at"))
	if endAtStr == "" {
		return startAt, nil, nil
	}
	endAt, err := time.ParseInLocation(datetimeLocalLayout, endAtStr, time.Local)
	if err != nil {
		return time.Time{}, nil, dto.ErrEventDatesInvalid
	}

	return startAt, &endAt, nil
}

func formatDateTimeLocal(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(datetimeLocalLayout)
}

func formatDateTimeLocalPtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format(datetimeLocalLayout)
}

func ptrString(v string) *string {
	copy := v
	return &copy
}
