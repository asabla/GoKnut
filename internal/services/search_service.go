// Package services provides business logic for the Twitch Chat Archiver.
package services

import (
	"context"
	"errors"
	"time"

	"github.com/asabla/goknut/internal/http/dto"
	"github.com/asabla/goknut/internal/observability"
	"github.com/asabla/goknut/internal/search"
)

// SearchService errors
var (
	ErrUserNotFound        = errors.New("user not found")
	ErrSearchQueryEmpty    = errors.New("search query is empty")
	ErrSearchQueryTooShort = errors.New("search query is too short")
)

// SearchService provides search operations for users and messages.
type SearchService struct {
	repo    *search.SearchRepository
	logger  *observability.Logger
	metrics *observability.Metrics
}

// NewSearchService creates a new search service.
func NewSearchService(
	repo *search.SearchRepository,
	logger *observability.Logger,
	metrics *observability.Metrics,
) *SearchService {
	return &SearchService{
		repo:    repo,
		logger:  logger,
		metrics: metrics,
	}
}

// UserSearchResult is the result of a user search.
type UserSearchResult struct {
	Users      []search.UserSearchResult
	TotalCount int
	Page       int
	PageSize   int
	TotalPages int
	HasNext    bool
	HasPrev    bool
}

// MessageSearchResult is the result of a message search.
type MessageSearchResult struct {
	Messages   []search.MessageSearchResult
	TotalCount int
	Page       int
	PageSize   int
	TotalPages int
	HasNext    bool
	HasPrev    bool
}

// SearchUsers searches for users by username fragment.
func (s *SearchService) SearchUsers(ctx context.Context, req dto.SearchUsersRequest) (*UserSearchResult, error) {
	start := time.Now()
	defer func() {
		latency := time.Since(start)
		if s.metrics != nil {
			s.metrics.RecordSearchRequest("users", latency)
		}
	}()

	if err := req.Validate(); err != nil {
		return nil, err
	}

	params := search.UserSearchParams{
		Query:    req.Query,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	users, totalCount, err := s.repo.SearchUsers(ctx, params)
	if err != nil {
		s.logger.Error("failed to search users", "query", req.Query, "error", err)
		return nil, err
	}

	totalPages := (totalCount + req.PageSize - 1) / req.PageSize
	if totalPages < 1 {
		totalPages = 1
	}

	result := &UserSearchResult{
		Users:      users,
		TotalCount: totalCount,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
		HasNext:    req.Page < totalPages,
		HasPrev:    req.Page > 1,
	}

	s.logger.Search("user search completed",
		"query", req.Query,
		"results", len(users),
		"total", totalCount,
		"latency_ms", time.Since(start).Milliseconds(),
	)

	return result, nil
}

// ListUsers returns all users with optional filtering and pagination.
func (s *SearchService) ListUsers(ctx context.Context, req dto.ListUsersRequest) (*UserSearchResult, error) {
	start := time.Now()
	defer func() {
		latency := time.Since(start)
		if s.metrics != nil {
			s.metrics.RecordSearchRequest("list_users", latency)
		}
	}()

	if err := req.Validate(); err != nil {
		return nil, err
	}

	params := search.ListUsersParams{
		Query:    req.Query,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	users, totalCount, err := s.repo.ListUsers(ctx, params)
	if err != nil {
		s.logger.Error("failed to list users", "query", req.Query, "error", err)
		return nil, err
	}

	totalPages := (totalCount + req.PageSize - 1) / req.PageSize
	if totalPages < 1 {
		totalPages = 1
	}

	result := &UserSearchResult{
		Users:      users,
		TotalCount: totalCount,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
		HasNext:    req.Page < totalPages,
		HasPrev:    req.Page > 1,
	}

	s.logger.Search("user list completed",
		"filter", req.Query,
		"results", len(users),
		"total", totalCount,
		"latency_ms", time.Since(start).Milliseconds(),
	)

	return result, nil
}

// SearchMessages searches for messages by text content.
func (s *SearchService) SearchMessages(ctx context.Context, req dto.SearchMessagesRequest) (*MessageSearchResult, error) {
	start := time.Now()
	defer func() {
		latency := time.Since(start)
		if s.metrics != nil {
			s.metrics.RecordSearchRequest("messages", latency)
		}
	}()

	if err := req.Validate(); err != nil {
		return nil, err
	}

	params := search.MessageSearchParams{
		Query:     req.Query,
		ChannelID: req.ChannelID,
		UserID:    req.UserID,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Page:      req.Page,
		PageSize:  req.PageSize,
	}

	messages, totalCount, err := s.repo.SearchMessages(ctx, params)
	if err != nil {
		s.logger.Error("failed to search messages", "query", req.Query, "error", err)
		return nil, err
	}

	totalPages := (totalCount + req.PageSize - 1) / req.PageSize
	if totalPages < 1 {
		totalPages = 1
	}

	result := &MessageSearchResult{
		Messages:   messages,
		TotalCount: totalCount,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
		HasNext:    req.Page < totalPages,
		HasPrev:    req.Page > 1,
	}

	s.logger.Search("message search completed",
		"query", req.Query,
		"channel_id", req.ChannelID,
		"results", len(messages),
		"total", totalCount,
		"latency_ms", time.Since(start).Milliseconds(),
	)

	return result, nil
}

// GetUserProfile returns detailed user information.
func (s *SearchService) GetUserProfile(ctx context.Context, userID int64) (*search.UserProfile, error) {
	start := time.Now()
	defer func() {
		latency := time.Since(start)
		if s.metrics != nil {
			s.metrics.RecordSearchRequest("profile", latency)
		}
	}()

	profile, err := s.repo.GetUserProfile(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user profile", "user_id", userID, "error", err)
		return nil, err
	}
	if profile == nil {
		return nil, ErrUserNotFound
	}

	s.logger.Search("user profile fetched",
		"user_id", userID,
		"username", profile.Username,
		"channels", len(profile.Channels),
		"latency_ms", time.Since(start).Milliseconds(),
	)

	return profile, nil
}

// GetUserProfileByUsername returns detailed user information by username.
func (s *SearchService) GetUserProfileByUsername(ctx context.Context, username string) (*search.UserProfile, error) {
	start := time.Now()
	defer func() {
		latency := time.Since(start)
		if s.metrics != nil {
			s.metrics.RecordSearchRequest("profile", latency)
		}
	}()

	profile, err := s.repo.GetUserProfileByUsername(ctx, username)
	if err != nil {
		s.logger.Error("failed to get user profile", "username", username, "error", err)
		return nil, err
	}
	if profile == nil {
		return nil, ErrUserNotFound
	}

	s.logger.Search("user profile fetched",
		"user_id", profile.ID,
		"username", profile.Username,
		"channels", len(profile.Channels),
		"latency_ms", time.Since(start).Milliseconds(),
	)

	return profile, nil
}

// GetUserMessages returns paginated messages for a user.
func (s *SearchService) GetUserMessages(ctx context.Context, userID int64, channelID *int64, page, pageSize int) (*MessageSearchResult, error) {
	start := time.Now()
	defer func() {
		latency := time.Since(start)
		if s.metrics != nil {
			s.metrics.RecordSearchRequest("user_messages", latency)
		}
	}()

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	messages, totalCount, err := s.repo.GetUserMessages(ctx, userID, channelID, page, pageSize)
	if err != nil {
		s.logger.Error("failed to get user messages", "user_id", userID, "error", err)
		return nil, err
	}

	totalPages := (totalCount + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	result := &MessageSearchResult{
		Messages:   messages,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	s.logger.Search("user messages fetched",
		"user_id", userID,
		"channel_id", channelID,
		"results", len(messages),
		"total", totalCount,
		"latency_ms", time.Since(start).Milliseconds(),
	)

	return result, nil
}

// GetUserMessagesByUsername returns paginated messages for a user by username.
func (s *SearchService) GetUserMessagesByUsername(ctx context.Context, username string, channelName *string, page, pageSize int) (*MessageSearchResult, error) {
	start := time.Now()
	defer func() {
		latency := time.Since(start)
		if s.metrics != nil {
			s.metrics.RecordSearchRequest("user_messages", latency)
		}
	}()

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	messages, totalCount, err := s.repo.GetUserMessagesByUsername(ctx, username, channelName, page, pageSize)
	if err != nil {
		s.logger.Error("failed to get user messages", "username", username, "error", err)
		return nil, err
	}

	totalPages := (totalCount + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	result := &MessageSearchResult{
		Messages:   messages,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	s.logger.Search("user messages fetched",
		"username", username,
		"channel_name", channelName,
		"results", len(messages),
		"total", totalCount,
		"latency_ms", time.Since(start).Milliseconds(),
	)

	return result, nil
}
