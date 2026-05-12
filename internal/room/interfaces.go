package room

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type UserID string
type RoomID string
type InviteID string
type InviteDirection int

const (
	FromUser InviteDirection = iota
	ToUser
)

var ErrNotFound = errors.New("not found")
var ErrRoomNotFound = fmt.Errorf("%w: room", ErrNotFound)
var ErrInviteNotFound = fmt.Errorf("%w: invite", ErrNotFound)
var ErrUserNotFound = fmt.Errorf("%w: user", ErrNotFound)
var ErrAlreadyMember = errors.New("user is already a member of the room")
var ErrAlreadyInvited = errors.New("user already invited to the room")
var ErrAlreadyMaster = errors.New("user is already the master of the room")
var ErrCannotRemoveMaster = errors.New("cannot remove master user from room")
var ErrNotMaster = errors.New("user is not the master of the room")
var ErrInviteNotAcceptable = errors.New("invite is not acceptable")

type User struct {
	ID   UserID `json:"id"`
	Name string `json:"name"`
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
	Members      []User    `json:"members"`
	Invites      []Invite  `json:"invites"`
}

type Repository interface {
	Add(context.Context, Room) error
	Get(context.Context, RoomID) (*Room, error)
	Delete(context.Context, RoomID) error
	ListExpired(ctx context.Context, ttl time.Duration) ([]RoomID, error)
	ListForUser(ctx context.Context, id UserID) ([]Room, error)
	Lock(context.Context, RoomID, func(context.Context, *Room) error) error
}

type Service interface {
	CreateRoom(context.Context, string) (*Room, error)
	DestroyRoom(context.Context, RoomID) error
	RequestRoomJoin(context.Context, RoomID) (*Room, bool, error)
	SendInvite(context.Context, RoomID, User) (*Room, bool, error)
	AcceptInvite(context.Context, RoomID, InviteID) (*Room, error)
	RejectInvite(context.Context, RoomID, InviteID) (*Room, error)
	RemoveUser(context.Context, RoomID, UserID) (*Room, error)
	ChangeMaster(context.Context, RoomID, UserID) (*Room, error)
}
