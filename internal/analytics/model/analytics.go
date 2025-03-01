package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DocumentView struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	DocumentID uuid.UUID `gorm:"type:uuid;not null" json:"document_id"`
	UserID     uuid.UUID `gorm:"type:uuid" json:"user_id"` 
	IPAddress  string    `gorm:"type:varchar(45)" json:"ip_address"`
	UserAgent  string    `gorm:"type:varchar(255)" json:"user_agent"`
	ViewedAt   time.Time `gorm:"not null" json:"viewed_at"`
}

func (dv *DocumentView) BeforeCreate(tx *gorm.DB) error {
	if dv.ID == uuid.Nil {
		dv.ID = uuid.New()
	}
	return nil
}

type DocumentEdit struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	DocumentID uuid.UUID `gorm:"type:uuid;not null" json:"document_id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Version    int       `gorm:"not null" json:"version"`
	EditedAt   time.Time `gorm:"not null" json:"edited_at"`
}

func (de *DocumentEdit) BeforeCreate(tx *gorm.DB) error {
	if de.ID == uuid.Nil {
		de.ID = uuid.New()
	}
	return nil
}

type DocumentViewsResponse struct {
	Total       int64 `json:"total"`
	UniqueUsers int64 `json:"unique_users"`
	Timeline    []struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	} `json:"timeline"`
}

type DocumentEditsResponse struct {
	Total   int64 `json:"total"`
	ByUsers []struct {
		UserID   uuid.UUID `json:"user_id"`
		UserName string    `json:"user_name"`
		Count    int       `json:"count"`
	} `json:"by_users"`
	Timeline []struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	} `json:"timeline"`
}

// DocumentAnalyticsResponse represents the document analytics response
type DocumentAnalyticsResponse struct {
	Views DocumentViewsResponse `json:"views"`
	Edits DocumentEditsResponse `json:"edits"`
}

// UserAnalyticsDocumentResponse represents a document in the user analytics response
type UserAnalyticsDocumentResponse struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
	Views int       `json:"views"`
	Edits int       `json:"edits"`
}

// UserActivityResponse represents user activity for analytics
type UserActivityResponse struct {
	Views    int64 `json:"views"`
	Edits    int64 `json:"edits"`
	Timeline []struct {
		Date  string `json:"date"`
		Views int    `json:"views"`
		Edits int    `json:"edits"`
	} `json:"timeline"`
}

// UserDocumentsResponse represents user documents for analytics
type UserDocumentsResponse struct {
	Total       int `json:"total"`
	Created     int `json:"created"`
	Collaborated int `json:"collaborated"`
}

// UserAnalyticsResponse represents the user analytics response
type UserAnalyticsResponse struct {
	Documents          UserDocumentsResponse          `json:"documents"`
	Activity           UserActivityResponse           `json:"activity"`
	MostActiveDocuments []UserAnalyticsDocumentResponse `json:"most_active_documents"`
}