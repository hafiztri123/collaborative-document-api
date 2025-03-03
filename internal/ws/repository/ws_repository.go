package repository

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hafiztri123/document-api/internal/ws/model"
	"go.uber.org/zap"
)

type Client struct {
	ID string
	UserID uuid.UUID
	Name string
	Conn *websocket.Conn
	Send chan []byte
}

type Repository interface {
	// Client management
	RegisterClient(client *Client)
	UnregisterClient(client *Client)
	GetClients() []*Client
	
	// Document subscriptions
	Subscribe(documentID uuid.UUID, clientID string)
	Unsubscribe(documentID uuid.UUID, clientID string)
	GetSubscribers(documentID uuid.UUID) []*Client
	
	// Broadcasting
	BroadcastToDocument(documentID uuid.UUID, message []byte, excludeClientID string)
	BroadcastCursorPosition(documentID uuid.UUID, message model.CursorMessage)
}

type wsRepository struct {
	clients map[string]*Client
	subscribers map[uuid.UUID]map[string]bool
	mutex sync.RWMutex
	logger *zap.Logger
}


func NewWSRepository(logger *zap.Logger) Repository {
	return &wsRepository{
		clients: make(map[string]*Client),
		subscribers: make(map[uuid.UUID]map[string]bool),
		logger: logger,
	}
}





func (r *wsRepository)	RegisterClient(client *Client) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.clients[client.ID] = client
	r.logger.Debug("Registered Websocket client",
		zap.String("clientID", client.ID),
		zap.String("userID", client.UserID.String()))
}


func (r *wsRepository)	UnregisterClient(client *Client){
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for documentID, subscribers := range r.subscribers {
		if _, ok := subscribers[client.ID]; ok {
			delete(subscribers, client.ID)
			r.logger.Debug("Unsubscriber client from document",
				zap.String("clientID", client.ID),
				zap.String("documentID", documentID.String()))
		}

		if len(subscribers) == 0 {
			delete(r.subscribers, documentID)
		}
	}

	if _, ok := r.clients[client.ID]; ok {
		delete(r.clients, client.ID)
		close(client.Send)
		r.logger.Debug("Unregistered Websocket client",
			zap.String("clientID", client.ID))
	}
}


func (r *wsRepository)	GetClients() []*Client{
	r.mutex.RLock()
	defer r.mutex.RLock()

	clients := make([]*Client, 0, len(r.clients))
	for _, client := range r.clients {
		clients = append(clients, client)
	}

	return clients
}


func (r *wsRepository)	Subscribe(documentID uuid.UUID, clientID string){
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _,ok := r.subscribers[documentID]; !ok {
		r.subscribers[documentID] = make(map[string]bool)
	}

	r.subscribers[documentID][clientID] = true
	r.logger.Debug("Client subscribed to document",
		zap.String("clientID", clientID),
		zap.String("documentID", documentID.String()))
}


func (r *wsRepository)	Unsubscribe(documentID uuid.UUID, clientID string){
	r.mutex.Lock()
	defer r.mutex.Unlock()


	if subscribers, ok := r.subscribers[documentID]; ok {
		delete(subscribers, clientID)
		r.logger.Debug("Client unsubscribed from document",
			zap.String("clientID", clientID),
			zap.String("documentID", documentID.String()))
		
		if len(subscribers) == 0 {
			delete(r.subscribers, documentID)
		}
	}

}


func (r *wsRepository)	GetSubscribers(documentID uuid.UUID) []*Client{
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var subscribers []*Client

	if subscriptionMap, ok := r.subscribers[documentID]; ok {
		for clientID := range subscriptionMap {
			if client, ok := r.clients[clientID]; ok {
				subscribers = append(subscribers, client)
			}
		}
	}

	return subscribers

}


func (r *wsRepository)	BroadcastToDocument(documentID uuid.UUID, message []byte, excludeClientID string){
	subscribers := r.GetSubscribers(documentID)

	for _, client := range subscribers {
		if client.ID == excludeClientID {
			continue
		}

		select {
		case client.Send <- message:
			r.logger.Debug("Broadcast to document",
				zap.String("clientID", client.ID),
				zap.String("documentID", documentID.String()))
		default:
			r.logger.Warn("Client send buffer full, closing connection",
				zap.String("clientID", client.ID))
			r.UnregisterClient(client)
		}
	}
}


// BroadcastCursorPosition sends a cursor position to all clients subscribed to a document
func (r *wsRepository) BroadcastCursorPosition(documentID uuid.UUID, message model.CursorMessage) {
	subscribers := r.GetSubscribers(documentID)

	for _, client := range subscribers {
		if client.UserID == message.User.ID {
			continue
		}

		messageBytes, err := json.Marshal(message)
		if err != nil {
			r.logger.Error("Failed to marshal cursor message", 
				zap.Error(err),
				zap.String("clientID", client.ID),
				zap.String("documentID", documentID.String()))
			continue
		}

		select {
		case client.Send <- messageBytes:
			r.logger.Debug("Cursor position broadcasted to client",
				zap.String("clientID", client.ID),
				zap.String("documentID", documentID.String()))
		default:
			// Client send buffer is full, unregister the client
			r.logger.Warn("Client send buffer full, closing connection", 
				zap.String("clientID", client.ID))
			r.UnregisterClient(client)
		}
	}
}


