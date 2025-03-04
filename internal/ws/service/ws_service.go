package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hafiztri123/document-api/internal/document/model"
	docRepo "github.com/hafiztri123/document-api/internal/document/repository"
	wsModel "github.com/hafiztri123/document-api/internal/ws/model"
	wsRepo "github.com/hafiztri123/document-api/internal/ws/repository"
	"go.uber.org/zap"
)


var (
	ErrInvalidMessageType = errors.New("invalid message type")
	ErrUnauthorized       = errors.New("unauthorized access to document")
)


type Service interface {
	// Client operations
	HandleConnection(conn *websocket.Conn, userID uuid.UUID, userName string)
	
	// Message handling
	ProcessMessage(ctx context.Context, clientID string, userID uuid.UUID, messageType string, data []byte) error
	
	// Document update broadcasting
	BroadcastDocumentUpdate(ctx context.Context, documentID uuid.UUID, userID uuid.UUID, userName string, version int, patches []wsModel.JSONPatchOperation) error
}

type wsService struct {
	wsRepo wsRepo.Repository
	docRepo docRepo.Repository
	logger *zap.Logger
}

func NewWSService(wsRepo wsRepo.Repository, docRepo docRepo.Repository, logger *zap.Logger) Service {
	return &wsService{
		wsRepo: wsRepo,
		docRepo: docRepo,
		logger: logger,
	}
}


func (s *wsService)	HandleConnection(conn *websocket.Conn, userID uuid.UUID, userName string){
	clientID := uuid.New().String()

	client := &wsRepo.Client{
		ID: clientID,
		UserID: userID,
		Name: userName,
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	s.wsRepo.RegisterClient(client)
	s.logger.Info("Websocket client connected",
		zap.String("clientID", clientID),
		zap.String("userID", userID.String()),
		zap.String("userName", userName))
	
	go s.readPump(client)
	go s.writePump(client)

}

func (s *wsService) readPump(client *wsRepo.Client) {
	defer func() {
		s.wsRepo.UnregisterClient(client)
		client.Conn.Close()
		s.logger.Info("WebSocket client disconnected", 
			zap.String("clientID", client.ID))
	}()
	
	client.Conn.SetReadLimit(4096) // Max message size
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	
	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Error("WebSocket error", zap.Error(err))
			}
			break
		}
		
		var baseMsg wsModel.BaseMessage
		if err := json.Unmarshal(message, &baseMsg); err != nil {
			s.logger.Error("Failed to parse WebSocket message", zap.Error(err))
			continue
		}
		
		if err := s.ProcessMessage(context.Background(), client.ID, client.UserID, string(baseMsg.Type), message); err != nil {
			s.logger.Error("Failed to process WebSocket message", 
				zap.Error(err),
				zap.String("messageType", string(baseMsg.Type)))
			
			errorMsg := wsModel.ErrorMessage{
				BaseMessage: wsModel.BaseMessage{Type: wsModel.MessageTypeError},
				Code:        "error",
				Message:     err.Error(),
			}
			
			if errorBytes, err := json.Marshal(errorMsg); err == nil {
				client.Send <- errorBytes
			}
		}
	}
}

func (s *wsService) writePump(client *wsRepo.Client) {
	ticker := time.NewTicker(45 *time.Second)
	defer func ()  {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <- client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				s.logger.Error("Failed to write websocket message", zap.Error(err))
				return
			}
		
		case <- ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				s.logger.Error("Failed to write ping message", zap.Error(err))
				return
			}
		}
	}
}


func (s *wsService)	ProcessMessage(ctx context.Context, clientID string, userID uuid.UUID, messageType string, data []byte) error{
	switch messageType {
	case string(wsModel.MessageTypeSubscribe):
		return s.handleSubscribe(ctx, clientID, userID, data)
	case string(wsModel.MessageTypeCursor):
		return s.handleCursor(ctx, clientID, userID, data)
	case string(wsModel.MessageTypePing):
		return s.handlePing(ctx, clientID, data)
	default:
		return ErrInvalidMessageType
	}
}

func (s *wsService) handleSubscribe(ctx context.Context, clientID string, userID uuid.UUID, data []byte) error {
	var message wsModel.SubscribeMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return err
	}

	canAccess, err := s.docRepo.CanUserAccess(ctx, message.DocumentID, userID, model.PermissionRead)
	if err != nil {
		return err
	}
	if !canAccess {
		return ErrUnauthorized
	}

	s.wsRepo.Subscribe(message.DocumentID, clientID)
	s.logger.Info("Client subscribed to document",
		zap.String("clientID", clientID),
		zap.String("documentID", message.DocumentID.String()))
	
	return nil
}

func (s *wsService) handleCursor(ctx context.Context, clientID string, userID uuid.UUID, data []byte) error {
	var message wsModel.CursorMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return err
	}

	canAccess, err := s.docRepo.CanUserAccess(ctx, message.DocumentID, userID,  model.PermissionRead)
	if err != nil {
		return err
	}

	if !canAccess {
		return ErrUnauthorized
	}

	s.wsRepo.BroadcastCursorPosition(message.DocumentID, message)

	return nil
}

func (s *wsService) handlePing(ctx context.Context, clientID string, data []byte) error {
	pong := wsModel.PongMessage{
		BaseMessage: wsModel.BaseMessage{
			Type: wsModel.MessageTypePong,
		},
	}

	response, err := json.Marshal(pong)
	if err != nil {
		return err
	}

	clients := s.wsRepo.GetClients()
	for _, client := range clients {
		if client.ID == clientID {
			client.Send <- response
			break
		}
	}

	return nil
}


func (s *wsService)	BroadcastDocumentUpdate(ctx context.Context, documentID uuid.UUID, userID uuid.UUID, userName string, version int, patches []wsModel.JSONPatchOperation) error{
	message := wsModel.UpdateMessage{
		BaseMessage: wsModel.BaseMessage{
			Type: wsModel.MessageTypeUpdate,
		},
		DocumentID: documentID,
		Version: version,
		Patches: patches,
		User: struct {
			ID uuid.UUID `json:"id"`
			Name string `json:"name"`
		}{
			ID: userID,
			Name: userName,
		},
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	var excludeClientID string
	clients := s.wsRepo.GetClients()
	for _, client := range clients {
		if client.UserID == userID {
			excludeClientID = client.ID
			break
		}
	}

	s.wsRepo.BroadcastToDocument(documentID, data, excludeClientID)
	
	return nil

}


