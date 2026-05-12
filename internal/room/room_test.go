package room

import (
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/require"
	"github.com/google/uuid"
)

func TestRoom_MemberManagement(t *testing.T) {
	now := time.Now()
	member := User{ID: "test", Name: "test"}

	t.Run("nil", func(t *testing.T) {
		var r *Room
		err := r.AddMember(member, now)

		require.ErrorIs(t, err, ErrRoomNotFound)
		require.False(t, r.IsMember(member.ID))

		err = r.RemoveMember(member.ID, now)

		require.ErrorIs(t, err, ErrRoomNotFound)
	})

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

	joined, err := room.AddInvite(invite, now)

	require.NoError(t, err)
	require.False(t, joined)

	err = room.AddMember(member, now)

	require.NoError(t, err)
	require.Equal(t, []User{member}, room.Members)
	require.True(t, room.IsMember(member.ID))
	require.False(t, room.IsInvited(member.ID))

	t.Run("existing", func(t *testing.T) {
		err = room.AddMember(member, now)

		require.ErrorIs(t, err, ErrAlreadyMember)
	})

	t.Run("remove master", func(t *testing.T) {
		err = room.RemoveMember(member.ID, now)

		require.ErrorIs(t, err, ErrCannotRemoveMaster)
		require.True(t, room.IsMember(member.ID))
	})

	t.Run("remove unknown", func(t *testing.T) {
		err = room.RemoveMember("unknown", now)

		require.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("remove", func(t *testing.T) {
		err = room.AddMember(User{ID: "extra", Name: "extra"}, now)

		require.NoError(t, err)

		err = room.RemoveMember("extra", now)

		require.Nil(t, err)
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
	t.Run("nil", func(t *testing.T) {
		var r *Room
		err := r.ChangeMaster("test", time.Now())

		require.ErrorIs(t, err, ErrRoomNotFound)
	})

	now := time.Now()
	r := &Room{
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
		err := r.ChangeMaster("test", now)

		require.ErrorIs(t, err, ErrAlreadyMaster)
	})

	t.Run("member", func(t *testing.T) {
		err := r.ChangeMaster("xxx", now)

		require.NoError(t, err)
	})

	t.Run("non member", func(t *testing.T) {
		err := r.ChangeMaster("yyy", now)

		require.ErrorIs(t, err, ErrUserNotFound)
	})
}
