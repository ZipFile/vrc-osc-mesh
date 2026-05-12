package room

import (
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/require"
	"github.com/google/uuid"
)

func TestInvite_IsAcceptable(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var invite *Invite
		require.False(t, invite.IsAcceptable("", ""))
		require.False(t, invite.IsRejectable("", ""))
	})

	master := User{ID: "master", Name: "master"}
	user := User{ID: "user", Name: "user"}
	invite := Invite{
		ID:        InviteID(uuid.New().String()),
		UserID:    user.ID,
		Name:      "test",
		Direction: FromUser,
		CreatedAt: time.Now(),
	}

	require.True(t, invite.IsAcceptable(master.ID, master.ID))
	require.False(t, invite.IsAcceptable(user.ID, master.ID))
	require.False(t, invite.IsAcceptable("random", master.ID))

	require.True(t, invite.IsRejectable(master.ID, master.ID))
	require.True(t, invite.IsRejectable(user.ID, master.ID))
	require.False(t, invite.IsRejectable("random", master.ID))

	invite.Direction = ToUser

	require.False(t, invite.IsAcceptable(master.ID, master.ID))
	require.True(t, invite.IsAcceptable(user.ID, master.ID))
	require.False(t, invite.IsAcceptable("random", master.ID))
}

func TestRoom_GetInvite(t *testing.T) {
	invite := Invite{
		ID:        InviteID(uuid.New().String()),
		UserID:    UserID(uuid.New().String()),
		Name:      "test",
		Direction: FromUser,
		CreatedAt: time.Now(),
	}
	room := Room{
		Invites: []Invite{invite},
	}

	require.Equal(t, &invite, room.GetInvite(invite.ID))
	require.Nil(t, room.GetInvite(InviteID(uuid.New().String())))
}

func TestInvite_Member(t *testing.T) {
	invite := Invite{
		ID:        InviteID(uuid.New().String()),
		UserID:    UserID(uuid.New().String()),
		Name:      "test",
		Direction: FromUser,
		CreatedAt: time.Now(),
	}
	expected := User{
		ID:   invite.UserID,
		Name: invite.Name,
	}

	require.Equal(t, expected, invite.Member())
}

func TestRoom_Invites(t *testing.T) {
	now := time.Now()
	invite := Invite{
		ID:        InviteID(uuid.New().String()),
		UserID:    UserID(uuid.New().String()),
		Name:      "test",
		Direction: ToUser,
		CreatedAt: now,
	}
	room := Room{}

	require.False(t, room.IsInvited(invite.UserID))

	joined, err := room.AddInvite(invite, now)

	require.NoError(t, err)
	require.False(t, joined)
	require.True(t, room.IsInvited(invite.UserID))

	joined, err = room.AddInvite(invite, now)

	require.ErrorIs(t, err, ErrAlreadyInvited)
	require.False(t, joined)

	t.Run("accept", func(t *testing.T) {
		roomCopy := room.Copy()
		err = roomCopy.AcceptInvite(invite.ID, now)

		require.NoError(t, err)
		require.False(t, roomCopy.IsInvited(invite.UserID))
		require.True(t, roomCopy.IsMember(invite.UserID))

		joined, err = roomCopy.AddInvite(invite, now)

		require.ErrorIs(t, err, ErrAlreadyMember)
		require.False(t, joined)

		err = roomCopy.AcceptInvite(invite.ID, now)

		require.ErrorIs(t, err, ErrInviteNotFound)
	})

	t.Run("remove", func(t *testing.T) {
		roomCopy := room.Copy()
		err := roomCopy.RemoveInvite(invite.ID, now)

		require.NoError(t, err)
		require.False(t, roomCopy.IsInvited(invite.UserID))
		require.False(t, roomCopy.IsMember(invite.UserID))

		err = roomCopy.RemoveInvite(invite.ID, now)

		require.ErrorIs(t, err, ErrInviteNotFound)
	})
}

func TestRoom_Invite_PendingJoinRequest(t *testing.T) {
	now := time.Now()
	userID := UserID(uuid.New().String())
	invite := Invite{
		ID:        InviteID(uuid.New().String()),
		UserID:    userID,
		Direction: FromUser,
		Name:      "test",
		CreatedAt: now,
	}
	request := Invite{
		ID:        InviteID(uuid.New().String()),
		UserID:    userID,
		Direction: ToUser,
		Name:      "tset",
		CreatedAt: now,
	}
	room := Room{}
	joined, err := room.AddInvite(request, now)

	require.NoError(t, err)
	require.False(t, joined)

	joined, err = room.AddInvite(invite, now)

	require.NoError(t, err)
	require.True(t, joined)
	require.False(t, room.IsRequested(userID))
	require.False(t, room.IsInvited(userID))
	require.True(t, room.IsMember(userID))
}

func TestInvite_IsJoinRequest(t *testing.T) {
	invite := Invite{
		ID:        InviteID(uuid.New().String()),
		UserID:    UserID(uuid.New().String()),
		Name:      "test",
		Direction: FromUser,
		CreatedAt: time.Now(),
	}

	require.True(t, invite.IsJoinRequest())
}
