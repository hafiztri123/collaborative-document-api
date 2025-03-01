package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hafiztri123/document-api/internal/analytics/model"
	documentModel "github.com/hafiztri123/document-api/internal/document/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Repository interface {
	// Document view tracking
	RecordDocumentView(ctx context.Context, documentID, userID uuid.UUID, ipAddress, userAgent string) error
	GetDocumentViews(ctx context.Context, documentID uuid.UUID, period string) (*model.DocumentViewsResponse, error)
	
	// Document edit tracking
	RecordDocumentEdit(ctx context.Context, documentID, userID uuid.UUID, version int) error
	GetDocumentEdits(ctx context.Context, documentID uuid.UUID, period string) (*model.DocumentEditsResponse, error)
	
	// User analytics
	GetUserDocumentsAnalytics(ctx context.Context, userID uuid.UUID) (*model.UserDocumentsResponse, error)
	GetUserActivityAnalytics(ctx context.Context, userID uuid.UUID, period string) (*model.UserActivityResponse, error)
	GetUserMostActiveDocuments(ctx context.Context, userID uuid.UUID, limit int) ([]model.UserAnalyticsDocumentResponse, error)
}

type analyticsRepository struct {
	db *gorm.DB
	logger *zap.Logger
}

func NewAnalyticsRepository (db *gorm.DB, logger *zap.Logger) Repository {
	return &analyticsRepository{
		db: db,
		logger: logger,
	}
}



	// Document view tracking
func (r *analyticsRepository) RecordDocumentView(ctx context.Context, documentID, userID uuid.UUID, ipAddress, userAgent string) error {
	view := model.DocumentView{
		DocumentID: documentID,
		UserID: userID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		ViewedAt: time.Now(),
	}

	err := r.db.WithContext(ctx).Create(&view).Error
	if err != nil {
		r.logger.Error("Failed to record document view", zap.Error(err))
		return err
	}

	return nil
	
}
func (r *analyticsRepository)	GetDocumentViews(ctx context.Context, documentID uuid.UUID, period string) (*model.DocumentViewsResponse, error) {
	response := &model.DocumentViewsResponse{
		Timeline: []struct {
			Date string `json:"date"`
			Count int `json:"count"`
		}{},
	}

	now := time.Now()
	var startTime time.Time
	var groupFormat string

switch period {
	case "day":
		startTime = now.AddDate(0, 0, -1)
		groupFormat = "YYYY-MM-DD HH24:00"
	case "week":
		startTime = now.AddDate(0, 0, -7)
		groupFormat = "YYYY-MM-DD"
	case "year":
		startTime = now.AddDate(-1, 0, 0)
		groupFormat = "YYYY-MM"
	default:
		//Default: month
		startTime = now.AddDate(0, -1, 0)
		groupFormat = "YYYY-MM-DD"
	}

	//total views
	err := r.db.WithContext(ctx).
		Model(&model.DocumentView{}).
		Where("document_id = ? AND viewed_at >= ?", documentID, startTime).
		Count(&response.Total).Error

	if err != nil {
		r.logger.Error("Failed to get total document views", zap.Error(err))
		return nil, err
	}


	//unique users
	if err := r.db.WithContext(ctx).Model(&model.DocumentView{}).
		Where("document_id = ? AND viewed_at >= ? AND user_id IS NOT NULL", documentID, startTime).
		Distinct("user_id").
		Count(&response.UniqueUsers).Error; err != nil {
			r.logger.Error("Failed to get unique users for document views", zap.Error(err))
			return nil, err
		}

	type TimelineResult struct {
		Date string
		Count int
	}

	var timelineResults []TimelineResult

	if err := r.db.WithContext(ctx).Raw(`
		SELECT TO_CHAR(viewed_at, ?) as date, COUNT(*) as count
		FROM document_views
		WHERE document_id = ? AND viewed_at = >= ?
		GROUP BY date
		ORDER BY date
	`, groupFormat, documentID, startTime).Scan(&timelineResults).Error; err != nil {
		r.logger.Error("Failed to get document views timeline", zap.Error(err))
		return nil, err
	}

	for _, result := range timelineResults {
		response.Timeline = append(response.Timeline, struct {
			Date string `json:"date"`
			Count int 	`json:"count"`
		}{
			Date: result.Date,
			Count: result.Count,
		} )
	}

	return response, nil
}
func (r *analyticsRepository)	RecordDocumentEdit(ctx context.Context, documentID, userID uuid.UUID, version int) error {
	edit := model.DocumentEdit{
		DocumentID: documentID,
		UserID: userID,
		Version: version,
		EditedAt: time.Now(),
	}

	if err := r.db.WithContext(ctx).Create(&edit).Error; err != nil {
		r.logger.Error("Failed to record document edit", zap.Error(err))
		return err
	}

	return nil

}
func (r *analyticsRepository)	GetDocumentEdits(ctx context.Context, documentID uuid.UUID, period string) (*model.DocumentEditsResponse, error) {
	response := &model.DocumentEditsResponse{
		ByUsers: []struct {
			UserID uuid.UUID `json:"user_id"`
			UserName string `json:"user_name"`
			Count int `json:"count"`
		}{},
		Timeline: []struct {
			Date string `json:"date"`
			Count int `json:"count"`
		}{},
	}

	now := time.Now()
	var startTime time.Time
	var groupFormat string

		switch period {
	case "day":
		startTime = now.AddDate(0, 0, -1)
		groupFormat = "YYYY-MM-DD HH24:00"
	case "week":
		startTime = now.AddDate(0, 0, -7)
		groupFormat = "YYYY-MM-DD"
	case "year":
		startTime = now.AddDate(-1, 0, 0)
		groupFormat = "YYYY-MM"
	default:
		// Default to month
		startTime = now.AddDate(0, -1, 0)
		groupFormat = "YYYY-MM-DD"
	}

	if err := r.db.WithContext(ctx).
	Model(&model.DocumentEdit{}).
	Where("document_id = ? AND edited_at >= ?", documentID, startTime).
	Count(&response.Total).Error; err != nil {
		r.logger.Error("Failed to get total document edits", zap.Error(err))
		return nil, err
	} 

	type UserEditResult struct {
		UserID uuid.UUID
		UserName string
		Count int
	}

	var userEditResults []UserEditResult
	if err := r.db.WithContext(ctx).Raw(`
		SELECT de.user_id, u.name as user_name, COUNT(*) as count
		FROM document_edits de
		JOIN users u ON de.user_id = u.id
		WHERE de.document_id = ? AND de.edited_at >= ?
		GROUP BY de.user_id, u.name
		ORDER BY count DESC
	`, documentID, startTime).Scan(&userEditResults).Error; err != nil {
		r.logger.Error("Failed to get document edits by user", zap.Error(err))
		return nil, err
	}

	for _, result := range userEditResults {
		response.ByUsers = append(response.ByUsers, struct {
			UserID uuid.UUID `json:"user_id"`
			UserName string `json:"user_name"`
			Count int `json:"count"`
		}{
			UserID: result.UserID,
			UserName: result.UserName,
			Count: result.Count,
		})
	}

	type TimelineResult struct {
		Date string
		Count int
	}

	var timelineResults []TimelineResult
	if err := r.db.WithContext(ctx).Raw(`
		SELECT TO_CHAR(edited_at, ?) as date, COUNT(*) as count
		FROM document_edits
		WHERE document_id = ? AND edited_at >= ?
		GROUP BY date
		ORDER BY date
	`, groupFormat, documentID, startTime).Scan(&timelineResults).Error; err != nil {
		r.logger.Error("Failed to get document edits timeline", zap.Error(err))
		return nil, err
	}

	for _, result := range timelineResults {
		response.Timeline = append(response.Timeline, struct {
			Date string `json:"date"`
			Count int `json:"count"`
		}{
			Date: result.Date,
			Count: result.Count,
		})
	}

	return response, nil

	
}
func (r *analyticsRepository)	GetUserDocumentsAnalytics(ctx context.Context, userID uuid.UUID) (*model.UserDocumentsResponse, error) {
	response := &model.UserDocumentsResponse{}

	var docsCreated int64
	if err := r.db.WithContext(ctx).Model(&documentModel.Document{}).Where("owner_id = ?", userID).Count(&docsCreated).Error; err != nil {
		r.logger.Error("Failed to count user created documents", zap.Error(err))
		return nil, err
	}

	var docsCollaborated int64
	if err := r.db.WithContext(ctx).Model(&documentModel.Collaborator{}).Where("user_id = ?", userID).Distinct("document_id").Count(&docsCollaborated).Error; err != nil {
		r.logger.Error("Failed to count user collaborated documents", zap.Error(err))
		return nil, err
	}

	response.Total = int(docsCreated + docsCollaborated)
	response.Created = int(docsCreated)
	response.Collaborated = int(docsCollaborated)

	return response, nil


}

func (r *analyticsRepository) GetUserActivityAnalytics(ctx context.Context, userID uuid.UUID, period string) (*model.UserActivityResponse, error) {
	response := &model.UserActivityResponse{
		Timeline: []struct {
			Date  string `json:"date"`
			Views int    `json:"views"`
			Edits int    `json:"edits"`
		}{},
	}
	
	// Calculate time range based on period
	now := time.Now()
	var startTime time.Time
	var groupFormat string
	
	switch period {
	case "day":
		startTime = now.AddDate(0, 0, -1)
		groupFormat = "YYYY-MM-DD HH24:00"
	case "week":
		startTime = now.AddDate(0, 0, -7)
		groupFormat = "YYYY-MM-DD"
	case "year":
		startTime = now.AddDate(-1, 0, 0)
		groupFormat = "YYYY-MM"
	default:
		// Default to month
		startTime = now.AddDate(0, -1, 0)
		groupFormat = "YYYY-MM-DD"
	}
	
	// Get total views
	if err := r.db.WithContext(ctx).
		Model(&model.DocumentView{}).
		Where("user_id = ? AND viewed_at >= ?", userID, startTime).
		Count(&response.Views).Error; err != nil {
		r.logger.Error("Failed to get total user views", zap.Error(err))
		return nil, err
	}
	
	// Get total edits
	if err := r.db.WithContext(ctx).
		Model(&model.DocumentEdit{}).
		Where("user_id = ? AND edited_at >= ?", userID, startTime).
		Count(&response.Edits).Error; err != nil {
		r.logger.Error("Failed to get total user edits", zap.Error(err))
		return nil, err
	}
	
	// Get timeline data combining views and edits
	type TimelineResult struct {
		Date  string
		Views int
		Edits int
	}
	
	var timelineResults []TimelineResult
	if err := r.db.WithContext(ctx).Raw(`
		SELECT 
			dates.date,
			COALESCE(views.view_count, 0) as views,
			COALESCE(edits.edit_count, 0) as edits
		FROM (
			SELECT DISTINCT TO_CHAR(date_series, ?) as date
			FROM generate_series(?, CURRENT_DATE, '1 day'::interval) as date_series
		) dates
		LEFT JOIN (
			SELECT TO_CHAR(viewed_at, ?) as date, COUNT(*) as view_count
			FROM document_views
			WHERE user_id = ? AND viewed_at >= ?
			GROUP BY date
		) views ON dates.date = views.date
		LEFT JOIN (
			SELECT TO_CHAR(edited_at, ?) as date, COUNT(*) as edit_count
			FROM document_edits
			WHERE user_id = ? AND edited_at >= ?
			GROUP BY date
		) edits ON dates.date = edits.date
		ORDER BY dates.date
	`, groupFormat, startTime, groupFormat, userID, startTime, groupFormat, userID, startTime).Scan(&timelineResults).Error; err != nil {
		r.logger.Error("Failed to get user activity timeline", zap.Error(err))
		return nil, err
	}
	
	// Convert timeline results to response format
	for _, result := range timelineResults {
		response.Timeline = append(response.Timeline, struct {
			Date  string `json:"date"`
			Views int    `json:"views"`
			Edits int    `json:"edits"`
		}{
			Date:  result.Date,
			Views: result.Views,
			Edits: result.Edits,
		})
	}
	
	return response, nil
}

func (r *analyticsRepository)	GetUserMostActiveDocuments(ctx context.Context, userID uuid.UUID, limit int) ([]model.UserAnalyticsDocumentResponse, error) {
	var response []model.UserAnalyticsDocumentResponse
	
	// Set default limit if not provided
	if limit <= 0 {
		limit = 5
	}
	
	// Use a raw query to get documents with combined view and edit counts
	if err := r.db.WithContext(ctx).Raw(`
		SELECT 
			d.id,
			d.title,
			COALESCE(v.view_count, 0) as views,
			COALESCE(e.edit_count, 0) as edits
		FROM documents d
		LEFT JOIN (
			SELECT document_id, COUNT(*) as view_count
			FROM document_views
			WHERE viewed_at >= NOW() - INTERVAL '30 days'
			GROUP BY document_id
		) v ON d.id = v.document_id
		LEFT JOIN (
			SELECT document_id, COUNT(*) as edit_count
			FROM document_edits
			WHERE edited_at >= NOW() - INTERVAL '30 days'
			GROUP BY document_id
		) e ON d.id = e.document_id
		WHERE d.owner_id = ? 
		   OR d.id IN (
			SELECT document_id 
			FROM collaborators 
			WHERE user_id = ?
		   )
		ORDER BY (COALESCE(v.view_count, 0) + COALESCE(e.edit_count, 0) * 2) DESC
		LIMIT ?
	`, userID, userID, limit).Scan(&response).Error; err != nil {
		r.logger.Error("Failed to get user's most active documents", zap.Error(err))
		return nil, err
	}
	
	return response, nil
}
