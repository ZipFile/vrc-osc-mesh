package room_service

import (
	"errors"
	"testing"
	"time"

	"github.com/ZipFile/vrc-osc-mesh/internal/room"
	"github.com/go-openapi/testify/v2/require"
	"github.com/google/uuid"

	memory_room_repository "github.com/ZipFile/vrc-osc-mesh/internal/room/memory_repository"
)

func TestService_OK(t *testing.T) {
	ctx := t.Context()
	uuids := make([]uuid.UUID, 0, 5)
	now := time.Now()
	uuidFactory := func() (uuid.UUID, error) {
		u := uuid.New()
		uuids = append(uuids, u)
		return u, nil
	}
	nowFunc := func() time.Time { return now }
	repo := memory_room_repository.New(memory_room_repository.WithNowFunc(nowFunc))
	svc := New(repo, WithNowFunc(nowFunc), WithUUIDFactory(uuidFactory))
	master := room.User{ID: "master", Name: "Master"}
	requireRoom := func(t *testing.T, r *room.Room, e *room.Room, err error) {
		require.NoError(t, err, "error performing action")
		require.NotNil(t, r, "action returned nil room")
		require.Equal(t, e, r, "action result mismatch")

		r, err = repo.Get(ctx, r.ID)

		require.NoError(t, err, "error retrieving room")
		require.NotNil(t, r, "retrieved room is nil")
		require.Equal(t, e, r, "retrieved room mismatch")
	}
	lastID := func(t *testing.T) string {
		if len(uuids) == 0 {
			t.Fatal("no uuids")
		}

		return uuids[len(uuids)-1].String()
	}

	r, err := svc.CreateRoom(ctx, master, "Dungeon")
	expected := &room.Room{
		ID:           room.RoomID(lastID(t)),
		MasterID:     "master",
		Name:         "Dungeon",
		LastActivity: now,
		CreatedAt:    now,
		Members:      []room.User{master},
		Invites:      []room.Invite{},
	}

	requireRoom(t, r, expected, err)

	srarTable := []struct {
		Name   string
		Send   bool
		Accept bool
	}{
		{"SendAccept", true, true},
		{"SendReject", true, false},
		{"RequestAccept", false, true},
		{"RequestReject", false, false},
	}

	for _, tt := range srarTable {
		t.Run(tt.Name, func(t *testing.T) {
			var joined bool
			var dir room.InviteDirection
			u := room.User{ID: room.UserID(tt.Name), Name: "test"}

			if tt.Send {
				r, joined, err = svc.SendInvite(ctx, r.ID, u)
				dir = room.ToUser
			} else {
				r, joined, err = svc.RequestRoomJoin(ctx, u, r.ID)
				dir = room.FromUser
			}

			localExpected := expected.Copy()
			localExpected.Invites = append(localExpected.Invites, room.Invite{
				ID:        room.InviteID(lastID(t)),
				UserID:    u.ID,
				Name:      u.Name,
				Direction: dir,
				CreatedAt: now,
			})

			require.False(t, joined)
			requireRoom(t, r, localExpected, err)

			if tt.Accept {
				r, err = svc.AcceptInvite(ctx, r.ID, room.InviteID(lastID(t)))
				localExpected.Members = append(localExpected.Members, u)
			} else {
				r, err = svc.RejectInvite(ctx, r.ID, room.InviteID(lastID(t)))
			}

			localExpected.Invites = localExpected.Invites[0 : len(localExpected.Invites)-1]

			requireRoom(t, r, localExpected, err)

			expected = localExpected
		})
	}
}

func TestService_UUIDErrors(t *testing.T) {
	ctx := t.Context()
	now := time.Now()
	nowFunc := func() time.Time { return now }
	repo := memory_room_repository.New(memory_room_repository.WithNowFunc(nowFunc))
	master := room.User{ID: "master", Name: "Master"}
	errUuidFactory := func() (u uuid.UUID, err error) {
		err = require.ErrTest
		return
	}

	t.Run("CreateRoom", func(t *testing.T) {
		svc := New(repo, WithNowFunc(nowFunc), WithUUIDFactory(errUuidFactory))

		_, err := svc.CreateRoom(ctx, master, "Dungeon")

		require.ErrorIs(t, err, require.ErrTest)
	})

	t.Run("SendInvite", func(t *testing.T) {
		svc := New(repo, WithNowFunc(nowFunc), WithUUIDFactory(uuid.NewUUID))
		r, err := svc.CreateRoom(ctx, master, "Dungeon")

		require.NoError(t, err)
		require.NotNil(t, r)

		svc = New(repo, WithNowFunc(nowFunc), WithUUIDFactory(errUuidFactory))

		r, joined, err := svc.SendInvite(ctx, r.ID, room.User{ID: "user", Name: "User"})

		require.ErrorIs(t, err, require.ErrTest)
		require.False(t, joined)
		require.Nil(t, r)
	})
}

func TestService_RepoErrors(t *testing.T) {
	ctx := t.Context()
	now := time.Now()
	nowFunc := func() time.Time { return now }
	repo := memory_room_repository.New(memory_room_repository.WithNowFunc(nowFunc))
	svc := New(repo, WithNowFunc(nowFunc), WithUUIDFactory(uuid.NewUUID))
	master := room.User{ID: "master", Name: "Master"}
	user := room.User{ID: "user", Name: "User"}
	clearErrors := func() {
		repo.Error = nil
		repo.LockError = nil
	}
	repoError := errors.New("repo error")
	lockError := errors.New("lock error")

	t.Run("CreateRoom", func(t *testing.T) {
		t.Cleanup(clearErrors)
		repo.Error = repoError
		_, err := svc.CreateRoom(ctx, master, "Dungeon")

		require.ErrorIs(t, err, repoError)
	})

	r, err := svc.CreateRoom(ctx, master, "Dungeon")

	require.NoError(t, err)
	require.NotNil(t, r)

	testMethod := func(t *testing.T, name string, fn func() error) {
		t.Run(name, func(t *testing.T) {
			t.Cleanup(clearErrors)
			t.Run("Error", func(t *testing.T) {
				repo.Error = repoError
				err = fn()
				require.ErrorIs(t, err, repoError)
			})
			t.Run("LockError", func(t *testing.T) {
				repo.LockError = lockError
				err = fn()
				require.ErrorIs(t, err, lockError)
			})
		})
	}

	testMethod(t, "SendInvite", func() error {
		_, _, err := svc.SendInvite(ctx, r.ID, user)
		return err
	})

	testMethod(t, "RequestRoomJoin", func() error {
		_, _, err := svc.RequestRoomJoin(ctx, user, r.ID)
		return err
	})

	r, joined, err := svc.SendInvite(ctx, r.ID, room.User{ID: "user", Name: "User"})

	require.NoError(t, err)
	require.False(t, joined)
	inv := r.Invites[0]

	testMethod(t, "AcceptInvite", func() error {
		_, err := svc.AcceptInvite(ctx, r.ID, inv.ID)
		return err
	})

	testMethod(t, "RejectInvite", func() error {
		_, err := svc.RejectInvite(ctx, r.ID, inv.ID)
		return err
	})

	r, err = svc.AcceptInvite(ctx, r.ID, inv.ID)

	require.NoError(t, err)

	testMethod(t, "RemoveUser", func() error {
		_, err := svc.RemoveUser(ctx, r.ID, user.ID)
		return err
	})

	testMethod(t, "ChangeMaster", func() error {
		_, err := svc.ChangeMaster(ctx, r.ID, user.ID)
		return err
	})

	t.Run("DestroyRoom", func(t *testing.T) {
		t.Cleanup(clearErrors)
		repo.Error = repoError
		err = svc.DestroyRoom(ctx, r.ID)

		require.ErrorIs(t, err, repoError)
	})
}

func TestService_NotFounds(t *testing.T) {
	ctx := t.Context()
	now := time.Now()
	nowFunc := func() time.Time { return now }
	repo := memory_room_repository.New(memory_room_repository.WithNowFunc(nowFunc))
	master := room.User{ID: "master", Name: "Master"}
	svc := New(repo, WithNowFunc(nowFunc), WithUUIDFactory(uuid.NewUUID))
	r, err := svc.CreateRoom(ctx, master, "Dungeon")

	require.NoError(t, err)
	require.NotNil(t, r)

	t.Run("RemoveUser", func(t *testing.T) {
		_, err = svc.RemoveUser(ctx, r.ID, "user")

		require.NoError(t, err)
	})

	t.Run("RejectInvite", func(t *testing.T) {
		_, err = svc.RejectInvite(ctx, r.ID, "test")

		require.ErrorIs(t, err, room.ErrInviteNotFound)
	})

	t.Run("AcceptInvite", func(t *testing.T) {
		_, err = svc.AcceptInvite(ctx, r.ID, "test")

		require.ErrorIs(t, err, room.ErrInviteNotFound)
	})

	t.Run("ChangeMaster", func(t *testing.T) {
		_, err = svc.ChangeMaster(ctx, r.ID, "user")

		require.NoError(t, err)
	})
}

func TestService_DoubleInvite(t *testing.T) {
	ctx := t.Context()
	now := time.Now()
	nowFunc := func() time.Time { return now }
	repo := memory_room_repository.New(memory_room_repository.WithNowFunc(nowFunc))
	master := room.User{ID: "master", Name: "Master"}
	user := room.User{ID: "user", Name: "User"}
	svc := New(repo, WithNowFunc(nowFunc), WithUUIDFactory(uuid.NewUUID))
	r, err := svc.CreateRoom(ctx, master, "Dungeon")

	require.NoError(t, err)
	require.NotNil(t, r)

	r, joined, err := svc.SendInvite(ctx, r.ID, user)

	require.NoError(t, err)
	require.False(t, joined)
	require.NotNil(t, r)

	r, joined, err = svc.SendInvite(ctx, r.ID, user)

	require.ErrorIs(t, err, room.ErrAlreadyInvited)
	require.False(t, joined)
	require.Nil(t, r)
}
