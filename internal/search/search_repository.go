// Package search provides search functionality for users and messages.
package search

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/asabla/goknut/internal/repository"
)

// parseTimeValue converts database time values to time.Time.
// Handles both string (SQLite) and time.Time (Postgres) values.
func parseTimeValue(v any) time.Time {
	if v == nil {
		return time.Time{}
	}
	switch t := v.(type) {
	case time.Time:
		return t
	case string:
		parsed, _ := repository.ParseSQLiteDatetime(t)
		return parsed
	case *time.Time:
		if t != nil {
			return *t
		}
	case *string:
		if t != nil {
			parsed, _ := repository.ParseSQLiteDatetime(*t)
			return parsed
		}
	}
	return time.Time{}
}

// MessageSearchParams defines parameters for message search.
type MessageSearchParams struct {
	Query       string
	ChannelName *string
	Username    *string
	StartTime   *time.Time
	EndTime     *time.Time
	Page        int
	PageSize    int
}

// UserSearchParams defines parameters for user search.
type UserSearchParams struct {
	Query    string
	Page     int
	PageSize int
}

// UserSearchResult represents a user in search results.
type UserSearchResult struct {
	ID            int64
	Username      string
	DisplayName   string
	FirstSeenAt   time.Time
	LastSeenAt    time.Time
	TotalMessages int64
	ChannelCount  int64 // Number of distinct channels
}

// MessageSearchResult represents a message in search results.
type MessageSearchResult struct {
	ID              int64
	ChannelID       int64
	ChannelName     string
	UserID          int64
	Username        string
	DisplayName     string
	Text            string
	HighlightedText string // Text with search terms highlighted
	SentAt          time.Time
	Tags            map[string]string
}

// UserProfile represents detailed user information.
type UserProfile struct {
	ID            int64
	Username      string
	DisplayName   string
	FirstSeenAt   time.Time
	LastSeenAt    time.Time
	TotalMessages int64
	Channels      []UserChannelSummary
}

// UserChannelSummary represents a channel in user profile.
type UserChannelSummary struct {
	ID            int64
	Name          string
	DisplayName   string
	MessageCount  int64
	LastMessageAt time.Time
}

// SearchRepository provides search operations.
type SearchRepository struct {
	db        repository.Database
	enableFTS bool
}

// NewSearchRepository creates a new search repository.
func NewSearchRepository(db repository.Database, enableFTS bool) *SearchRepository {
	return &SearchRepository{
		db:        db,
		enableFTS: enableFTS,
	}
}

// ph returns the appropriate placeholder for the given index (1-based).
func (r *SearchRepository) ph(index int) string {
	return r.db.Placeholder(index)
}

// SearchUsers searches for users by username fragment.
func (r *SearchRepository) SearchUsers(ctx context.Context, params UserSearchParams) ([]UserSearchResult, int, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}
	offset := (params.Page - 1) * params.PageSize

	// Build LIKE pattern for username search
	pattern := BuildLIKEPattern(params.Query)

	// Count query
	countQuery := `
		SELECT COUNT(*) FROM users
		WHERE username LIKE ` + r.ph(1) + ` ESCAPE '\'
	`
	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, pattern).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Main query with channel count
	query := `
		SELECT 
			u.id, u.username, u.display_name, u.first_seen_at, u.last_seen_at, u.total_messages,
			(SELECT COUNT(DISTINCT channel_id) FROM messages WHERE user_id = u.id) as channel_count
		FROM users u
		WHERE u.username LIKE ` + r.ph(1) + ` ESCAPE '\'
		ORDER BY u.total_messages DESC, u.username ASC
		LIMIT ` + r.ph(2) + ` OFFSET ` + r.ph(3) + `
	`

	rows, err := r.db.QueryContext(ctx, query, pattern, params.PageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search users: %w", err)
	}
	defer rows.Close()

	var results []UserSearchResult
	for rows.Next() {
		var u UserSearchResult
		var firstSeen, lastSeen any
		var displayName sql.NullString

		if err := rows.Scan(&u.ID, &u.Username, &displayName, &firstSeen, &lastSeen, &u.TotalMessages, &u.ChannelCount); err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}

		u.FirstSeenAt = parseTimeValue(firstSeen)
		u.LastSeenAt = parseTimeValue(lastSeen)
		if displayName.Valid {
			u.DisplayName = displayName.String
		}

		results = append(results, u)
	}

	return results, totalCount, rows.Err()
}

// ListUsersParams defines parameters for listing users.
type ListUsersParams struct {
	Query    string // Optional filter by username
	Page     int
	PageSize int
}

// ListUsers returns all users with optional filtering and pagination.
func (r *SearchRepository) ListUsers(ctx context.Context, params ListUsersParams) ([]UserSearchResult, int, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}
	offset := (params.Page - 1) * params.PageSize

	var conditions []string
	var args []any
	argIndex := 1

	// Apply optional username filter
	if params.Query != "" {
		pattern := BuildLIKEPattern(params.Query)
		conditions = append(conditions, "u.username LIKE "+r.ph(argIndex)+" ESCAPE '\\'")
		args = append(args, pattern)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM users u %s`, whereClause)
	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Main query with channel count
	query := fmt.Sprintf(`
		SELECT 
			u.id, u.username, u.display_name, u.first_seen_at, u.last_seen_at, u.total_messages,
			(SELECT COUNT(DISTINCT channel_id) FROM messages WHERE user_id = u.id) as channel_count
		FROM users u
		%s
		ORDER BY u.total_messages DESC, u.username ASC
		LIMIT %s OFFSET %s
	`, whereClause, r.ph(argIndex), r.ph(argIndex+1))

	args = append(args, params.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var results []UserSearchResult
	for rows.Next() {
		var u UserSearchResult
		var firstSeen, lastSeen any
		var displayName sql.NullString

		if err := rows.Scan(&u.ID, &u.Username, &displayName, &firstSeen, &lastSeen, &u.TotalMessages, &u.ChannelCount); err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}

		u.FirstSeenAt = parseTimeValue(firstSeen)
		u.LastSeenAt = parseTimeValue(lastSeen)
		if displayName.Valid {
			u.DisplayName = displayName.String
		}

		results = append(results, u)
	}

	return results, totalCount, rows.Err()
}

// SearchMessages searches for messages by text content.
func (r *SearchRepository) SearchMessages(ctx context.Context, params MessageSearchParams) ([]MessageSearchResult, int, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}
	offset := (params.Page - 1) * params.PageSize

	if r.enableFTS {
		return r.searchMessagesFTS(ctx, params, offset)
	}
	return r.searchMessagesLIKE(ctx, params, offset)
}

// searchMessagesFTS performs FTS5-based message search.
func (r *SearchRepository) searchMessagesFTS(ctx context.Context, params MessageSearchParams, offset int) ([]MessageSearchResult, int, error) {
	// Build WHERE clauses for filters
	var conditions []string
	var args []any
	argIndex := 1

	// FTS query is optional when using filters
	if params.Query != "" {
		ftsQuery := BuildFTSQuery(params.Query)
		conditions = append(conditions, "f.content MATCH "+r.ph(argIndex))
		args = append(args, ftsQuery)
		argIndex++
	}

	if params.ChannelName != nil {
		conditions = append(conditions, "c.name = "+r.ph(argIndex))
		args = append(args, *params.ChannelName)
		argIndex++
	}
	if params.Username != nil {
		conditions = append(conditions, "u.username = "+r.ph(argIndex))
		args = append(args, *params.Username)
		argIndex++
	}
	if params.StartTime != nil {
		conditions = append(conditions, "m.sent_at >= "+r.ph(argIndex))
		args = append(args, params.StartTime.Format(time.RFC3339))
		argIndex++
	}
	if params.EndTime != nil {
		conditions = append(conditions, "m.sent_at <= "+r.ph(argIndex))
		args = append(args, params.EndTime.Format(time.RFC3339))
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query - need to join tables for name-based filtering
	var countQuery string
	if params.Query != "" {
		countQuery = fmt.Sprintf(`
			SELECT COUNT(*)
			FROM messages_fts f
			JOIN messages m ON f.rowid = m.id
			JOIN channels c ON m.channel_id = c.id
			JOIN users u ON m.user_id = u.id
			%s
		`, whereClause)
	} else {
		countQuery = fmt.Sprintf(`
			SELECT COUNT(*)
			FROM messages m
			JOIN channels c ON m.channel_id = c.id
			JOIN users u ON m.user_id = u.id
			%s
		`, whereClause)
	}

	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count messages: %w", err)
	}

	// Main query
	var query string
	if params.Query != "" {
		query = fmt.Sprintf(`
			SELECT 
				m.id, m.channel_id, c.name, m.user_id, u.username, u.display_name,
				m.text, m.sent_at, m.tags
			FROM messages_fts f
			JOIN messages m ON f.rowid = m.id
			JOIN channels c ON m.channel_id = c.id
			JOIN users u ON m.user_id = u.id
			%s
			ORDER BY m.sent_at DESC, m.id DESC
			LIMIT %s OFFSET %s
		`, whereClause, r.ph(argIndex), r.ph(argIndex+1))
	} else {
		query = fmt.Sprintf(`
			SELECT 
				m.id, m.channel_id, c.name, m.user_id, u.username, u.display_name,
				m.text, m.sent_at, m.tags
			FROM messages m
			JOIN channels c ON m.channel_id = c.id
			JOIN users u ON m.user_id = u.id
			%s
			ORDER BY m.sent_at DESC, m.id DESC
			LIMIT %s OFFSET %s
		`, whereClause, r.ph(argIndex), r.ph(argIndex+1))
	}

	args = append(args, params.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search messages: %w", err)
	}
	defer rows.Close()

	return r.scanMessageResults(rows, params.Query)
}

// searchMessagesLIKE performs LIKE-based message search (fallback).
func (r *SearchRepository) searchMessagesLIKE(ctx context.Context, params MessageSearchParams, offset int) ([]MessageSearchResult, int, error) {
	// Build WHERE clauses for filters
	var conditions []string
	var args []any
	argIndex := 1

	// Text search is optional when using filters
	if params.Query != "" {
		pattern := BuildLIKEPattern(params.Query)
		conditions = append(conditions, "m.text LIKE "+r.ph(argIndex)+" ESCAPE '\\'")
		args = append(args, pattern)
		argIndex++
	}

	if params.ChannelName != nil {
		conditions = append(conditions, "c.name = "+r.ph(argIndex))
		args = append(args, *params.ChannelName)
		argIndex++
	}
	if params.Username != nil {
		conditions = append(conditions, "u.username = "+r.ph(argIndex))
		args = append(args, *params.Username)
		argIndex++
	}
	if params.StartTime != nil {
		conditions = append(conditions, "m.sent_at >= "+r.ph(argIndex))
		args = append(args, params.StartTime.Format(time.RFC3339))
		argIndex++
	}
	if params.EndTime != nil {
		conditions = append(conditions, "m.sent_at <= "+r.ph(argIndex))
		args = append(args, params.EndTime.Format(time.RFC3339))
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM messages m
		JOIN channels c ON m.channel_id = c.id
		JOIN users u ON m.user_id = u.id
		%s
	`, whereClause)

	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count messages: %w", err)
	}

	// Main query
	query := fmt.Sprintf(`
		SELECT 
			m.id, m.channel_id, c.name, m.user_id, u.username, u.display_name,
			m.text, m.sent_at, m.tags
		FROM messages m
		JOIN channels c ON m.channel_id = c.id
		JOIN users u ON m.user_id = u.id
		%s
		ORDER BY m.sent_at DESC, m.id DESC
		LIMIT %s OFFSET %s
	`, whereClause, r.ph(argIndex), r.ph(argIndex+1))

	args = append(args, params.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search messages: %w", err)
	}
	defer rows.Close()

	return r.scanMessageResults(rows, params.Query)
}

func (r *SearchRepository) scanMessageResults(rows *sql.Rows, query string) ([]MessageSearchResult, int, error) {
	var results []MessageSearchResult

	for rows.Next() {
		var m MessageSearchResult
		var sentAt any
		var displayName, tagsJSON sql.NullString

		if err := rows.Scan(
			&m.ID, &m.ChannelID, &m.ChannelName, &m.UserID, &m.Username, &displayName,
			&m.Text, &sentAt, &tagsJSON,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan message: %w", err)
		}

		m.SentAt = parseTimeValue(sentAt)
		if displayName.Valid {
			m.DisplayName = displayName.String
		}
		if tagsJSON.Valid && tagsJSON.String != "" {
			_ = json.Unmarshal([]byte(tagsJSON.String), &m.Tags)
		}

		// Generate highlighted text
		m.HighlightedText = HighlightTerm(m.Text, query)

		results = append(results, m)
	}

	return results, len(results), rows.Err()
}

// GetUserProfile returns detailed user information.
func (r *SearchRepository) GetUserProfile(ctx context.Context, userID int64) (*UserProfile, error) {
	// Get user basic info
	userQuery := `
		SELECT id, username, display_name, first_seen_at, last_seen_at, total_messages
		FROM users
		WHERE id = ` + r.ph(1) + `
	`

	return r.getUserProfile(ctx, userQuery, userID)
}

// GetUserProfileByUsername returns detailed user information by username.
func (r *SearchRepository) GetUserProfileByUsername(ctx context.Context, username string) (*UserProfile, error) {
	// Get user basic info
	userQuery := `
		SELECT id, username, display_name, first_seen_at, last_seen_at, total_messages
		FROM users
		WHERE username = ` + r.ph(1) + `
	`

	return r.getUserProfile(ctx, userQuery, username)
}

// getUserProfile is a shared helper for fetching user profiles.
func (r *SearchRepository) getUserProfile(ctx context.Context, userQuery string, arg any) (*UserProfile, error) {
	var profile UserProfile
	var firstSeen, lastSeen any
	var displayName sql.NullString

	err := r.db.QueryRowContext(ctx, userQuery, arg).Scan(
		&profile.ID, &profile.Username, &displayName, &firstSeen, &lastSeen, &profile.TotalMessages,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	profile.FirstSeenAt = parseTimeValue(firstSeen)
	profile.LastSeenAt = parseTimeValue(lastSeen)
	if displayName.Valid {
		profile.DisplayName = displayName.String
	}

	// Get channel summaries using the profile.ID we just fetched
	channelQuery := `
		SELECT 
			c.id, c.name, c.display_name,
			COUNT(m.id) as message_count,
			MAX(m.sent_at) as last_message_at
		FROM channels c
		JOIN messages m ON c.id = m.channel_id
		WHERE m.user_id = ` + r.ph(1) + `
		GROUP BY c.id, c.name, c.display_name
		ORDER BY message_count DESC
	`

	rows, err := r.db.QueryContext(ctx, channelQuery, profile.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user channels: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ch UserChannelSummary
		var lastMsgAt any

		if err := rows.Scan(&ch.ID, &ch.Name, &ch.DisplayName, &ch.MessageCount, &lastMsgAt); err != nil {
			return nil, fmt.Errorf("failed to scan channel: %w", err)
		}

		ch.LastMessageAt = parseTimeValue(lastMsgAt)
		profile.Channels = append(profile.Channels, ch)
	}

	return &profile, rows.Err()
}

// GetUserMessages returns paginated messages for a user.
func (r *SearchRepository) GetUserMessages(ctx context.Context, userID int64, channelID *int64, page, pageSize int) ([]MessageSearchResult, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// Build WHERE clause
	var conditions []string
	var args []any
	argIndex := 1

	conditions = append(conditions, "m.user_id = "+r.ph(argIndex))
	args = append(args, userID)
	argIndex++

	if channelID != nil {
		conditions = append(conditions, "m.channel_id = "+r.ph(argIndex))
		args = append(args, *channelID)
		argIndex++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count query
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM messages m WHERE %s`, whereClause)

	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count messages: %w", err)
	}

	// Main query
	query := fmt.Sprintf(`
		SELECT 
			m.id, m.channel_id, c.name, m.user_id, u.username, u.display_name,
			m.text, m.sent_at, m.tags
		FROM messages m
		JOIN channels c ON m.channel_id = c.id
		JOIN users u ON m.user_id = u.id
		WHERE %s
		ORDER BY m.sent_at DESC, m.id DESC
		LIMIT %s OFFSET %s
	`, whereClause, r.ph(argIndex), r.ph(argIndex+1))

	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user messages: %w", err)
	}
	defer rows.Close()

	var results []MessageSearchResult
	for rows.Next() {
		var m MessageSearchResult
		var sentAt any
		var displayName, tagsJSON sql.NullString

		if err := rows.Scan(
			&m.ID, &m.ChannelID, &m.ChannelName, &m.UserID, &m.Username, &displayName,
			&m.Text, &sentAt, &tagsJSON,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan message: %w", err)
		}

		m.SentAt = parseTimeValue(sentAt)
		if displayName.Valid {
			m.DisplayName = displayName.String
		}
		if tagsJSON.Valid && tagsJSON.String != "" {
			_ = json.Unmarshal([]byte(tagsJSON.String), &m.Tags)
		}
		m.HighlightedText = m.Text // No highlighting for user messages view

		results = append(results, m)
	}

	return results, totalCount, rows.Err()
}

// GetUserMessagesByUsername returns paginated messages for a user by username.
func (r *SearchRepository) GetUserMessagesByUsername(ctx context.Context, username string, channelName *string, page, pageSize int) ([]MessageSearchResult, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// Build WHERE clause using JOINs to resolve names to IDs
	var conditions []string
	var args []any
	argIndex := 1

	conditions = append(conditions, "u.username = "+r.ph(argIndex))
	args = append(args, username)
	argIndex++

	if channelName != nil {
		conditions = append(conditions, "c.name = "+r.ph(argIndex))
		args = append(args, *channelName)
		argIndex++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM messages m
		JOIN users u ON m.user_id = u.id
		JOIN channels c ON m.channel_id = c.id
		WHERE %s
	`, whereClause)

	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count messages: %w", err)
	}

	// Main query
	query := fmt.Sprintf(`
		SELECT 
			m.id, m.channel_id, c.name, m.user_id, u.username, u.display_name,
			m.text, m.sent_at, m.tags
		FROM messages m
		JOIN channels c ON m.channel_id = c.id
		JOIN users u ON m.user_id = u.id
		WHERE %s
		ORDER BY m.sent_at DESC, m.id DESC
		LIMIT %s OFFSET %s
	`, whereClause, r.ph(argIndex), r.ph(argIndex+1))

	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user messages: %w", err)
	}
	defer rows.Close()

	var results []MessageSearchResult
	for rows.Next() {
		var m MessageSearchResult
		var sentAt any
		var displayName, tagsJSON sql.NullString

		if err := rows.Scan(
			&m.ID, &m.ChannelID, &m.ChannelName, &m.UserID, &m.Username, &displayName,
			&m.Text, &sentAt, &tagsJSON,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan message: %w", err)
		}

		m.SentAt = parseTimeValue(sentAt)
		if displayName.Valid {
			m.DisplayName = displayName.String
		}
		if tagsJSON.Valid && tagsJSON.String != "" {
			_ = json.Unmarshal([]byte(tagsJSON.String), &m.Tags)
		}
		m.HighlightedText = m.Text // No highlighting for user messages view

		results = append(results, m)
	}

	return results, totalCount, rows.Err()
}

// BuildFTSQuery builds an FTS5 query from user input.
func BuildFTSQuery(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	// Handle quoted phrases
	var parts []string
	inQuote := false
	var current strings.Builder

	for _, r := range input {
		if r == '"' {
			if inQuote {
				// End of quoted phrase
				if current.Len() > 0 {
					parts = append(parts, `"`+current.String()+`"`)
					current.Reset()
				}
				inQuote = false
			} else {
				// Start of quoted phrase
				if current.Len() > 0 {
					// Add previous words with prefix matching
					for _, word := range strings.Fields(current.String()) {
						word = sanitizeFTSWord(word)
						if word != "" {
							parts = append(parts, word+"*")
						}
					}
					current.Reset()
				}
				inQuote = true
			}
		} else {
			current.WriteRune(r)
		}
	}

	// Handle remaining content
	if current.Len() > 0 {
		if inQuote {
			// Unclosed quote, treat as phrase
			parts = append(parts, `"`+current.String()+`"`)
		} else {
			for _, word := range strings.Fields(current.String()) {
				word = sanitizeFTSWord(word)
				if word != "" {
					parts = append(parts, word+"*")
				}
			}
		}
	}

	return strings.Join(parts, " ")
}

// sanitizeFTSWord removes special characters from FTS word.
func sanitizeFTSWord(word string) string {
	// Remove FTS5 special characters
	reg := regexp.MustCompile(`[^a-zA-Z0-9]`)
	return reg.ReplaceAllString(word, "")
}

// BuildLIKEPattern builds a LIKE pattern with proper escaping.
func BuildLIKEPattern(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return "%"
	}

	// Escape special LIKE characters
	input = strings.ReplaceAll(input, "\\", "\\\\")
	input = strings.ReplaceAll(input, "%", "\\%")
	input = strings.ReplaceAll(input, "_", "\\_")

	return "%" + input + "%"
}

// HighlightTerm highlights search terms in text with HTML marks.
func HighlightTerm(text, term string) string {
	if term == "" {
		return html.EscapeString(text)
	}

	// First escape HTML
	escaped := html.EscapeString(text)
	escapedTerm := html.EscapeString(term)

	// Case-insensitive replacement
	reg := regexp.MustCompile(`(?i)(` + regexp.QuoteMeta(escapedTerm) + `)`)
	return reg.ReplaceAllString(escaped, "<mark>$1</mark>")
}

// GetRecentMessages returns the most recent messages across all channels with pagination.
func (r *SearchRepository) GetRecentMessages(ctx context.Context, page, pageSize int) ([]MessageSearchResult, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// Count query
	countQuery := `SELECT COUNT(*) FROM messages`
	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count messages: %w", err)
	}

	// Main query
	query := `
		SELECT 
			m.id, m.channel_id, c.name, m.user_id, u.username, u.display_name,
			m.text, m.sent_at, m.tags
		FROM messages m
		JOIN channels c ON m.channel_id = c.id
		JOIN users u ON m.user_id = u.id
		ORDER BY m.sent_at DESC, m.id DESC
		LIMIT ` + r.ph(1) + ` OFFSET ` + r.ph(2) + `
	`

	rows, err := r.db.QueryContext(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get recent messages: %w", err)
	}
	defer rows.Close()

	var results []MessageSearchResult
	for rows.Next() {
		var m MessageSearchResult
		var sentAt any
		var displayName, tagsJSON sql.NullString

		if err := rows.Scan(
			&m.ID, &m.ChannelID, &m.ChannelName, &m.UserID, &m.Username, &displayName,
			&m.Text, &sentAt, &tagsJSON,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan message: %w", err)
		}

		m.SentAt = parseTimeValue(sentAt)
		if displayName.Valid {
			m.DisplayName = displayName.String
		}
		if tagsJSON.Valid && tagsJSON.String != "" {
			_ = json.Unmarshal([]byte(tagsJSON.String), &m.Tags)
		}
		m.HighlightedText = html.EscapeString(m.Text) // No highlighting for recent messages

		results = append(results, m)
	}

	return results, totalCount, rows.Err()
}
