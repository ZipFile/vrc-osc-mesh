package memory_room_repository

import (
	"context"
	"sync"
	"time"

	"github.com/ZipFile/vrc-osc-mesh/internal/room"
)

type Repository struct {
	rooms     map[room.RoomID]*room.Room
	locks     map[room.RoomID]*sync.Mutex
	mu        sync.RWMutex
	now       func() time.Time
	Error     error
	LockError error
}

var _ room.Repository = (*Repository)(nil)

type Option func(*Repository)

func WithNowFunc(now func() time.Time) Option {
	return func(rr *Repository) {
		rr.now = now
	}
}

func New(options ...Option) *Repository {
	r := &Repository{
		rooms: make(map[room.RoomID]*room.Room),
		locks: make(map[room.RoomID]*sync.Mutex),
		now:   time.Now,
	}

	for _, option := range options {
		option(r)
	}

	return r
}

func (rr *Repository) Add(_ context.Context, r room.Room) error {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if rr.Error != nil {
		return rr.Error
	}

	rr.rooms[r.ID] = r.Copy()
	rr.locks[r.ID] = &sync.Mutex{}
	return nil
}

func (rr *Repository) Get(_ context.Context, id room.RoomID) (*room.Room, error) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	if rr.Error != nil {
		return nil, rr.Error
	}

	return rr.rooms[id].Copy(), nil
}

func (rr *Repository) Delete(_ context.Context, id room.RoomID) error {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if rr.Error != nil {
		return rr.Error
	}

	delete(rr.rooms, id)
	delete(rr.locks, id)

	return nil
}

func (rr *Repository) ListExpired(_ context.Context, ttl time.Duration) ([]room.RoomID, error) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if rr.Error != nil {
		return nil, rr.Error
	}

	deadline := rr.now().Add(-ttl)
	toDelete := make([]room.RoomID, 0)

	for _, r := range rr.rooms {
		if r.LastActivity.Before(deadline) {
			toDelete = append(toDelete, r.ID)
		}
	}

	for _, id := range toDelete {
		delete(rr.rooms, id)
	}

	return toDelete, nil
}

func (rr *Repository) ListForUser(_ context.Context, id room.UserID) ([]room.Room, error) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	if rr.Error != nil {
		return nil, rr.Error
	}

	var rooms []room.Room
	var m room.User
	var i room.Invite
	var r *room.Room

	for _, r = range rr.rooms {
		if r.MasterID == id {
			goto match
		}
		for _, m = range r.Members {
			if m.ID == id {
				goto match
			}
		}
		for _, i = range r.Invites {
			if i.UserID == id {
				goto match
			}
		}
		continue

	match:
		rooms = append(rooms, *r.Copy())
	}

	return rooms, nil
}

func (rr *Repository) Lock(ctx context.Context, id room.RoomID, fn func(context.Context, *room.Room) error) error {
	rr.mu.RLock()

	if rr.LockError != nil {
		rr.mu.RUnlock()
		return rr.LockError
	}

	mu, ok := rr.locks[id]
	rr.mu.RUnlock()

	if !ok {
		return room.ErrRoomNotFound
	}

	mu.Lock()
	defer mu.Unlock()

	return fn(ctx, rr.rooms[id].Copy())
}
