package memory_room_repository

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/require"
	"github.com/google/uuid"

	"github.com/ZipFile/vrc-osc-mesh/internal/room"
)

var testDate = time.Date(2026, 4, 24, 16, 45, 0, 0, time.UTC)

func TestNew(t *testing.T) {
	repo := New(WithNowFunc(func() time.Time { return testDate }))

	require.Equal(t, repo.now(), testDate)
}

func TestRepository_AddGetDelete(t *testing.T) {
	ctx := t.Context()
	repo := New()
	roomID := room.RoomID(uuid.New().String())
	inviteID := room.InviteID(uuid.New().String())

	r := room.Room{
		ID:           roomID,
		MasterID:     "test",
		Name:         "test",
		LastActivity: time.Now(),
		CreatedAt:    time.Now(),
		Members: []room.User{
			{ID: "test", Name: "test"},
		},
		Invites: []room.Invite{
			{ID: inviteID, UserID: "test", Direction: room.ToUser, Name: "test", CreatedAt: testDate},
			{ID: inviteID, UserID: "test", Direction: room.FromUser, Name: "test", CreatedAt: testDate},
		},
	}

	safeCopy := r.Copy()
	err := repo.Add(ctx, r)

	require.NoError(t, err)

	r.Name = "modified"
	r.LastActivity = testDate
	r.CreatedAt = testDate
	r.Invites[0].UserID = "modified"
	r.Invites[1].UserID = "modified"
	r.Members = []room.User{
		{ID: "modified", Name: "modified"},
	}

	stored, err := repo.Get(ctx, roomID)

	require.NoError(t, err)
	require.Equal(t, safeCopy, stored)

	err = repo.Delete(ctx, roomID)

	require.NoError(t, err)

	stored, err = repo.Get(ctx, roomID)

	require.NoError(t, err)
	require.Nil(t, stored)
}

func TestRepository_ListExpired(t *testing.T) {
	var err error
	ctx := t.Context()
	repo := New()
	normalRoomID := room.RoomID(uuid.New().String())
	expiredRoomID := room.RoomID(uuid.New().String())

	rooms := []room.Room{
		{
			ID:           normalRoomID,
			MasterID:     "test",
			Name:         "test",
			LastActivity: time.Now(),
			CreatedAt:    time.Now(),
		},
		{
			ID:           expiredRoomID,
			MasterID:     "test",
			Name:         "expired",
			LastActivity: time.Now().Add(-1*time.Hour - time.Minute),
			CreatedAt:    testDate,
		},
	}

	for _, r := range rooms {
		err = repo.Add(ctx, r)
		require.NoError(t, err)
	}

	ids, err := repo.ListExpired(ctx, time.Hour)

	require.NoError(t, err)
	require.Equal(t, []room.RoomID{expiredRoomID}, ids)
}

func TestRepository_ListForUser(t *testing.T) {
	var err error
	ctx := t.Context()
	repo := New()
	userID := room.UserID(uuid.New().String())

	rooms := []room.Room{
		{
			ID:           room.RoomID(uuid.New().String()),
			MasterID:     userID,
			Name:         "master",
			LastActivity: testDate.Add(1 * time.Second),
			CreatedAt:    testDate,
			Members:      make([]room.User, 0),
			Invites:      make([]room.Invite, 0),
		},
		{
			ID:           room.RoomID(uuid.New().String()),
			MasterID:     room.UserID(uuid.New().String()),
			Name:         "invited",
			LastActivity: testDate.Add(3 * time.Second),
			CreatedAt:    testDate,
			Members:      make([]room.User, 0),
			Invites: []room.Invite{
				{
					ID:        room.InviteID(uuid.New().String()),
					UserID:    userID,
					Name:      "test",
					Direction: room.ToUser,
					CreatedAt: testDate,
				},
			},
		},
		{
			ID:           room.RoomID(uuid.New().String()),
			MasterID:     room.UserID(uuid.New().String()),
			Name:         "member",
			LastActivity: testDate.Add(4 * time.Second),
			CreatedAt:    testDate,
			Members: []room.User{
				{ID: userID, Name: "test"},
			},
			Invites: make([]room.Invite, 0),
		},
		{
			ID:           room.RoomID(uuid.New().String()),
			MasterID:     "test",
			Name:         "test",
			LastActivity: time.Now(),
			CreatedAt:    time.Now(),
			Members: []room.User{
				{ID: "test", Name: "test"},
			},
			Invites: []room.Invite{
				{
					ID:        room.InviteID(uuid.New().String()),
					UserID:    "test",
					Name:      "test",
					Direction: room.ToUser,
					CreatedAt: testDate,
				},
			},
		},
	}

	for _, r := range rooms {
		err = repo.Add(ctx, r)

		require.NoError(t, err)
	}

	storedRooms, err := repo.ListForUser(ctx, userID)

	require.NoError(t, err)
	require.Equal(t, 3, len(storedRooms))

	sort.Slice(storedRooms, func(i, j int) bool {
		return storedRooms[i].LastActivity.Before(storedRooms[j].LastActivity)
	})

	for i, r := range storedRooms {
		require.Equal(t, rooms[i], r)
	}
}

func TestRepository_Lock(t *testing.T) {
	ctx := t.Context()
	repo := New()
	r := room.Room{
		ID:           room.RoomID(uuid.New().String()),
		MasterID:     room.UserID(uuid.New().String()),
		Name:         "master",
		LastActivity: testDate.Add(1 * time.Second),
		CreatedAt:    testDate,
		Members:      make([]room.User, 0),
		Invites:      make([]room.Invite, 0),
	}

	t.Run("not found", func(t *testing.T) {
		err := repo.Lock(ctx, r.ID, nil)

		require.ErrorIs(t, err, room.ErrRoomNotFound)
	})

	require.NoError(t, repo.Add(ctx, r))

	t.Run("proxy error", func(t *testing.T) {
		err := repo.Lock(ctx, r.ID, func(context.Context, *room.Room) error {
			return require.ErrTest
		})

		require.ErrorIs(t, err, require.ErrTest)
	})

	t.Run("ok", func(t *testing.T) {
		done := make(chan any)
		doneClosed := false
		takeLock := make(chan any)
		takeLockClosed := false

		defer func() {
			if !takeLockClosed {
				close(takeLock)
			}
			if !doneClosed {
				close(done)
			}
		}()

		go func() {
			defer func() {
				close(done)
				doneClosed = true
			}()
			<-takeLock
			err := repo.Lock(ctx, r.ID, func(ctx context.Context, r *room.Room) error {
				r.Name = "modified twice"
				return repo.Add(ctx, *r)
			})
			require.NoError(t, err)
		}()

		err := repo.Lock(ctx, r.ID, func(ctx context.Context, r *room.Room) error {
			close(takeLock)
			takeLockClosed = true
			r.Name = "modified"
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return repo.Add(ctx, *r)
			}
		})

		require.NoError(t, err)

		select {
		case <-ctx.Done():
			return
		case <-done:
			require.NoError(t, err)
		}
	})
}

func TestErrors(t *testing.T) {
	testMethod := func(t *testing.T, name string, fn func() error) {
		t.Run(name, func(t *testing.T) {
			err := fn()
			require.ErrorIs(t, err, require.ErrTest)
		})
	}
	ctx := t.Context()
	repo := New()
	repo.Error = require.ErrTest
	repo.LockError = require.ErrTest
	r := room.Room{
		ID:           room.RoomID(uuid.New().String()),
		MasterID:     room.UserID(uuid.New().String()),
		Name:         "master",
		LastActivity: testDate.Add(1 * time.Second),
		CreatedAt:    testDate,
		Members:      make([]room.User, 0),
		Invites:      make([]room.Invite, 0),
	}

	testMethod(t, "Add", func() error {
		return repo.Add(ctx, r)
	})
	testMethod(t, "Get", func() error {
		_, err := repo.Get(ctx, r.ID)
		return err
	})
	testMethod(t, "Delete", func() error {
		return repo.Delete(ctx, r.ID)
	})
	testMethod(t, "ListExpired", func() error {
		_, err := repo.ListExpired(ctx, time.Hour)
		return err
	})
	testMethod(t, "ListForUser", func() error {
		_, err := repo.ListForUser(ctx, r.MasterID)
		return err
	})
	testMethod(t, "Lock", func() error {
		return repo.Lock(ctx, r.ID, nil)
	})
}
