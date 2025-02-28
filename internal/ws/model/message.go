package model

import (
	"time"

	"github.com/google/uuid"
)

type MessageType string

const (
	MessageTypeSubscribe MessageType = "subscribe"
	MessageTypeUpdate MessageType = "update"
	MessageTypeCursor MessageType = "cursor"
	MessageTypeError MessageType = "error"
	MessageTypePing MessageType = "ping"
	MessageTypePong MessageType = "pong"
)

type BaseMessage struct {
	Type MessageType `json:"type"`
}

type SubscribeMessage struct {
	BaseMessage
	DocumentID uuid.UUID `json:"document_id"`
}

type JSONPatchOperation struct {
	Op    string      `json:"op"`    // add, remove, replace, etc.
	Path  string      `json:"path"`  // JSON pointer path
	Value interface{} `json:"value,omitempty"` // Optional for some operations
}

type UpdateMessage struct {
	BaseMessage
	DocumentID uuid.UUID           `json:"document_id"`
	Version    int                 `json:"version"`
	Patches    []JSONPatchOperation `json:"patches"`
	User       struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	} `json:"user"`
	Timestamp time.Time `json:"timestamp"`
}

type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type CursorMessage struct {
	BaseMessage
	DocumentID uuid.UUID `json:"document_id"`
	Position   Position  `json:"position"`
	User       struct {
		ID    uuid.UUID `json:"id"`
		Name  string    `json:"name"`
		Color string    `json:"color"` 
	} `json:"user"`
}

type ErrorMessage struct {
	BaseMessage
	Code    string `json:"code"`
	Message string `json:"message"`
}

type PingMessage struct {
	BaseMessage
}

type PongMessage struct {
	BaseMessage
}