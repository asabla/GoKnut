package unit

import (
	"testing"

	"github.com/asabla/goknut/internal/http/dto"
)

func TestCreateChannelRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     dto.CreateChannelRequest
		wantErr error
	}{
		{
			name:    "valid channel",
			req:     dto.CreateChannelRequest{Name: "testchannel", Enabled: true},
			wantErr: nil,
		},
		{
			name:    "empty name",
			req:     dto.CreateChannelRequest{Name: "", Enabled: true},
			wantErr: dto.ErrChannelNameRequired,
		},
		{
			name:    "invalid characters",
			req:     dto.CreateChannelRequest{Name: "test@channel!", Enabled: true},
			wantErr: dto.ErrChannelNameInvalid,
		},
		{
			name:    "name too long",
			req:     dto.CreateChannelRequest{Name: "abcdefghijklmnopqrstuvwxyz0123", Enabled: true},
			wantErr: dto.ErrChannelNameTooLong,
		},
		{
			name:    "name with underscores",
			req:     dto.CreateChannelRequest{Name: "test_channel_123", Enabled: true},
			wantErr: nil,
		},
		{
			name:    "name normalized to lowercase",
			req:     dto.CreateChannelRequest{Name: "TestChannel", Enabled: true},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			// If valid, check normalization
			if err == nil && tt.req.Name != "" {
				// Name should be lowercase after validation
				for _, r := range tt.req.Name {
					if r >= 'A' && r <= 'Z' {
						t.Errorf("expected lowercase name, got %q", tt.req.Name)
						break
					}
				}
			}
		})
	}
}

func TestPaginationRequestValidation(t *testing.T) {
	tests := []struct {
		name         string
		req          dto.PaginationRequest
		wantErr      error
		wantPage     int
		wantPageSize int
	}{
		{
			name:         "valid pagination",
			req:          dto.PaginationRequest{Page: 1, PageSize: 20},
			wantErr:      nil,
			wantPage:     1,
			wantPageSize: 20,
		},
		{
			name:         "zero page defaults to 1",
			req:          dto.PaginationRequest{Page: 0, PageSize: 20},
			wantErr:      nil,
			wantPage:     1,
			wantPageSize: 20,
		},
		{
			name:         "zero page size defaults to 20",
			req:          dto.PaginationRequest{Page: 1, PageSize: 0},
			wantErr:      nil,
			wantPage:     1,
			wantPageSize: 20,
		},
		{
			name:         "page size over 100",
			req:          dto.PaginationRequest{Page: 1, PageSize: 150},
			wantErr:      dto.ErrPageSizeInvalid,
			wantPage:     1,
			wantPageSize: 150,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil {
				if tt.req.Page != tt.wantPage {
					t.Errorf("expected Page %d, got %d", tt.wantPage, tt.req.Page)
				}
				if tt.req.PageSize != tt.wantPageSize {
					t.Errorf("expected PageSize %d, got %d", tt.wantPageSize, tt.req.PageSize)
				}
			}
		})
	}
}

func TestSearchMessagesRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     dto.SearchMessagesRequest
		wantErr error
	}{
		{
			name:    "valid search",
			req:     dto.SearchMessagesRequest{Query: "hello world", PaginationRequest: dto.PaginationRequest{Page: 1, PageSize: 20}},
			wantErr: nil,
		},
		{
			name:    "query too short",
			req:     dto.SearchMessagesRequest{Query: "a", PaginationRequest: dto.PaginationRequest{Page: 1, PageSize: 20}},
			wantErr: dto.ErrSearchQueryTooShort,
		},
		{
			name:    "query too long",
			req:     dto.SearchMessagesRequest{Query: string(make([]byte, 101)), PaginationRequest: dto.PaginationRequest{Page: 1, PageSize: 20}},
			wantErr: dto.ErrSearchQueryTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
