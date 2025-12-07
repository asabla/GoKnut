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
	Query     string     `json:"q"`
	ChannelID *int64     `json:"channel_id,omitempty"`
	UserID    *int64     `json:"user_id,omitempty"`
	StartTime *time.Time `json:"start,omitempty"`
	EndTime   *time.Time `json:"end,omitempty"`
	PaginationRequest
}

// Validate validates the search messages request.
func (r *SearchMessagesRequest) Validate() error {
	r.Query = strings.TrimSpace(r.Query)
	if len(r.Query) < 2 {
		return ErrSearchQueryTooShort
	}
	if len(r.Query) > 100 {
		return ErrSearchQueryTooLong
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
