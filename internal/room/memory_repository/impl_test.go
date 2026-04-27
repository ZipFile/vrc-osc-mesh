package memory_room_repository

import (
	"sort"
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/require"
	"github.com/google/uuid"

	"github.com/ZipFile/vrc-osc-mesh/internal/room"
)

var testDate = time.Date(2026, 4, 24, 16, 45, 0, 0, time.UTC)

func TestNew(t *testing.T) {
	repo := New(WithNow(func() time.Time { return testDate }))

	require.Equal(t, repo.now(), testDate)
}

func TestRepository_AddGetDelete(t *testing.T) {
	repo := New()
	roomID := room.RoomID(uuid.New())
	inviteID := room.InviteID(uuid.New())
	joinRequestID := room.JoinRequestID(uuid.New())

	r := room.Room{
		ID:           roomID,
		MasterID:     "test",
		Name:         "test",
		LastActivity: time.Now(),
		CreatedAt:    time.Now(),
		Members: []room.Member{
			{UserID: "test", Name: "test"},
		},
		Invites: []room.Invite{
			{ID: inviteID, To: "test", From: "test"},
		},
		Requests: []room.JoinRequest{
			{ID: joinRequestID, To: "test", From: "test"},
		},
	}

	safeCopy := r.Copy()
	err := repo.Add(r)

	require.NoError(t, err)

	r.Name = "modified"
	r.LastActivity = testDate
	r.CreatedAt = testDate
	r.Invites[0].From = "modified"
	r.Requests[0].From = "modified"
	r.Members = []room.Member{
		{UserID: "modified", Name: "modified"},
	}

	stored, err := repo.Get(roomID)

	require.NoError(t, err)
	require.Equal(t, safeCopy, stored)

	err = repo.Delete(roomID)

	require.NoError(t, err)

	stored, err = repo.Get(roomID)

	require.NoError(t, err)
	require.Nil(t, stored)
}

func TestRepository_ListExpired(t *testing.T) {
	var err error
	repo := New()
	normalRoomID := room.RoomID(uuid.New())
	expiredRoomID := room.RoomID(uuid.New())

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
		err = repo.Add(r)
		require.NoError(t, err)
	}

	ids, err := repo.ListExpired(time.Hour)

	require.NoError(t, err)
	require.Equal(t, []room.RoomID{expiredRoomID}, ids)
}

func TestRepository_ListForUser(t *testing.T) {
	var err error
	repo := New()
	userID := room.UserID(uuid.New().String())
	otherUserID := room.UserID(uuid.New().String())

	rooms := []room.Room{
		{
			ID:           room.RoomID(uuid.New()),
			MasterID:     userID,
			Name:         "master",
			LastActivity: testDate.Add(1 * time.Second),
			CreatedAt:    testDate,
			Members:      make([]room.Member, 0),
			Invites:      make([]room.Invite, 0),
			Requests:     make([]room.JoinRequest, 0),
		},
		{
			ID:           room.RoomID(uuid.New()),
			MasterID:     room.UserID(uuid.New().String()),
			Name:         "requested",
			LastActivity: testDate.Add(2 * time.Second),
			CreatedAt:    testDate,
			Members:      make([]room.Member, 0),
			Invites:      make([]room.Invite, 0),
			Requests: []room.JoinRequest{
				{ID: room.JoinRequestID(uuid.New()), To: otherUserID, From: userID, CreatedAt: testDate},
			},
		},
		{
			ID:           room.RoomID(uuid.New()),
			MasterID:     room.UserID(uuid.New().String()),
			Name:         "invited",
			LastActivity: testDate.Add(3 * time.Second),
			CreatedAt:    testDate,
			Members:      make([]room.Member, 0),
			Invites: []room.Invite{
				{ID: room.InviteID(uuid.New()), To: userID, From: otherUserID, CreatedAt: testDate},
			},
			Requests: make([]room.JoinRequest, 0),
		},
		{
			ID:           room.RoomID(uuid.New()),
			MasterID:     room.UserID(uuid.New().String()),
			Name:         "member",
			LastActivity: testDate.Add(4 * time.Second),
			CreatedAt:    testDate,
			Members: []room.Member{
				{UserID: userID, Name: "test"},
			},
			Invites:  make([]room.Invite, 0),
			Requests: make([]room.JoinRequest, 0),
		},
		{
			ID:           room.RoomID(uuid.New()),
			MasterID:     "test",
			Name:         "test",
			LastActivity: time.Now(),
			CreatedAt:    time.Now(),
			Members: []room.Member{
				{UserID: "test", Name: "test"},
			},
			Invites: []room.Invite{
				{ID: room.InviteID(uuid.New()), To: "test", From: "test", CreatedAt: testDate},
			},
			Requests: []room.JoinRequest{
				{ID: room.JoinRequestID(uuid.New()), To: "test", From: "test", CreatedAt: testDate},
			},
		},
	}

	for _, r := range rooms {
		err = repo.Add(r)
		require.NoError(t, err)
	}

	storedRooms, err := repo.ListForUser(userID)

	require.NoError(t, err)
	require.Equal(t, 4, len(storedRooms))

	sort.Slice(storedRooms, func(i, j int) bool {
		return storedRooms[i].LastActivity.Before(storedRooms[j].LastActivity)
	})

	for i, r := range storedRooms {
		require.Equal(t, rooms[i], r)
	}
}
