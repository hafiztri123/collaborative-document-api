package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	analyticsModel "github.com/hafiztri123/document-api/internal/analytics/model"
	analyticsRepo "github.com/hafiztri123/document-api/internal/analytics/repository"
	userRepo "github.com/hafiztri123/document-api/internal/auth/repository"
	"github.com/hafiztri123/document-api/internal/document/model"
	docRepo "github.com/hafiztri123/document-api/internal/document/repository"
	"go.uber.org/zap"
)

var (
	ErrDocumentNotFound      = errors.New("document not found")
	ErrUnauthorized          = errors.New("unauthorized access to document")
	ErrVersionNotFound       = errors.New("document version not found")
	ErrUserNotFound          = errors.New("user not found")
	ErrAlreadyCollaborator   = errors.New("user is already a collaborator")
	ErrNotCollaborator       = errors.New("user is not a collaborator")
	ErrCannotRemoveOwner     = errors.New("cannot remove document owner as collaborator")
)


type Service interface {
	// Document operations
	CreateDocument(ctx context.Context, ownerID uuid.UUID, req model.DocumentCreateRequest) (*model.Document, error)
	GetDocumentByID(ctx context.Context, id uuid.UUID, userID uuid.UUID, recordView bool, ipAddress, userAgent string) (*model.Document, error)
	GetUserDocuments(ctx context.Context, userID uuid.UUID, page, perPage int, sortBy, sortDir, query string) ([]*model.DocumentListResponse, int64, error)
	UpdateDocument(ctx context.Context, id uuid.UUID, userID uuid.UUID, req model.DocumentUpdateRequest) (*model.Document, error)
	DeleteDocument(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	
	// Document history operations
	GetDocumentHistory(ctx context.Context, documentID uuid.UUID, userID uuid.UUID, page, perPage int) ([]*model.DocumentHistoryResponse, int64, error)
	RestoreDocumentVersion(ctx context.Context, documentID uuid.UUID, userID uuid.UUID, version int) (*model.Document, error)
	
	// Collaboration operations
	ShareDocument(ctx context.Context, documentID uuid.UUID, ownerID uuid.UUID, req model.CollaboratorCreateRequest) (*model.CollaboratorResponse, error)
	UpdateCollaboratorPermission(ctx context.Context, documentID uuid.UUID, ownerID uuid.UUID, userID uuid.UUID, req model.CollaboratorUpdateRequest) (*model.CollaboratorResponse, error)
	RemoveCollaborator(ctx context.Context, documentID uuid.UUID, ownerID uuid.UUID, userID uuid.UUID) error
	
	// Analytics operations
	GetDocumentAnalytics(ctx context.Context, documentID uuid.UUID, userID uuid.UUID, period string) (*analyticsModel.DocumentAnalyticsResponse, error)
	GetUserAnalytics(ctx context.Context, userID uuid.UUID, period string) (*analyticsModel.UserAnalyticsResponse, error)
}

type documentService struct {
	docRepo       docRepo.Repository
	userRepo      userRepo.Repository
	analyticsRepo analyticsRepo.Repository
	logger        *zap.Logger
}

// NewDocumentService creates a new document service
func NewDocumentService(
	docRepo docRepo.Repository,
	userRepo userRepo.Repository,
	analyticsRepo analyticsRepo.Repository,
	logger *zap.Logger,
) Service {
	return &documentService{
		docRepo:       docRepo,
		userRepo:      userRepo,
		analyticsRepo: analyticsRepo,
		logger:        logger,
	}
}


func(s *documentService) 	CreateDocument(ctx context.Context, ownerID uuid.UUID, req model.DocumentCreateRequest) (*model.Document, error){
	document := &model.Document{
		Title: req.Title,
		Content: req.Content,
		IsPublic: req.IsPublic,
		OwnerID: ownerID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.docRepo.CreateDocument(ctx, document); err != nil {
		s.logger.Error("Failed to create document", zap.Error(err))
		return nil, err
	}

	history := &model.DocumentHistory{
		DocumentID: document.ID,
		Version: document.Version,
		Content: document.Content,
		UpdatedByID: ownerID,
		UpdatedAt: document.CreatedAt,
	}

	if err := s.docRepo.CreateDocumentHistory(ctx, history); err != nil {
		s.logger.Error("Failed to create document history", zap.Error(err))
		return document, nil
	}

	_ = s.analyticsRepo.RecordDocumentEdit(ctx, document.ID, ownerID, document.Version)

	return document ,nil
}


func(s *documentService)	GetDocumentByID(ctx context.Context, id uuid.UUID, userID uuid.UUID, recordView bool, ipAddress, userAgent string) (*model.Document, error){
	document, err := s.docRepo.GetDocumentByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get document by ID", zap.Error(err))
		return nil, err
	}

	if document == nil {
		return nil, ErrDocumentNotFound
	}

	canAccess, err := s.docRepo.CanUserAccess(ctx, id, userID, model.PermissionRead)
	if err != nil {
		s.logger.Error("Failed to check user access", zap.Error(err))
		return nil, err
	}

	if !canAccess {
		return nil, ErrUnauthorized
	}

	if recordView {
		_ = s.analyticsRepo.RecordDocumentView(ctx, id, userID, ipAddress, userAgent)
	}

	return document, nil
}


func(s *documentService)	GetUserDocuments(ctx context.Context, userID uuid.UUID, page, perPage int, sortBy, sortDir, query string) ([]*model.DocumentListResponse, int64, error){

	documents, total, err := s.docRepo.GetDocumentsByUserID(ctx, userID, page, perPage, sortBy, sortDir, query)
	if err != nil {
		s.logger.Error("Failed to get documents by user ID", zap.Error(err))
		return nil, 0, err
	}

	response := make([]*model.DocumentListResponse, 0, len(documents))
	for _, doc := range documents {
		listResp := doc.ToListResponse()
		response = append(response, &listResp)
	}

	return response, total, nil
}


func(s *documentService)	UpdateDocument(ctx context.Context, id uuid.UUID, userID uuid.UUID, req model.DocumentUpdateRequest) (*model.Document, error){
	document, err := s.docRepo.GetDocumentByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get document by ID", zap.Error(err))
		return nil, err
	}

	if document == nil {
		return nil, ErrDocumentNotFound
	}

	canWrite, err := s.docRepo.CanUserAccess(ctx, id, userID, model.PermissionWrite)
	if err != nil {
		s.logger.Error("Failed to check user access", zap.Error(err))
		return nil, err
	}
	if !canWrite {
		return nil, ErrUnauthorized
	}

	if req.Title != nil {
		document.Title = *req.Title
	}

	// var oldContent string
	var contentUpdated bool

	if req.Content != nil && *req.Content != document.Content {
		// oldContent = document.Content
		document.Content = *req.Content
		contentUpdated = true
	}

	if req.IsPublic != nil {
		document.IsPublic = *req.IsPublic
	}

	if contentUpdated {
		document.UpdatedAt = time.Now()
		if err := s.docRepo.UpdateDocument(ctx, document); err != nil {
			s.logger.Error("Failed to update document", zap.Error(err))
			return nil, err
		}

		history := &model.DocumentHistory{
			DocumentID: document.ID,
			Version: document.Version,
			Content: document.Content,
			UpdatedByID: userID,
			UpdatedAt: document.UpdatedAt,
		}

		if err := s.docRepo.CreateDocumentHistory(ctx, history); err != nil {
			s.logger.Error("Failed to create document history", zap.Error(err))
		}

		_ = s.analyticsRepo.RecordDocumentEdit(ctx, document.ID, userID, document.Version)
	} else if req.Title != nil || req.IsPublic != nil {
		document.UpdatedAt = time.Now()
		if err := s.docRepo.UpdateDocument(ctx, document); err != nil {
			s.logger.Error("Failed to update document metadata", zap.Error(err))
			return nil, err
		}
	}

	return document ,nil
}


func(s *documentService)	DeleteDocument(ctx context.Context, id uuid.UUID, userID uuid.UUID) error{
	document, err := s.docRepo.GetDocumentByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get document by ID", zap.Error(err))
		return err
	}

	if document == nil {
		return ErrDocumentNotFound
	}

	if document.OwnerID != userID {
		return ErrUnauthorized
	}

	if err := s.docRepo.DeleteDocument(ctx, id); err != nil {
		s.logger.Error("Failed to delete document", zap.Error(err))
		return err
	}

	return nil
}


func(s *documentService)	GetDocumentHistory(ctx context.Context, documentID uuid.UUID, userID uuid.UUID, page, perPage int) ([]*model.DocumentHistoryResponse, int64, error){
	canAccess, err := s.docRepo.CanUserAccess(ctx, documentID, userID, model.PermissionRead)
	if err != nil {
		s.logger.Error("Failed to check user access", zap.Error(err))
		return nil, 0, err
	}
	if !canAccess {
		return nil, 0, ErrUnauthorized
	}

	history, total, err := s.docRepo.GetDocumentHistory(ctx, documentID, page, perPage)
	if err != nil {
		s.logger.Error("Failed to get document history", zap.Error(err))
		return nil, 0, err
	}

	response := make([]*model.DocumentHistoryResponse, 0, len(history))
	for _, h := range history {
		resp := &model.DocumentHistoryResponse{
			Version: h.Version,
			Content: h.Content,
			UpdatedBy: struct {
				ID uuid.UUID `json:"id"`
				Name string `json:"name"`
			}{
				ID: h.UpdatedByID,
				Name: h.UpdatedBy.Name,
			},
			UpdatedAt: h.UpdatedAt,
		}
		response = append(response, resp)
	}

	return response, total, nil
}


func(s *documentService)	RestoreDocumentVersion(ctx context.Context, documentID uuid.UUID, userID uuid.UUID, version int) (*model.Document, error){
	canWrite, err := s.docRepo.CanUserAccess(ctx, documentID, userID, model.PermissionWrite)
	if err != nil {
		s.logger.Error("Failed to check user access", zap.Error(err))
		return nil, err
	}
	if !canWrite {
		return nil, ErrUnauthorized
	}

	document, err := s.docRepo.GetDocumentByID(ctx, documentID)
	if err != nil {
		s.logger.Error("Failed to get document by ID", zap.Error(err))
		return nil, err
	}

	if document == nil {
		return nil, ErrDocumentNotFound
	}

	history, err := s.docRepo.GetDocumentHistoryByVersion(ctx, documentID, version)
	if err != nil {
		s.logger.Error("Failed to get document history by version", zap.Error(err))
		return nil, err
	}

	if history == nil {
		return nil, ErrVersionNotFound
	}

	document.Content = history.Content
	document.UpdatedAt = time.Now()

	if err := s.docRepo.UpdateDocument(ctx, document); err != nil {
		s.logger.Error("Failed to update document", zap.Error(err))
		return nil, err
	}

	newHistory := &model.DocumentHistory{
		DocumentID: document.ID,
		Version: document.Version,
		Content: document.Content,
		UpdatedByID: userID,
		UpdatedAt: document.UpdatedAt,
	}

	if err := s.docRepo.CreateDocumentHistory(ctx, newHistory); err != nil {
		s.logger.Error("Failed to create document history", zap.Error(err))
	}

	_ = s.analyticsRepo.RecordDocumentEdit(ctx, document.ID, userID, document.Version)

	return document, nil

}


func(s *documentService)	ShareDocument(ctx context.Context, documentID uuid.UUID, ownerID uuid.UUID, req model.CollaboratorCreateRequest) (*model.CollaboratorResponse, error){
	document, err := s.docRepo.GetDocumentByID(ctx, documentID)
	if err != nil {
		s.logger.Error("Failed to get document by ID", zap.Error(err))	
		return nil, err
	}

	if document == nil {
		return nil, ErrDocumentNotFound
	}

	if document.OwnerID != ownerID {
		return nil, ErrUnauthorized
	}

	user, err := s.userRepo.FindUserByEmail(ctx, req.UserEmail)
	if err != nil {
		s.logger.Error("Failed to find user by email", zap.Error(err))
		return nil, err
	}

	if user == nil {
		return nil, ErrUserNotFound
	}

	existing, err := s.docRepo.GetCollaborator(ctx, documentID, user.ID)
	if err != nil {
		s.logger.Error("Failed to get collaborator", zap.Error(err))
		return nil, err
	}

	if existing != nil {
		return nil, ErrAlreadyCollaborator
	}

	collaborator := &model.Collaborator{
		DocumentID: documentID,
		UserID: user.ID,
		User: *user,
		Permission: req.Permission,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.docRepo.AddCollaborator(ctx, collaborator); err != nil {
		s.logger.Error("Failed to add collaborator", zap.Error(err))
		return nil, err
	}

	response := collaborator.ToResponse()
	return &response, nil

}


func(s *documentService)	UpdateCollaboratorPermission(ctx context.Context, documentID uuid.UUID, ownerID uuid.UUID, userID uuid.UUID, req model.CollaboratorUpdateRequest) (*model.CollaboratorResponse, error){
	document, err := s.docRepo.GetDocumentByID(ctx, documentID)
	if err != nil {
		s.logger.Error("Failed to get document by ID", zap.Error(err))
		return nil, err
	}
	if document == nil {
		return nil, ErrDocumentNotFound
	}

	if document.OwnerID != ownerID {
		return nil, ErrUnauthorized
	}

	collaborator, err := s.docRepo.GetCollaborator(ctx, documentID, userID)
	if err != nil {
		s.logger.Error("Failed to get collaborator", zap.Error(err))
		return nil, err
	}
	if collaborator == nil {
		return nil, ErrNotCollaborator
	}

	collaborator.Permission = req.Permission
	collaborator.UpdatedAt = time.Now()

	if err := s.docRepo.UpdateCollaborator(ctx, collaborator); err != nil {
		s.logger.Error("Failed to updated collaborator", zap.Error(err))
		return nil, err
	}

	response := collaborator.ToResponse()
	return &response, nil

}


func(s *documentService)	RemoveCollaborator(ctx context.Context, documentID uuid.UUID, ownerID uuid.UUID, userID uuid.UUID) error{
	document, err := s.docRepo.GetDocumentByID(ctx, documentID)
	if err != nil {
		s.logger.Error("Failed to get document by ID", zap.Error(err))
		return err
	}
	if document == nil {
		return ErrDocumentNotFound
	}

	if document.OwnerID != ownerID {
		return ErrUnauthorized
	}

	if document.OwnerID == userID {
		return ErrCannotRemoveOwner
	}

	if err := s.docRepo.RemoveCollaborator(ctx, documentID, userID); err != nil {
		s.logger.Error("Failed to remove collaborator", zap.Error(err))
		return err
	}

	return nil

}


func(s *documentService)	GetDocumentAnalytics(ctx context.Context, documentID uuid.UUID, userID uuid.UUID, period string) (*analyticsModel.DocumentAnalyticsResponse, error){
	canAcess, err := s.docRepo.CanUserAccess(ctx, documentID, userID, model.PermissionRead)
	if err != nil {
		s.logger.Error("Failed to check user access", zap.Error(err))
		return nil, err
	}

	if !canAcess {
		return nil, ErrUnauthorized
	}

	views, err := s.analyticsRepo.GetDocumentViews(ctx, documentID, period)
	if err != nil {
		s.logger.Error("Failed to get document views", zap.Error(err))
		views = &analyticsModel.DocumentViewsResponse{}
	}

	edits, err := s.analyticsRepo.GetDocumentEdits(ctx, documentID, period)
	if err != nil {
		s.logger.Error("Failed to get document edits", zap.Error(err))
		edits = &analyticsModel.DocumentEditsResponse{}
	}

	response := &analyticsModel.DocumentAnalyticsResponse{
		Views: *views,
		Edits: *edits,
	}

	return response, nil

}


func(s *documentService)	GetUserAnalytics(ctx context.Context, userID uuid.UUID, period string) (*analyticsModel.UserAnalyticsResponse, error){
	documents, err := s.analyticsRepo.GetUserDocumentsAnalytics(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get user document analytics", zap.Error(err))
		documents = &analyticsModel.UserDocumentsResponse{}
	}

	activity, err := s.analyticsRepo.GetUserActivityAnalytics(ctx, userID, period)
	if err != nil {
		s.logger.Error("Failed to get user activity analytics", zap.Error(err))
		activity = &analyticsModel.UserActivityResponse{}
	}

	mostActive, err := s.analyticsRepo.GetUserMostActiveDocuments(ctx, userID, 5)
	if err != nil {
		s.logger.Error("Failed to get user's most active documents", zap.Error(err))
		mostActive = []analyticsModel.UserAnalyticsDocumentResponse{}
	}

	response := &analyticsModel.UserAnalyticsResponse{
		Documents: *documents,
		Activity: *activity,
		MostActiveDocuments: mostActive,
	}

	return response, nil
}




