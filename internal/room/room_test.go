package room

import (
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/require"
	"github.com/google/uuid"
)

func TestRoom_MemberManagement(t *testing.T) {
	now := time.Now()
	room := &Room{
		ID:           RoomID(uuid.New()),
		MasterID:     UserID("test"),
		Name:         "test",
		LastActivity: now,
		CreatedAt:    now,
		Members:      []Member{},
		Invites:      []Invite{},
	}
	invite := Invite{
		ID:        InviteID(uuid.New()),
		UserID:    "test",
		Name:      "test",
		Direction: FromUser,
		CreatedAt: now,
	}
	member := Member{UserID: "test", Name: "test"}

	err := room.AddInvite(invite, now)

	require.NoError(t, err)

	err = room.AddMember(member, now)

	require.NoError(t, err)
	require.Equal(t, []Member{member}, room.Members)
	require.True(t, room.IsMember(member.UserID))
	require.False(t, room.IsInvited(member.UserID))

	t.Run("existing", func(t *testing.T) {
		err := room.AddMember(member, now)

		require.ErrorIs(t, err, ErrAlreadyMember)
	})

	t.Run("remove", func(t *testing.T) {
		room.RemoveMember(member.UserID, now)

		require.NoError(t, err)
		require.False(t, room.IsMember(member.UserID))
	})
}

func TestRoom_Copy(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var room *Room
		require.Nil(t, room.Copy())
	})

	now := time.Now()
	room := &Room{
		ID:           RoomID(uuid.New()),
		MasterID:     UserID("test"),
		Name:         "test",
		LastActivity: now,
		CreatedAt:    now,
		Members:      []Member{{Name: "test", UserID: "test"}},
		Invites: []Invite{
			{
				ID:        InviteID(uuid.New()),
				UserID:    UserID(uuid.New().String()),
				Name:      "test",
				Direction: ToUser,
				CreatedAt: now,
			},
		},
	}
	roomCopy := room.Copy()

	require.Equal(t, room, roomCopy)

	room.Members[0].Name = "modified"
	room.Invites[0].UserID = "modified"

	require.NotEqual(t, room, roomCopy)
}
