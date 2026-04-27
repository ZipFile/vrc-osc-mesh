package room

import (
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/require"
	"github.com/google/uuid"
)

func TestInvite_Member(t *testing.T) {
	invite := Invite{
		ID:        InviteID(uuid.New()),
		UserID:    UserID(uuid.New().String()),
		Name:      "test",
		Direction: FromUser,
		CreatedAt: time.Now(),
	}
	expected := Member{
		UserID: invite.UserID,
		Name:   invite.Name,
	}

	require.Equal(t, expected, invite.Member())
}

func TestRoom_Invites(t *testing.T) {
	now := time.Now()
	invite := Invite{
		ID:        InviteID(uuid.New()),
		UserID:    UserID(uuid.New().String()),
		Name:      "test",
		Direction: ToUser,
		CreatedAt: now,
	}
	room := Room{}

	require.False(t, room.IsInvited(invite.UserID))

	err := room.AddInvite(invite, now)

	require.NoError(t, err)
	require.True(t, room.IsInvited(invite.UserID))

	err = room.AddInvite(invite, now)

	require.ErrorIs(t, err, ErrAlreadyInvited)

	t.Run("accept", func(t *testing.T) {
		roomCopy := room.Copy()
		err := roomCopy.AcceptInvite(invite.ID, now)

		require.NoError(t, err)
		require.False(t, roomCopy.IsInvited(invite.UserID))
		require.True(t, roomCopy.IsMember(invite.UserID))

		err = roomCopy.AddInvite(invite, now)

		require.ErrorIs(t, err, ErrAlreadyMember)

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
		ID:        InviteID(uuid.New()),
		UserID:    userID,
		Direction: FromUser,
		Name:      "test",
		CreatedAt: now,
	}
	request := Invite{
		ID:        InviteID(uuid.New()),
		UserID:    userID,
		Direction: ToUser,
		Name:      "tset",
		CreatedAt: now,
	}
	room := Room{}
	err := room.AddInvite(request, now)

	require.NoError(t, err)

	err = room.AddInvite(invite, now)

	require.NoError(t, err)
	require.False(t, room.IsRequested(userID))
	require.False(t, room.IsInvited(userID))
	require.True(t, room.IsMember(userID))
}

func TestInvite_IsJoinRequest(t *testing.T) {
	invite := Invite{
		ID:        InviteID(uuid.New()),
		UserID:    UserID(uuid.New().String()),
		Name:      "test",
		Direction: FromUser,
		CreatedAt: time.Now(),
	}

	require.True(t, invite.IsJoinRequest())
}
