package room

import (
	"time"

	"github.com/google/uuid"
)

type UserID string
type RoomID uuid.UUID
type JoinRequestID uuid.UUID
type InviteID uuid.UUID

type Member struct {
	UserID UserID `json:"user_id"`
	Name   string `json:"name"`
}

type JoinRequest struct {
	ID        JoinRequestID `json:"id"`
	To        UserID        `json:"to"`
	From      UserID        `json:"from"`
	CreatedAt time.Time     `json:"created_at"`
}

type Invite struct {
	ID        InviteID  `json:"id"`
	To        UserID    `json:"to"`
	From      UserID    `json:"from"`
	CreatedAt time.Time `json:"created_at"`
}

type Room struct {
	ID           RoomID        `json:"id"`
	MasterID     UserID        `json:"master_id"`
	Name         string        `json:"name"`
	LastActivity time.Time     `json:"last_activity"`
	CreatedAt    time.Time     `json:"created_at"`
	Members      []Member      `json:"members"`
	Invites      []Invite      `json:"invites"`
	Requests     []JoinRequest `json:"requests"`
}

func (r *Room) Copy() *Room {
	if r == nil {
		return nil
	}

	members := make([]Member, len(r.Members))
	invites := make([]Invite, len(r.Invites))
	requsts := make([]JoinRequest, len(r.Requests))

	copy(members, r.Members)
	copy(invites, r.Invites)
	copy(requsts, r.Requests)

	c := *r
	c.Invites = invites
	c.Members = members
	c.Requests = requsts

	return &c
}

type Repository interface {
	Add(Room) error
	Get(RoomID) (*Room, error)
	Delete(RoomID) error
	ListExpired(ttl time.Duration) ([]RoomID, error)
	ListForUser(id UserID) ([]Room, error)
}
