package model

import (
	"time"

	"github.com/google/uuid"
	userModel "github.com/hafiztri123/document-api/internal/user/model"
	"gorm.io/gorm"
)

// Document represents a document in the system
type Document struct {
	ID           	uuid.UUID     	 	`gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Title        	string        	 	`gorm:"type:varchar(255);not null" json:"title"`
	Content      	string        	 	`gorm:"type:text" json:"content"`
	Version      	int           	 	`gorm:"not null;default:1" json:"version"`
	IsPublic     	bool          	 	`gorm:"not null;default:false" json:"is_public"`
	OwnerID      	uuid.UUID     	 	`gorm:"type:uuid;not null" json:"owner_id"`
	Owner        	userModel.User	 	`gorm:"foreignKey:OwnerID" json:"-"`
	CreatedAt    	time.Time     	 	`gorm:"not null" json:"created_at"`
	UpdatedAt    	time.Time     	 	`gorm:"not null" json:"updated_at"`
	DeletedAt    	gorm.DeletedAt	 	`gorm:"index" json:"-"` // Soft delete
	Collaborators 	[]Collaborator	 	`gorm:"foreignKey:DocumentID" json:"collaborators,omitempty"`
	History     	[]DocumentHistory 	`gorm:"foreignKey:DocumentID" json:"-"`
}

func (d *Document) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	d.Version = 1
	return nil
}

func (d *Document) BeforeUpdate(tx *gorm.DB) error {
	d.Version++
	return nil
}

type DocumentHistory struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	DocumentID uuid.UUID      `gorm:"type:uuid;not null" json:"document_id"`
	Version    int            `gorm:"not null" json:"version"`
	Content    string         `gorm:"type:text" json:"content"`
	UpdatedByID uuid.UUID     `gorm:"type:uuid;not null" json:"updated_by_id"`
	UpdatedBy  userModel.User `gorm:"foreignKey:UpdatedByID" json:"updated_by"`
	UpdatedAt  time.Time      `gorm:"not null" json:"updated_at"`
}

type DocumentHistoryResponse struct {
	Version   int       `json:"version"`
	Content   string    `json:"content"`
	UpdatedBy struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	} `json:"updated_by"`
	UpdatedAt time.Time `json:"updated_at"`
}


type DocumentCreateRequest struct {
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content"`
	IsPublic bool   `json:"is_public"`
}

type DocumentUpdateRequest struct {
	Title    *string `json:"title"`
	Content  *string `json:"content"`
	IsPublic *bool   `json:"is_public"`
}



type DocumentListResponse struct {
	ID                uuid.UUID `json:"id"`
	Title             string    `json:"title"`
	Snippet           string    `json:"snippet"`
	Version           int       `json:"version"`
	IsPublic          bool      `json:"is_public"`
	OwnerID           uuid.UUID `json:"owner_id"`
	CollaboratorsCount int       `json:"collaborators_count"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ToListResponse converts a Document to a DocumentListResponse
func (d *Document) ToListResponse() DocumentListResponse {
	snippet := d.Content
	if len(snippet) > 150 {
		snippet = snippet[:150] + "..."
	}
	
	return DocumentListResponse{
		ID:                d.ID,
		Title:             d.Title,
		Snippet:           snippet,
		Version:           d.Version,
		IsPublic:          d.IsPublic,
		OwnerID:           d.OwnerID,
		CollaboratorsCount: len(d.Collaborators),
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
	}
}

