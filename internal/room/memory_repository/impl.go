package memory_room_repository

import (
	"sync"
	"time"

	"github.com/ZipFile/vrc-osc-mesh/internal/room"
)

type Repository struct {
	rooms map[room.RoomID]*room.Room
	mu    sync.RWMutex
	now   func() time.Time
}

var _ room.Repository = (*Repository)(nil)

type Option func(*Repository)

func WithNow(now func() time.Time) Option {
	return func(rr *Repository) {
		rr.now = now
	}
}

func New(options ...Option) *Repository {
	r := &Repository{
		rooms: make(map[room.RoomID]*room.Room),
		now:   time.Now,
	}

	for _, option := range options {
		option(r)
	}

	return r
}

func (rr *Repository) Add(r room.Room) error {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	rr.rooms[r.ID] = r.Copy()
	return nil
}

func (rr *Repository) Get(id room.RoomID) (*room.Room, error) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()
	return rr.rooms[id].Copy(), nil
}

func (rr *Repository) Delete(id room.RoomID) error {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	delete(rr.rooms, id)
	return nil
}

func (rr *Repository) ListExpired(ttl time.Duration) ([]room.RoomID, error) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

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

func (rr *Repository) ListForUser(id room.UserID) ([]room.Room, error) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()
	var rooms []room.Room
	var m room.Member
	var i room.Invite
	var r *room.Room

	for _, r = range rr.rooms {
		if r.MasterID == id {
			goto match
		}
		for _, m = range r.Members {
			if m.UserID == id {
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
