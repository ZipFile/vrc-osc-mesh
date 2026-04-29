package room

import (
	"context"
	"errors"
	"time"
)

const (
	userCtxKey = "vrc_mesh_user"
)

var ErrNoUserInContext = errors.New("user not found in context")

func UserFromContext(ctx context.Context) (*User, error) {
	if v := ctx.Value(userCtxKey); v != nil {
		if user, ok := v.(*User); ok {
			return user, nil
		}
	}
	return nil, ErrNoUserInContext
}

func WithUserInContext(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userCtxKey, user)
}

func (u *User) NewRoom(id RoomID, name string, now time.Time) *Room {
	if u == nil {
		return nil
	}

	return &Room{
		ID:           id,
		MasterID:     u.ID,
		Name:         name,
		LastActivity: now,
		CreatedAt:    now,
		Members:      []User{*u},
		Invites:      []Invite{},
	}
}

func (u *User) newInvite(id InviteID, now time.Time, dir InviteDirection) *Invite {
	if u == nil {
		return nil
	}

	return &Invite{
		ID:        id,
		UserID:    u.ID,
		Name:      u.Name,
		Direction: dir,
		CreatedAt: now,
	}
}

func (u *User) NewInvite(id InviteID, now time.Time) *Invite {
	return u.newInvite(id, now, ToUser)
}

func (u *User) NewJoinRequest(id InviteID, now time.Time) *Invite {
	return u.newInvite(id, now, FromUser)
}
