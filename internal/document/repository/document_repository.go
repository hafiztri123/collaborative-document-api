package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hafiztri123/document-api/internal/document/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)


type Repository interface {
	CreateDocument(ctx context.Context, document *model.Document) error
	GetDocumentByID(ctx context.Context, id uuid.UUID) (*model.Document, error)
	GetDocumentsByUserID(ctx context.Context, userID uuid.UUID, page, perPage int, sortBy string, sortDir string, query string) ([]*model.Document, int64, error)
	UpdateDocument(ctx context.Context, document *model.Document) error
	DeleteDocument(ctx context.Context, id uuid.UUID) error
	
	CreateDocumentHistory(ctx context.Context, history *model.DocumentHistory) error
	GetDocumentHistory(ctx context.Context, documentID uuid.UUID, page, perPage int) ([]*model.DocumentHistory, int64, error)
	GetDocumentHistoryByVersion(ctx context.Context, documentID uuid.UUID, version int) (*model.DocumentHistory, error)
	
	AddCollaborator(ctx context.Context, collaborator *model.Collaborator) error
	UpdateCollaborator(ctx context.Context, collaborator *model.Collaborator) error
	RemoveCollaborator(ctx context.Context, documentID, userID uuid.UUID) error
	GetCollaborators(ctx context.Context, documentID uuid.UUID) ([]*model.Collaborator, error)
	GetCollaborator(ctx context.Context, documentID, userID uuid.UUID) (*model.Collaborator, error)
	
	CanUserAccess(ctx context.Context, documentID, userID uuid.UUID, requiredPermission model.Permission) (bool, error)
}

type documentRepository struct {
	db 		*gorm.DB
	logger 	*zap.Logger
}

func NewDocumentRepository(db *gorm.DB, logger *zap.Logger) Repository {
	return &documentRepository{
		db: db,
		logger: logger,
	}
}

func (r *documentRepository) CreateDocument(ctx context.Context, document *model.Document) error {
	err := r.db.WithContext(ctx).Create(document).Error
	if err != nil {
		r.logger.Error("Failed to create document", zap.Error(err))
		return err
	}
	return nil
}


func (r *documentRepository)	GetDocumentByID(ctx context.Context, id uuid.UUID) (*model.Document, error){
	var document model.Document
	err := r.db.WithContext(ctx).Preload("Collaborators.User").Where("id = ?", id).First(&document).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("Failed to get document by ID", zap.Error(err))
		return nil, err
	}
	return &document, nil
}

func (r *documentRepository)	GetDocumentsByUserID(ctx context.Context, userID uuid.UUID, page, perPage int, sortBy string, sortDir string, query string) ([]*model.Document, int64, error){
	var documents []*model.Document
	var total int64

	db := r.db.WithContext(ctx).Model(&model.Document{})

	//So it search both for documents that user owned and user collaborated but not necessarily own
	db = db.Where("owner_id", userID).
		Or(
			"id IN (?)", 
			r.db.Model(&model.Collaborator{}).
			Select("document_id").
			Where("user_id = ?", userID))
	
	if query != "" {
		db = db.Where("title ILIKE ? OR content ILIKE ?", "%"+query+"%", "%"+query+"%") //search with case insensitive
	}

	if err := db.Count(&total).Error;  err != nil{
		r.logger.Error("Failed to count documents", zap.Error(err))
		return nil, 0, err
	}

	if sortBy == "" {
		sortBy = "updated_at"
	}

	if sortDir == "" {
		sortDir = "desc"
	}

	order := fmt.Sprintf("%s %s", sortBy, sortDir)

	if page < 1 {
		page  = 1
	}


	if perPage < 1 {
		perPage = 20
	}

	/*
	number of item skipped
	example: Item per page == 20

	Page 1 -> Item 1 - 20
	Page 2 -> Item 21 - 40

	Each page skipped 20 items which means
	page - 1 * perPage

	Page 1
	(1 - 1) * 20 = 0
	(2 - 1) * 20 = 20
	*/

	offset := (page - 1) * perPage

	if err := db.Order(order).Limit(perPage).Offset(offset).Preload("Collaborators").Find(&documents).Error; err != nil {
		r.logger.Error("Failed to get documents by User ID", zap.Error(err))
		return nil, 0, err
	}

	return documents, total, nil

}
func (r *documentRepository)	UpdateDocument(ctx context.Context, document *model.Document) error{
	err := r.db.WithContext(ctx).Save(document).Error
	if err != nil {
		r.logger.Error("Failed to update document", zap.Error(err))
		return err
	}
	return nil
}
func (r *documentRepository)	DeleteDocument(ctx context.Context, id uuid.UUID) error{
	err := r.db.WithContext(ctx).Delete(&model.Document{}, id).Error
	if err != nil {
		r.logger.Error("Failed to delete document", zap.Error(err))
		return err
	}
	return nil

}
func (r *documentRepository)	CreateDocumentHistory(ctx context.Context, history *model.DocumentHistory) error{
	if err := r.db.Create(history).Error; err != nil {
		r.logger.Error("Failed to create document history", zap.Error(err))
		return err
	}

	return nil

}
func (r *documentRepository)	GetDocumentHistory(ctx context.Context, documentID uuid.UUID, page, perPage int) ([]*model.DocumentHistory, int64, error){
	var historyDocuments []*model.DocumentHistory
	var total int64
	
	err := r.db.WithContext(ctx).
		Model(&model.DocumentHistory{}).
		Where("document_id = ?", documentID).
		Count(&total).Error

	if err != nil {
		r.logger.Error("Failed to count document history", zap.Error(err))
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}

	if perPage < 1 {
		perPage = 20
	}

	offset := (page - 1) * perPage

	err = r.db.WithContext(ctx).
		Where("document_id = ?", documentID).
		Order("version DESC").
		Limit(perPage).
		Offset(offset).
		Preload("UpdatedBy").
		Find(&historyDocuments).
		Error
	
	if err != nil{
		r.logger.Error("Failed to get document history", zap.Error(err))
		return nil, 0, err
	}

	return historyDocuments, total, nil
}
func (r *documentRepository)	GetDocumentHistoryByVersion(ctx context.Context, documentID uuid.UUID, version int) (*model.DocumentHistory, error){
	var history model.DocumentHistory

	err := r.db.WithContext(ctx).Where("document_id = ? AND version = ?", documentID, version).Preload("UpdatedBy").First(&history).Error
	
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("Failed to get document history by version", zap.Error(err))
		return nil, err
	}

	return &history, nil
}
func (r *documentRepository)	AddCollaborator(ctx context.Context, collaborator *model.Collaborator) error{
	err := r.db.WithContext(ctx).Create(collaborator).Error
	if err != nil {
		r.logger.Error("Failed to add collaborator", zap.Error(err))
		return err
	}
	return nil
}
func (r *documentRepository)	UpdateCollaborator(ctx context.Context, collaborator *model.Collaborator) error{
	err := r.db.WithContext(ctx).Save(collaborator).Error
	if err != nil {
		r.logger.Error("Failed to update collaborator", zap.Error(err))
		return err
	}
	return nil
}
func (r *documentRepository)	RemoveCollaborator(ctx context.Context, documentID, userID uuid.UUID) error{
	err := r.db.WithContext(ctx).Where("document_id = ? AND user_id = ?", documentID, userID).Delete(&model.Collaborator{}).Error
	if err != nil {
		r.logger.Error("Failed to remove collaborator", zap.Error(err))
		return err
	}

	return nil

}
func (r *documentRepository)	GetCollaborators(ctx context.Context, documentID uuid.UUID) ([]*model.Collaborator, error){
	var collaborators []*model.Collaborator

	err := r.db.WithContext(ctx).Where("document_id = ?", documentID).Preload("User").Find(&collaborators).Error
	if err != nil {
		r.logger.Error("Failed to get collaborators", zap.Error(err))
		return nil, err
	}

	return collaborators, nil

}
func (r *documentRepository)	GetCollaborator(ctx context.Context, documentID, userID uuid.UUID) (*model.Collaborator, error){
	var collaborator model.Collaborator

	err := r.db.WithContext(ctx).Where("document_id = ? AND user_id = ?", documentID, userID).Preload("User").First(&collaborator).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		r.logger.Error("Failed to get collaborator", zap.Error(err))
		return nil, err
	}

	return &collaborator, nil
}

func (r *documentRepository) CanUserAccess(ctx context.Context, documentID, userID uuid.UUID, requiredPermission model.Permission) (bool, error) {
	//check ownership by count document with id and user id
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Document{}).Where("id = ? AND owner_id = ?", documentID, userID).Count(&count).Error
	if err != nil {
		r.logger.Error("Failed to check document ownership", zap.Error(err))
		return false,err
	}

	//if document is owned by user then return true
	if count > 0 {
		return true, nil
	}

	/*
	if document is public and has permission for read only, then everyone can access it
	*/

	if requiredPermission == model.PermissionRead {
		var isPublic bool
		err := r.db.WithContext(ctx).Model(&model.Document{}).Select("is_public").Where("id = ?", documentID).First(&isPublic).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, nil
			}
			r.logger.Error("Failed to check if document is public", zap.Error(err))
			return false, err
		}

		if isPublic {
			return true, nil
		}
	}

	//if user is collaborator, then even if its not public, they have the required permission
	var collaborator model.Collaborator
	err = r.db.WithContext(ctx).Where("document_id = ? AND user_id = ?", documentID, userID).First(&collaborator).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		r.logger.Error("Failed to check collaborator permissions", zap.Error(err))
		return false, err
	}

	if requiredPermission == model.PermissionRead {
		return true, nil
	}

	return collaborator.Permission == model.PermissionWrite, nil
}
