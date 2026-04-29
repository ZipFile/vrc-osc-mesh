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
		ID:           RoomID(uuid.New().String()),
		MasterID:     UserID("test"),
		Name:         "test",
		LastActivity: now,
		CreatedAt:    now,
		Members:      []User{},
		Invites:      []Invite{},
	}
	invite := Invite{
		ID:        InviteID(uuid.New().String()),
		UserID:    "test",
		Name:      "test",
		Direction: FromUser,
		CreatedAt: now,
	}
	member := User{ID: "test", Name: "test"}

	joined, err := room.AddInvite(invite, now)

	require.NoError(t, err)
	require.False(t, joined)

	err = room.AddMember(member, now)

	require.NoError(t, err)
	require.Equal(t, []User{member}, room.Members)
	require.True(t, room.IsMember(member.ID))
	require.False(t, room.IsInvited(member.ID))

	t.Run("existing", func(t *testing.T) {
		err := room.AddMember(member, now)

		require.ErrorIs(t, err, ErrAlreadyMember)
	})

	t.Run("remove master", func(t *testing.T) {
		ok := room.RemoveMember(member.ID, now)

		require.False(t, ok)
		require.True(t, room.IsMember(member.ID))
	})

	t.Run("remove", func(t *testing.T) {
		err = room.AddMember(User{ID: "extra", Name: "extra"}, now)

		require.NoError(t, err)

		ok := room.RemoveMember("extra", now)

		require.True(t, ok)
		require.False(t, room.IsMember("extra"))
	})
}

func TestRoom_Copy(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var room *Room
		require.Nil(t, room.Copy())
	})

	now := time.Now()
	room := &Room{
		ID:           RoomID(uuid.New().String()),
		MasterID:     UserID("test"),
		Name:         "test",
		LastActivity: now,
		CreatedAt:    now,
		Members:      []User{{Name: "test", ID: "test"}},
		Invites: []Invite{
			{
				ID:        InviteID(uuid.New().String()),
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

func TestRoom_ChangeMaster(t *testing.T) {
	var r *Room

	t.Run("nil", func(t *testing.T) {
		var r *Room
		require.False(t, r.ChangeMaster(UserID("test"), time.Now()))
	})

	now := time.Now()
	r = &Room{
		ID:           RoomID(uuid.New().String()),
		MasterID:     "test",
		Name:         "test",
		LastActivity: now,
		CreatedAt:    now,
		Members: []User{
			{Name: "test", ID: "test"},
			{Name: "xxx", ID: "xxx"},
		},
	}

	t.Run("master", func(t *testing.T) {
		require.False(t, r.ChangeMaster("test", now))
	})

	t.Run("member", func(t *testing.T) {
		require.True(t, r.ChangeMaster("xxx", now))
	})

	t.Run("non member", func(t *testing.T) {
		require.False(t, r.ChangeMaster("yyy", now))
	})
}
