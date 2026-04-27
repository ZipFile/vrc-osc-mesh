package room

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type UserID string
type RoomID uuid.UUID
type InviteID uuid.UUID
type InviteDirection int

const (
	FromUser InviteDirection = iota
	ToUser
)

var ErrNotFound = errors.New("not found")
var ErrInviteNotFound = fmt.Errorf("invite not found: %w", ErrNotFound)
var ErrRequestNotFound = fmt.Errorf("request not found: %w", ErrNotFound)
var ErrAlreadyMember = errors.New("user is already a member of the room")
var ErrAlreadyRequested = errors.New("user already requested to join the room")
var ErrAlreadyInvited = errors.New("user already invited to the room")

type Member struct {
	UserID UserID `json:"user_id"`
	Name   string `json:"name"`
}

type Invite struct {
	ID        InviteID        `json:"id"`
	Name      string          `json:"name"`
	UserID    UserID          `json:"user_id"`
	Direction InviteDirection `json:"direction"`
	CreatedAt time.Time       `json:"created_at"`
}

type Room struct {
	ID           RoomID    `json:"id"`
	MasterID     UserID    `json:"master_id"`
	Name         string    `json:"name"`
	LastActivity time.Time `json:"last_activity"`
	CreatedAt    time.Time `json:"created_at"`
	Members      []Member  `json:"members"`
	Invites      []Invite  `json:"invites"`
}

type Repository interface {
	Add(Room) error
	Get(RoomID) (*Room, error)
	Delete(RoomID) error
	ListExpired(ttl time.Duration) ([]RoomID, error)
	ListForUser(id UserID) ([]Room, error)
}
