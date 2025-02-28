package model

import (
	"time"

	"github.com/google/uuid"
	userModel "github.com/hafiztri123/document-api/internal/user/model"
	"gorm.io/gorm"
)

type Permission string

const (
	PermissionRead  Permission = "read"
	PermissionWrite Permission = "write"
)

type Collaborator struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	DocumentID uuid.UUID      `gorm:"type:uuid;not null" json:"document_id"`
	UserID     uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	User       userModel.User `gorm:"foreignKey:UserID" json:"user"`
	Permission Permission     `gorm:"type:varchar(20);not null" json:"permission"`
	CreatedAt  time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"not null" json:"updated_at"`
}

func (c *Collaborator) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// CollaboratorResponse represents the collaborator data returned to clients

type CollaboratorResponse struct {
	ID         uuid.UUID `json:"id"`
	DocumentID uuid.UUID `json:"document_id"`
	User       struct {
		ID    uuid.UUID `json:"id"`
		Name  string    `json:"name"`
		Email string    `json:"email"`
	} `json:"user"`
	Permission Permission `json:"permission"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at,omitempty"`
}

type CollaboratorCreateRequest struct {
	UserEmail  string     `json:"user_email" binding:"required,email"`
	Permission Permission `json:"permission" binding:"required,oneof=read write"`
}

type CollaboratorUpdateRequest struct {
	Permission Permission `json:"permission" binding:"required,oneof=read write"`
}




func (c *Collaborator) ToResponse() CollaboratorResponse {
	response := CollaboratorResponse{
		ID:         c.ID,
		DocumentID: c.DocumentID,
		User: struct {
			ID    uuid.UUID `json:"id"`
			Name  string    `json:"name"`
			Email string    `json:"email"`
		}{
			ID:    c.User.ID,
			Name:  c.User.Name,
			Email: c.User.Email,
		},
		Permission: c.Permission,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
	
	return response
}

