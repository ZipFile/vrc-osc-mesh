package room_service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ZipFile/vrc-osc-mesh/internal/room"
	"github.com/go-openapi/testify/v2/require"
	"github.com/google/uuid"

	memory_room_repository "github.com/ZipFile/vrc-osc-mesh/internal/room/memory_repository"
)

func TestService_OK(t *testing.T) {
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

		r, err = repo.Get(t.Context(), r.ID)

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

	ctx := room.WithUserInContext(t.Context(), &master)

	r, err := svc.CreateRoom(ctx, "Dungeon")
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
			userCtx := room.WithUserInContext(t.Context(), &u)

			if tt.Send {
				r, joined, err = svc.SendInvite(ctx, r.ID, u)
				dir = room.ToUser
			} else {
				r, joined, err = svc.RequestRoomJoin(userCtx, r.ID)
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
				var acceptCtx context.Context
				if tt.Send {
					acceptCtx = userCtx
				} else {
					acceptCtx = ctx
				}

				r, err = svc.AcceptInvite(acceptCtx, r.ID, room.InviteID(lastID(t)))
				localExpected.Members = append(localExpected.Members, u)
			} else {
				r, err = svc.RejectInvite(ctx, r.ID, room.InviteID(lastID(t)))
			}

			localExpected.Invites = localExpected.Invites[0 : len(localExpected.Invites)-1]

			requireRoom(t, r, localExpected, err)

			expected = localExpected
		})
	}

	t.Run("same master", func(t *testing.T) {
		r, err = svc.ChangeMaster(ctx, r.ID, "master")

		require.ErrorIs(t, err, room.ErrAlreadyMaster)
	})
}

func TestService_UUIDErrors(t *testing.T) {
	now := time.Now()
	nowFunc := func() time.Time { return now }
	repo := memory_room_repository.New(memory_room_repository.WithNowFunc(nowFunc))
	master := room.User{ID: "master", Name: "Master"}
	user := room.User{ID: "user", Name: "User"}
	errUuidFactory := func() (u uuid.UUID, err error) {
		err = require.ErrTest
		return
	}
	ctx := room.WithUserInContext(t.Context(), &master)
	userCtx := room.WithUserInContext(t.Context(), &user)

	t.Run("CreateRoom", func(t *testing.T) {
		svc := New(repo, WithNowFunc(nowFunc), WithUUIDFactory(errUuidFactory))

		_, err := svc.CreateRoom(ctx, "Dungeon")

		require.ErrorIs(t, err, require.ErrTest)
	})

	t.Run("SendInvite", func(t *testing.T) {
		svc := New(repo, WithNowFunc(nowFunc), WithUUIDFactory(uuid.NewUUID))
		r, err := svc.CreateRoom(ctx, "Dungeon")

		require.NoError(t, err)
		require.NotNil(t, r)

		svc = New(repo, WithNowFunc(nowFunc), WithUUIDFactory(errUuidFactory))

		r, joined, err := svc.SendInvite(ctx, r.ID, user)

		require.ErrorIs(t, err, require.ErrTest)
		require.False(t, joined)
		require.Nil(t, r)
	})

	t.Run("RequestRoomJoin", func(t *testing.T) {
		svc := New(repo, WithNowFunc(nowFunc), WithUUIDFactory(uuid.NewUUID))
		r, err := svc.CreateRoom(ctx, "Dungeon")

		require.NoError(t, err)
		require.NotNil(t, r)

		svc = New(repo, WithNowFunc(nowFunc), WithUUIDFactory(errUuidFactory))

		r, joined, err := svc.RequestRoomJoin(userCtx, r.ID)

		require.ErrorIs(t, err, require.ErrTest)
		require.False(t, joined)
		require.Nil(t, r)
	})
}

func TestService_RepoErrors(t *testing.T) {
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
	ctx := room.WithUserInContext(t.Context(), &master)
	userCtx := room.WithUserInContext(t.Context(), &user)

	t.Run("CreateRoom", func(t *testing.T) {
		t.Cleanup(clearErrors)
		repo.Error = repoError
		_, err := svc.CreateRoom(ctx, "Dungeon")

		require.ErrorIs(t, err, repoError)
	})

	r, err := svc.CreateRoom(ctx, "Dungeon")

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
		_, _, err := svc.RequestRoomJoin(userCtx, r.ID)
		return err
	})

	r, joined, err := svc.SendInvite(ctx, r.ID, room.User{ID: "user", Name: "User"})

	require.NoError(t, err)
	require.False(t, joined)
	inv := r.Invites[0]

	testMethod(t, "AcceptInvite", func() error {
		_, err := svc.AcceptInvite(userCtx, r.ID, inv.ID)
		return err
	})

	testMethod(t, "RejectInvite", func() error {
		_, err := svc.RejectInvite(ctx, r.ID, inv.ID)
		return err
	})

	r, err = svc.AcceptInvite(userCtx, r.ID, inv.ID)

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
	now := time.Now()
	nowFunc := func() time.Time { return now }
	repo := memory_room_repository.New(memory_room_repository.WithNowFunc(nowFunc))
	master := room.User{ID: "master", Name: "Master"}
	svc := New(repo, WithNowFunc(nowFunc), WithUUIDFactory(uuid.NewUUID))
	ctx := room.WithUserInContext(t.Context(), &master)
	r, err := svc.CreateRoom(ctx, "Dungeon")

	require.NoError(t, err)
	require.NotNil(t, r)

	t.Run("RemoveUser", func(t *testing.T) {
		_, err = svc.RemoveUser(ctx, r.ID, "user")

		require.ErrorIs(t, err, room.ErrUserNotFound)
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

		require.ErrorIs(t, err, room.ErrUserNotFound)
	})
}

func TestService_NonMaster(t *testing.T) {
	now := time.Now()
	nowFunc := func() time.Time { return now }
	repo := memory_room_repository.New(memory_room_repository.WithNowFunc(nowFunc))
	master := room.User{ID: "master", Name: "Master"}
	user := room.User{ID: "user", Name: "User"}
	otherUser := room.User{ID: "other", Name: "Other"}
	svc := New(repo, WithNowFunc(nowFunc), WithUUIDFactory(uuid.NewUUID))
	ctx := room.WithUserInContext(t.Context(), &master)
	userCtx := room.WithUserInContext(t.Context(), &user)
	r, err := svc.CreateRoom(ctx, "Dungeon")

	require.NoError(t, err)
	require.NotNil(t, r)

	t.Run("DestroyRoom", func(t *testing.T) {
		err := svc.DestroyRoom(userCtx, r.ID)
		require.ErrorIs(t, err, room.ErrNotMaster)
	})

	t.Run("SendInvite", func(t *testing.T) {
		r, joined, err := svc.SendInvite(userCtx, r.ID, user)

		require.ErrorIs(t, err, room.ErrNotMaster)
		require.False(t, joined)
		require.Nil(t, r)
	})

	t.Run("RemoveUser", func(t *testing.T) {
		r, err := svc.RemoveUser(userCtx, r.ID, otherUser.ID)

		require.ErrorIs(t, err, room.ErrNotMaster)
		require.Nil(t, r)
	})

	t.Run("ChangeMaster", func(t *testing.T) {
		r, err := svc.ChangeMaster(userCtx, r.ID, otherUser.ID)

		require.ErrorIs(t, err, room.ErrNotMaster)
		require.Nil(t, r)
	})
}

func TestService_DoubleInvite(t *testing.T) {
	now := time.Now()
	nowFunc := func() time.Time { return now }
	repo := memory_room_repository.New(memory_room_repository.WithNowFunc(nowFunc))
	master := room.User{ID: "master", Name: "Master"}
	user := room.User{ID: "user", Name: "User"}
	svc := New(repo, WithNowFunc(nowFunc), WithUUIDFactory(uuid.NewUUID))
	ctx := room.WithUserInContext(t.Context(), &master)
	r, err := svc.CreateRoom(ctx, "Dungeon")

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

func TestService_DoubleRequest(t *testing.T) {
	now := time.Now()
	nowFunc := func() time.Time { return now }
	repo := memory_room_repository.New(memory_room_repository.WithNowFunc(nowFunc))
	master := room.User{ID: "master", Name: "Master"}
	user := room.User{ID: "user", Name: "User"}
	svc := New(repo, WithNowFunc(nowFunc), WithUUIDFactory(uuid.NewUUID))
	ctx := room.WithUserInContext(t.Context(), &master)
	userCtx := room.WithUserInContext(t.Context(), &user)
	r, err := svc.CreateRoom(ctx, "Dungeon")

	require.NoError(t, err)
	require.NotNil(t, r)

	r, joined, err := svc.RequestRoomJoin(userCtx, r.ID)

	require.NoError(t, err)
	require.False(t, joined)
	require.NotNil(t, r)

	r, joined, err = svc.RequestRoomJoin(userCtx, r.ID)

	require.ErrorIs(t, err, room.ErrAlreadyInvited)
	require.False(t, joined)
	require.Nil(t, r)
}

func TestService_UnacceptableInvite(t *testing.T) {
	now := time.Now()
	nowFunc := func() time.Time { return now }
	repo := memory_room_repository.New(memory_room_repository.WithNowFunc(nowFunc))
	master := room.User{ID: "master", Name: "Master"}
	user := room.User{ID: "user", Name: "User"}
	svc := New(repo, WithNowFunc(nowFunc), WithUUIDFactory(uuid.NewUUID))
	ctx := room.WithUserInContext(t.Context(), &master)
	r, err := svc.CreateRoom(ctx, "Dungeon")

	require.NoError(t, err)
	require.NotNil(t, r)

	r, _, err = svc.SendInvite(ctx, r.ID, user)

	require.NoError(t, err)
	require.NotNil(t, r)

	r, err = svc.AcceptInvite(ctx, r.ID, r.Invites[0].ID)

	require.ErrorIs(t, err, room.ErrInviteNotAcceptable)
	require.Nil(t, r)
}

func TestService_NoUser(t *testing.T) {
	now := time.Now()
	nowFunc := func() time.Time { return now }
	repo := memory_room_repository.New(memory_room_repository.WithNowFunc(nowFunc))
	svc := New(repo, WithNowFunc(nowFunc), WithUUIDFactory(uuid.NewUUID))
	ctx := context.Background()

	t.Run("CreateRoom", func(t *testing.T) {
		_, err := svc.CreateRoom(ctx, "Dungeon")

		require.ErrorIs(t, err, room.ErrNoUserInContext)
	})

	t.Run("userRoomAction", func(t *testing.T) {
		err := svc.userRoomAction(ctx, "any", nil)

		require.ErrorIs(t, err, room.ErrNoUserInContext)
	})
}
