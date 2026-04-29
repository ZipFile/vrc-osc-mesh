package room

import (
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/require"
	"github.com/google/uuid"
)

func TestUserContext(t *testing.T) {
	user := &User{ID: UserID("test"), Name: "test"}
	ctx := WithUserInContext(t.Context(), user)

	t.Run("err", func(t *testing.T) {
		ctxUser, err := UserFromContext(t.Context())

		require.Nil(t, ctxUser)
		require.ErrorIs(t, err, ErrNoUserInContext)
	})

	t.Run("ok", func(t *testing.T) {
		ctxUser, err := UserFromContext(ctx)

		require.NoError(t, err)
		require.Equal(t, user, ctxUser)
	})
}

func TestUser_NewRoom(t *testing.T) {
	now := time.Now()
	var user *User
	var room *Room

	room = user.NewRoom("test", "test", now)

	require.Nil(t, room)

	user = &User{ID: UserID("test_user"), Name: "Test User"}
	expected := Room{
		ID:           RoomID("test_room"),
		MasterID:     UserID("test_user"),
		Name:         "Test Room",
		Members:      []User{*user},
		Invites:      []Invite{},
		LastActivity: now,
		CreatedAt:    now,
	}
	room = user.NewRoom("test_room", "Test Room", now)

	require.NotNil(t, room)
	require.Equal(t, expected, *room)
}

func TestUser_newInvite(t *testing.T) {
	u := User{ID: "test", Name: "test"}
	now := time.Now()
	id := InviteID(uuid.New().String())

	t.Run("nil", func(t *testing.T) {
		var nu *User
		require.Nil(t, nu.NewInvite(id, now))
	})

	t.Run("invite", func(t *testing.T) {
		invite := u.NewInvite(id, now)
		expected := Invite{
			ID:        id,
			Name:      u.Name,
			UserID:    u.ID,
			Direction: ToUser,
			CreatedAt: now,
		}

		require.Equal(t, expected, *invite)
	})

	t.Run("request", func(t *testing.T) {
		invite := u.NewJoinRequest(id, now)
		expected := Invite{
			ID:        id,
			Name:      u.Name,
			UserID:    u.ID,
			Direction: FromUser,
			CreatedAt: now,
		}

		require.Equal(t, expected, *invite)
	})
}
