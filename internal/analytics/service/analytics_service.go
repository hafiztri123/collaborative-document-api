package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/hafiztri123/document-api/internal/analytics/model"
	"github.com/hafiztri123/document-api/internal/analytics/repository"
	"go.uber.org/zap"
)


type Service interface {
    RecordDocumentView(ctx context.Context, documentID, userID uuid.UUID, ipAddress, userAgent string) error
    GetDocumentViews(ctx context.Context, documentID uuid.UUID, period string) (*model.DocumentViewsResponse, error)
    RecordDocumentEdit(ctx context.Context, documentID, userID uuid.UUID, version int) error
    GetDocumentEdits(ctx context.Context, documentID uuid.UUID, period string) (*model.DocumentEditsResponse, error)
    GetUserAnalytics(ctx context.Context, userID uuid.UUID, period string) (*model.UserAnalyticsResponse, error)
}

type analyticsService struct {
	repo repository.Repository
	logger *zap.Logger
}

func NewAnalyticsService(repo repository.Repository, logger *zap.Logger) Service {
	return &analyticsService{
		repo: repo,
		logger: logger,
	}
}

func (s *analyticsService)   RecordDocumentView(ctx context.Context, documentID, userID uuid.UUID, ipAddress, userAgent string) error {
	return s.repo.RecordDocumentView(ctx, documentID, userID, ipAddress, userAgent)
}

func (s *analyticsService)    GetDocumentViews(ctx context.Context, documentID uuid.UUID, period string) (*model.DocumentViewsResponse, error){
	return s.repo.GetDocumentViews(ctx, documentID, period)

}

func (s *analyticsService)    RecordDocumentEdit(ctx context.Context, documentID, userID uuid.UUID, version int) error{
	return s.repo.RecordDocumentEdit(ctx, documentID, userID, version)

}

func (s *analyticsService)    GetDocumentEdits(ctx context.Context, documentID uuid.UUID, period string) (*model.DocumentEditsResponse, error){
	return s.repo.GetDocumentEdits(ctx, documentID, period)

}

func (s *analyticsService)    GetUserAnalytics(ctx context.Context, userID uuid.UUID, period string) (*model.UserAnalyticsResponse, error){
	documents, err := s.repo.GetUserDocumentsAnalytics(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get user document analytics", zap.Error(err))
		documents = &model.UserDocumentsResponse{}
	}

	activity, err := s.repo.GetUserActivityAnalytics(ctx, userID, period)
	if err != nil {
		s.logger.Error("Failed to get user acitivty analytics", zap.Error(err))
		activity = &model.UserActivityResponse{}
	}

	mostActive, err := s.repo.GetUserMostActiveDocuments(ctx, userID, 5)
	if err != nil {
		s.logger.Error("Failed to get user's most active documents", zap.Error(err))
		mostActive = []model.UserAnalyticsDocumentResponse{}
	}

	response := &model.UserAnalyticsResponse{
		Documents: *documents,
		Activity: *activity,
		MostActiveDocuments: mostActive,
	}

	return response, nil
}

