package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/hafiztri123/document-api/internal/document/model"
	"github.com/hafiztri123/document-api/internal/document/service"
)

type Controller interface {
	CreateDocument(c *gin.Context)
	GetDocuments(c *gin.Context)
	GetDocumentByID(c *gin.Context)
	UpdateDocument(c *gin.Context)
	DeleteDocument(c *gin.Context)
	
	GetDocumentHistory(c *gin.Context)
	RestoreDocumentVersion(c *gin.Context)
	
	ShareDocument(c *gin.Context)
	UpdateCollaboratorPermission(c *gin.Context)
	RemoveCollaborator(c *gin.Context)
	
	GetDocumentAnalytics(c *gin.Context)
	GetUserAnalytics(c *gin.Context)
}

type documentController struct {
	service service.Service
	logger  *zap.Logger
}

func NewDocumentController(service service.Service, logger *zap.Logger) Controller {
	return &documentController{
		service: service,
		logger:  logger,
	}
}

func (ctrl *documentController) CreateDocument(c *gin.Context) {
	var req model.DocumentCreateRequest
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "validation_error",
			"message": "Invalid request data",
			"details": err.Error(),
		}})
		return
	}
	
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
			"code":    "unauthorized",
			"message": "User not authenticated",
		}})
		return
	}
	
	document, err := ctrl.service.CreateDocument(c.Request.Context(), userID.(uuid.UUID), req)
	if err != nil {
		ctrl.logger.Error("Failed to create document", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to create document",
		}})
		return
	}
	
	c.JSON(http.StatusCreated, document)
}

func (ctrl *documentController) GetDocuments(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
			"code":    "unauthorized",
			"message": "User not authenticated",
		}})
		return
	}
	
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	
	sortBy := c.DefaultQuery("sort_by", "updated_at")
	sortDir := c.DefaultQuery("sort_dir", "desc")
	
	query := c.DefaultQuery("q", "")
	
	documents, total, err := ctrl.service.GetUserDocuments(
		c.Request.Context(),
		userID.(uuid.UUID),
		page,
		perPage,
		sortBy,
		sortDir,
		query,
	)
	
	if err != nil {
		ctrl.logger.Error("Failed to get documents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to retrieve documents",
		}})
		return
	}
	
	totalPages := (int(total) + perPage - 1) / perPage
	
	c.JSON(http.StatusOK, gin.H{
		"data": documents,
		"pagination": gin.H{
			"total":       total,
			"page":        page,
			"per_page":    perPage,
			"total_pages": totalPages,
		},
	})
}

func (ctrl *documentController) GetDocumentByID(c *gin.Context) {
	idStr := c.Param("id")
	documentID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "validation_error",
			"message": "Invalid document ID",
		}})
		return
	}
	
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
			"code":    "unauthorized",
			"message": "User not authenticated",
		}})
		return
	}
	
	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()
	
	document, err := ctrl.service.GetDocumentByID(
		c.Request.Context(),
		documentID,
		userID.(uuid.UUID),
		true, 
		ipAddress,
		userAgent,
	)
	
	if err != nil {
		if err == service.ErrDocumentNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
				"code":    "not_found",
				"message": "Document not found",
			}})
			return
		}
		
		if err == service.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{
				"code":    "forbidden",
				"message": "You don't have permission to access this document",
			}})
			return
		}
		
		ctrl.logger.Error("Failed to get document", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to retrieve document",
		}})
		return
	}
	
	c.JSON(http.StatusOK, document)
}

func (ctrl *documentController) UpdateDocument(c *gin.Context) {
	idStr := c.Param("id")
	documentID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "validation_error",
			"message": "Invalid document ID",
		}})
		return
	}
	
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
			"code":    "unauthorized",
			"message": "User not authenticated",
		}})
		return
	}
	
	var req model.DocumentUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "validation_error",
			"message": "Invalid request data",
			"details": err.Error(),
		}})
		return
	}
	
	document, err := ctrl.service.UpdateDocument(
		c.Request.Context(),
		documentID,
		userID.(uuid.UUID),
		req,
	)
	
	if err != nil {
		if err == service.ErrDocumentNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
				"code":    "not_found",
				"message": "Document not found",
			}})
			return
		}
		
		if err == service.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{
				"code":    "forbidden",
				"message": "You don't have permission to update this document",
			}})
			return
		}
		
		ctrl.logger.Error("Failed to update document", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to update document",
		}})
		return
	}
	
	c.JSON(http.StatusOK, document)
}

func (ctrl *documentController) DeleteDocument(c *gin.Context) {
	idStr := c.Param("id")
	documentID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "validation_error",
			"message": "Invalid document ID",
		}})
		return
	}
	
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
			"code":    "unauthorized",
			"message": "User not authenticated",
		}})
		return
	}
	
	err = ctrl.service.DeleteDocument(
		c.Request.Context(),
		documentID,
		userID.(uuid.UUID),
	)
	
	if err != nil {
		if err == service.ErrDocumentNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
				"code":    "not_found",
				"message": "Document not found",
			}})
			return
		}
		
		if err == service.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{
				"code":    "forbidden",
				"message": "You don't have permission to delete this document",
			}})
			return
		}
		
		ctrl.logger.Error("Failed to delete document", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to delete document",
		}})
		return
	}
	
	c.Status(http.StatusNoContent)
}

func (ctrl *documentController) GetDocumentHistory(c *gin.Context) {
	idStr := c.Param("id")
	documentID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "validation_error",
			"message": "Invalid document ID",
		}})
		return
	}
	
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
			"code":    "unauthorized",
			"message": "User not authenticated",
		}})
		return
	}
	
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	
	history, total, err := ctrl.service.GetDocumentHistory(
		c.Request.Context(),
		documentID,
		userID.(uuid.UUID),
		page,
		perPage,
	)
	
	if err != nil {
		if err == service.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{
				"code":    "forbidden",
				"message": "You don't have permission to access this document",
			}})
			return
		}
		
		ctrl.logger.Error("Failed to get document history", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to retrieve document history",
		}})
		return
	}
	
	totalPages := (int(total) + perPage - 1) / perPage
	
	c.JSON(http.StatusOK, gin.H{
		"data": history,
		"pagination": gin.H{
			"total":       total,
			"page":        page,
			"per_page":    perPage,
			"total_pages": totalPages,
		},
	})
}

func (ctrl *documentController) RestoreDocumentVersion(c *gin.Context) {
	idStr := c.Param("id")
	documentID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "validation_error",
			"message": "Invalid document ID",
		}})
		return
	}
	
	versionStr := c.Param("version")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "validation_error",
			"message": "Invalid version number",
		}})
		return
	}
	
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
			"code":    "unauthorized",
			"message": "User not authenticated",
		}})
		return
	}
	
	document, err := ctrl.service.RestoreDocumentVersion(
		c.Request.Context(),
		documentID,
		userID.(uuid.UUID),
		version,
	)
	
	if err != nil {
		if err == service.ErrDocumentNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
				"code":    "not_found",
				"message": "Document not found",
			}})
			return
		}
		
		if err == service.ErrVersionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
				"code":    "not_found",
				"message": "Document version not found",
			}})
			return
		}
		
		if err == service.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{
				"code":    "forbidden",
				"message": "You don't have permission to restore this document",
			}})
			return
		}
		
		ctrl.logger.Error("Failed to restore document version", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to restore document version",
		}})
		return
	}
	
	c.JSON(http.StatusOK, document)
}

func (ctrl *documentController) ShareDocument(c *gin.Context) {
	idStr := c.Param("id")
	documentID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "validation_error",
			"message": "Invalid document ID",
		}})
		return
	}
	
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
			"code":    "unauthorized",
			"message": "User not authenticated",
		}})
		return
	}
	
	var req model.CollaboratorCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "validation_error",
			"message": "Invalid request data",
			"details": err.Error(),
		}})
		return
	}
	
	collaborator, err := ctrl.service.ShareDocument(
		c.Request.Context(),
		documentID,
		userID.(uuid.UUID),
		req,
	)
	
	if err != nil {
		if err == service.ErrDocumentNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
				"code":    "not_found",
				"message": "Document not found",
			}})
			return
		}
		
		if err == service.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
				"code":    "not_found",
				"message": "User not found",
			}})
			return
		}
		
		if err == service.ErrAlreadyCollaborator {
			c.JSON(http.StatusConflict, gin.H{"error": gin.H{
				"code":    "conflict",
				"message": "User is already a collaborator",
			}})
			return
		}
		
		if err == service.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{
				"code":    "forbidden",
				"message": "You don't have permission to share this document",
			}})
			return
		}
		
		ctrl.logger.Error("Failed to share document", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to share document",
		}})
		return
	}
	
	c.JSON(http.StatusOK, collaborator)
}

func (ctrl *documentController) UpdateCollaboratorPermission(c *gin.Context) {
	idStr := c.Param("id")
	documentID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "validation_error",
			"message": "Invalid document ID",
		}})
		return
	}
	
	userIDStr := c.Param("user_id")
	collaboratorUserID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "validation_error",
			"message": "Invalid user ID",
		}})
		return
	}
	
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
			"code":    "unauthorized",
			"message": "User not authenticated",
		}})
		return
	}
	
	var req model.CollaboratorUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "validation_error",
			"message": "Invalid request data",
			"details": err.Error(),
		}})
		return
	}
	
	collaborator, err := ctrl.service.UpdateCollaboratorPermission(
		c.Request.Context(),
		documentID,
		userID.(uuid.UUID),
		collaboratorUserID,
		req,
	)
	
	if err != nil {
		if err == service.ErrDocumentNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
				"code":    "not_found",
				"message": "Document not found",
			}})
			return
		}
		
		if err == service.ErrNotCollaborator {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
				"code":    "not_found",
				"message": "User is not a collaborator",
			}})
			return
		}
		
		if err == service.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{
				"code":    "forbidden",
				"message": "You don't have permission to update collaborator permissions",
			}})
			return
		}
		
		ctrl.logger.Error("Failed to update collaborator permission", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to update collaborator permission",
		}})
		return
	}
	
	c.JSON(http.StatusOK, collaborator)
}

func (ctrl *documentController) RemoveCollaborator(c *gin.Context) {
	idStr := c.Param("id")
	documentID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "validation_error",
			"message": "Invalid document ID",
		}})
		return
	}
	
	userIDStr := c.Param("user_id")
	collaboratorUserID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "validation_error",
			"message": "Invalid user ID",
		}})
		return
	}
	
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
			"code":    "unauthorized",
			"message": "User not authenticated",
		}})
		return
	}
	
	err = ctrl.service.RemoveCollaborator(
		c.Request.Context(),
		documentID,
		userID.(uuid.UUID),
		collaboratorUserID,
	)
	
	if err != nil {
		if err == service.ErrDocumentNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
				"code":    "not_found",
				"message": "Document not found",
			}})
			return
		}
		
		if err == service.ErrCannotRemoveOwner {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
				"code":    "validation_error",
				"message": "Cannot remove document owner as collaborator",
			}})
			return
		}
		
		if err == service.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{
				"code":    "forbidden",
				"message": "You don't have permission to remove collaborators",
			}})
			return
		}
		
		ctrl.logger.Error("Failed to remove collaborator", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to remove collaborator",
		}})
		return
	}
	
	c.Status(http.StatusNoContent)
}

func (ctrl *documentController) GetDocumentAnalytics(c *gin.Context) {
	idStr := c.Param("id")
	documentID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "validation_error",
			"message": "Invalid document ID",
		}})
		return
	}
	
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
			"code":    "unauthorized",
			"message": "User not authenticated",
		}})
		return
	}
	
	period := c.DefaultQuery("period", "month")
	
	analytics, err := ctrl.service.GetDocumentAnalytics(
		c.Request.Context(),
		documentID,
		userID.(uuid.UUID),
		period,
	)
	
	if err != nil {
		if err == service.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{
				"code":    "forbidden",
				"message": "You don't have permission to access this document",
			}})
			return
		}
		
		ctrl.logger.Error("Failed to get document analytics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to retrieve document analytics",
		}})
		return
	}
	
	c.JSON(http.StatusOK, analytics)
}

func (ctrl *documentController) GetUserAnalytics(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
			"code":    "unauthorized",
			"message": "User not authenticated",
		}})
		return
	}
	
	period := c.DefaultQuery("period", "month")
	
	analytics, err := ctrl.service.GetUserAnalytics(
		c.Request.Context(),
		userID.(uuid.UUID),
		period,
	)
	
	if err != nil {
		ctrl.logger.Error("Failed to get user analytics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to retrieve user analytics",
		}})
		return
	}
	
	c.JSON(http.StatusOK, analytics)
}