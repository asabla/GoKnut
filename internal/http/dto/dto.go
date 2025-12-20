// Package dto provides shared data transfer objects and validation helpers.
package dto

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// Validation errors
var (
	ErrChannelNameRequired = errors.New("channel name is required")
	ErrChannelNameInvalid  = errors.New("channel name contains invalid characters")
	ErrChannelNameTooLong  = errors.New("channel name is too long (max 25 characters)")
	ErrMessageTextRequired = errors.New("message text is required")
	ErrMessageTextTooLong  = errors.New("message text is too long (max 500 characters)")
	ErrUsernameRequired    = errors.New("username is required")
	ErrUsernameInvalid     = errors.New("username contains invalid characters")
	ErrPageNumberInvalid   = errors.New("page number must be positive")
	ErrPageSizeInvalid     = errors.New("page size must be between 1 and 100")
	ErrSearchQueryTooShort = errors.New("search query must be at least 2 characters")
	ErrSearchQueryTooLong  = errors.New("search query is too long (max 100 characters)")
	ErrTimeRangeInvalid    = errors.New("end date must be on or after start date")

	ErrProfileNameRequired     = errors.New("profile name is required")
	ErrProfileChannelRequired  = errors.New("channel is required")
	ErrProfileChannelIDInvalid = errors.New("channel id must be a positive integer")

	ErrOrganizationNameRequired    = errors.New("organization name is required")
	ErrOrganizationMemberRequired  = errors.New("profile is required")
	ErrOrganizationMemberIDInvalid = errors.New("profile id must be a positive integer")

	ErrEventTitleRequired        = errors.New("event title is required")
	ErrEventStartAtRequired      = errors.New("start_at is required")
	ErrEventDatesInvalid         = errors.New("end_at must be on or after start_at")
	ErrEventParticipantRequired  = errors.New("profile is required")
	ErrEventParticipantIDInvalid = errors.New("profile id must be a positive integer")
)

// Validation patterns
var (
	channelNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	usernamePattern    = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

// Channel represents a channel in API responses.
type Channel struct {
	ID                    int64      `json:"id"`
	Name                  string     `json:"name"`
	DisplayName           string     `json:"display_name"`
	Enabled               bool       `json:"enabled"`
	RetainHistoryOnDelete bool       `json:"retain_history_on_delete"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	LastMessageAt         *time.Time `json:"last_message_at,omitempty"`
	TotalMessages         int64      `json:"total_messages"`
}

// CreateChannelRequest is the request body for creating a channel.
type CreateChannelRequest struct {
	Name                  string `json:"name"`
	DisplayName           string `json:"display_name"`
	Enabled               bool   `json:"enabled"`
	RetainHistoryOnDelete bool   `json:"retain_history_on_delete"`
}

// Validate validates the create channel request.
func (r *CreateChannelRequest) Validate() error {
	r.Name = strings.TrimSpace(strings.ToLower(r.Name))
	if r.Name == "" {
		return ErrChannelNameRequired
	}
	if len(r.Name) > 25 {
		return ErrChannelNameTooLong
	}
	if !channelNamePattern.MatchString(r.Name) {
		return ErrChannelNameInvalid
	}
	if r.DisplayName == "" {
		r.DisplayName = r.Name
	}
	return nil
}

// UpdateChannelRequest is the request body for updating a channel.
type UpdateChannelRequest struct {
	DisplayName           *string `json:"display_name,omitempty"`
	Enabled               *bool   `json:"enabled,omitempty"`
	RetainHistoryOnDelete *bool   `json:"retain_history_on_delete,omitempty"`
}

// DeleteChannelRequest is the request body for deleting a channel.
type DeleteChannelRequest struct {
	RetainHistory bool `json:"retain_history"`
}

// User represents a user in API responses.
type User struct {
	ID            int64     `json:"id"`
	Username      string    `json:"username"`
	DisplayName   string    `json:"display_name,omitempty"`
	FirstSeenAt   time.Time `json:"first_seen_at"`
	LastSeenAt    time.Time `json:"last_seen_at"`
	TotalMessages int64     `json:"total_messages"`
}

// Message represents a message in API responses.
type Message struct {
	ID          int64     `json:"id"`
	ChannelID   int64     `json:"channel_id"`
	ChannelName string    `json:"channel_name,omitempty"`
	UserID      int64     `json:"user_id"`
	Username    string    `json:"username,omitempty"`
	DisplayName string    `json:"display_name,omitempty"`
	Text        string    `json:"text"`
	SentAt      time.Time `json:"sent_at"`
}

// PaginationRequest holds pagination parameters.
type PaginationRequest struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// Validate validates pagination parameters.
func (p *PaginationRequest) Validate() error {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 || p.PageSize > 100 {
		if p.PageSize == 0 {
			p.PageSize = 20 // Default
		} else {
			return ErrPageSizeInvalid
		}
	}
	return nil
}

// Offset returns the SQL offset for pagination.
func (p *PaginationRequest) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// PaginatedResponse is a generic paginated response wrapper.
type PaginatedResponse[T any] struct {
	Items      []T  `json:"items"`
	Page       int  `json:"page"`
	PageSize   int  `json:"page_size"`
	TotalCount int  `json:"total_count"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

// SearchMessagesRequest is the request for searching messages.
type SearchMessagesRequest struct {
	Query       string     `json:"q"`
	ChannelName *string    `json:"channel,omitempty"`
	Username    *string    `json:"username,omitempty"`
	StartTime   *time.Time `json:"start,omitempty"`
	EndTime     *time.Time `json:"end,omitempty"`
	PaginationRequest
}

// Validate validates the search messages request.
func (r *SearchMessagesRequest) Validate() error {
	originalQuery := r.Query
	r.Query = strings.TrimSpace(r.Query)
	// Query is required only if no other filters are set
	hasFilters := r.ChannelName != nil || r.Username != nil || r.StartTime != nil || r.EndTime != nil
	if r.Query != "" {
		if len(r.Query) < 2 {
			return ErrSearchQueryTooShort
		}
		if len(r.Query) > 100 {
			return ErrSearchQueryTooLong
		}
	} else if originalQuery != "" {
		// User provided input but it was only whitespace - treat as too short
		return ErrSearchQueryTooShort
	} else if !hasFilters {
		// No query and no filters - this shouldn't be validated (used for recent messages)
		return nil
	}
	// Validate time range: end must be on or after start
	if r.StartTime != nil && r.EndTime != nil {
		if r.EndTime.Before(*r.StartTime) {
			return ErrTimeRangeInvalid
		}
	}
	return r.PaginationRequest.Validate()
}

// SearchUsersRequest is the request for searching users.
type SearchUsersRequest struct {
	Query string `json:"q"`
	PaginationRequest
}

// Validate validates the search users request.
func (r *SearchUsersRequest) Validate() error {
	r.Query = strings.TrimSpace(strings.ToLower(r.Query))
	if r.Query == "" {
		return ErrUsernameRequired
	}
	return r.PaginationRequest.Validate()
}

// ListUsersRequest is the request for listing users with optional filtering.
type ListUsersRequest struct {
	Query string `json:"q"` // Optional filter by username
	PaginationRequest
}

// Validate validates the list users request.
func (r *ListUsersRequest) Validate() error {
	r.Query = strings.TrimSpace(strings.ToLower(r.Query))
	// Query is optional for listing, so no required check
	return r.PaginationRequest.Validate()
}

// Profile represents a profile in API responses.
// (US1 uses server-rendered HTML, but DTOs keep handlers consistent.)
type Profile struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateProfileRequest is the request for creating a profile.
type CreateProfileRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (r *CreateProfileRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.Description = strings.TrimSpace(r.Description)
	if r.Name == "" {
		return ErrProfileNameRequired
	}
	return nil
}

// UpdateProfileRequest is the request for updating a profile.
type UpdateProfileRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (r *UpdateProfileRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.Description = strings.TrimSpace(r.Description)
	if r.Name == "" {
		return ErrProfileNameRequired
	}
	return nil
}

// LinkProfileChannelRequest is the request for linking a channel to a profile.
type LinkProfileChannelRequest struct {
	ChannelID int64 `json:"channel_id"`
}

func (r *LinkProfileChannelRequest) Validate() error {
	if r.ChannelID <= 0 {
		return ErrProfileChannelIDInvalid
	}
	return nil
}

// Organization represents an organization in API responses.
type Organization struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateOrganizationRequest is the request for creating an organization.
type CreateOrganizationRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (r *CreateOrganizationRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.Description = strings.TrimSpace(r.Description)
	if r.Name == "" {
		return ErrOrganizationNameRequired
	}
	return nil
}

// UpdateOrganizationRequest is the request for updating an organization.
type UpdateOrganizationRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (r *UpdateOrganizationRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.Description = strings.TrimSpace(r.Description)
	if r.Name == "" {
		return ErrOrganizationNameRequired
	}
	return nil
}

// AddOrganizationMemberRequest is the request for adding a profile membership.
type AddOrganizationMemberRequest struct {
	ProfileID int64 `json:"profile_id"`
}

func (r *AddOrganizationMemberRequest) Validate() error {
	if r.ProfileID <= 0 {
		return ErrOrganizationMemberIDInvalid
	}
	return nil
}

// Event represents an event in API responses.
type Event struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	StartAt     time.Time  `json:"start_at"`
	EndAt       *time.Time `json:"end_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreateEventRequest is the request for creating an event.
type CreateEventRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	StartAt     time.Time  `json:"start_at"`
	EndAt       *time.Time `json:"end_at,omitempty"`
}

func (r *CreateEventRequest) Validate() error {
	r.Title = strings.TrimSpace(r.Title)
	r.Description = strings.TrimSpace(r.Description)
	if r.Title == "" {
		return ErrEventTitleRequired
	}
	if r.StartAt.IsZero() {
		return ErrEventStartAtRequired
	}
	if r.EndAt != nil && !r.EndAt.IsZero() && r.EndAt.Before(r.StartAt) {
		return ErrEventDatesInvalid
	}
	return nil
}

// UpdateEventRequest is the request for updating an event.
type UpdateEventRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	StartAt     time.Time  `json:"start_at"`
	EndAt       *time.Time `json:"end_at,omitempty"`
}

func (r *UpdateEventRequest) Validate() error {
	r.Title = strings.TrimSpace(r.Title)
	r.Description = strings.TrimSpace(r.Description)
	if r.Title == "" {
		return ErrEventTitleRequired
	}
	if r.StartAt.IsZero() {
		return ErrEventStartAtRequired
	}
	if r.EndAt != nil && !r.EndAt.IsZero() && r.EndAt.Before(r.StartAt) {
		return ErrEventDatesInvalid
	}
	return nil
}

// AddEventParticipantRequest is the request for adding a participant.
type AddEventParticipantRequest struct {
	ProfileID int64 `json:"profile_id"`
}

func (r *AddEventParticipantRequest) Validate() error {
	if r.ProfileID <= 0 {
		return ErrEventParticipantIDInvalid
	}
	return nil
}

// ValidateUsername validates a username.
func ValidateUsername(username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return ErrUsernameRequired
	}
	if !usernamePattern.MatchString(username) {
		return ErrUsernameInvalid
	}
	return nil
}
