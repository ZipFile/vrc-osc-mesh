package room_service

import (
	"context"
	"time"

	"github.com/ZipFile/vrc-osc-mesh/internal/room"
	"github.com/google/uuid"
)

type Service struct {
	repo        room.Repository
	uuidFactory func() (uuid.UUID, error)
	nowFunc     func() time.Time
}

var _ room.Service = (*Service)(nil)

type Option func(*Service)

func WithUUIDFactory(uuidFactory func() (uuid.UUID, error)) Option {
	return func(s *Service) {
		s.uuidFactory = uuidFactory
	}
}

func WithNowFunc(nowFunc func() time.Time) Option {
	return func(s *Service) {
		s.nowFunc = nowFunc
	}
}

func New(repo room.Repository, options ...Option) *Service {
	s := &Service{
		repo:        repo,
		uuidFactory: uuid.NewV7,
		nowFunc:     time.Now,
	}

	for _, option := range options {
		option(s)
	}

	return s
}

func (s *Service) CreateRoom(ctx context.Context, master room.User, name string) (*room.Room, error) {
	id, err := s.uuidFactory()

	if err != nil {
		return nil, err
	}

	roomId := room.RoomID(id.String())
	now := s.nowFunc()
	r := master.NewRoom(roomId, name, now)
	err = s.repo.Add(ctx, *r)

	if err != nil {
		return nil, err
	}

	return r, nil
}

func (s *Service) DestroyRoom(ctx context.Context, id room.RoomID) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) RequestRoomJoin(ctx context.Context, u room.User, roomID room.RoomID) (*room.Room, bool, error) {
	return s.sendInvite(ctx, roomID, u, room.FromUser)
}

func (s *Service) SendInvite(ctx context.Context, roomID room.RoomID, u room.User) (*room.Room, bool, error) {
	return s.sendInvite(ctx, roomID, u, room.ToUser)
}

func (s *Service) sendInvite(ctx context.Context, roomID room.RoomID, u room.User, dir room.InviteDirection) (*room.Room, bool, error) {
	var out *room.Room
	var joined bool

	lockErr := s.repo.Lock(ctx, roomID, func(ctx context.Context, r *room.Room) error {
		now := s.nowFunc()
		id, err := s.uuidFactory()

		if err != nil {
			return err
		}

		invID := room.InviteID(id.String())
		var inv *room.Invite

		if dir == room.FromUser {
			inv = u.NewJoinRequest(invID, now)
		} else {
			inv = u.NewInvite(invID, now)
		}

		joined, err = r.AddInvite(*inv, now)

		if err != nil {
			return err
		}

		out = r

		return s.repo.Add(ctx, *r)
	})

	return out, joined, lockErr
}

func (s *Service) AcceptInvite(ctx context.Context, roomID room.RoomID, invID room.InviteID) (*room.Room, error) {
	var out *room.Room

	lockErr := s.repo.Lock(ctx, roomID, func(ctx context.Context, r *room.Room) error {
		err := r.AcceptInvite(invID, s.nowFunc())

		if err != nil {
			return err
		}

		out = r

		return s.repo.Add(ctx, *r)
	})

	return out, lockErr
}

func (s *Service) RejectInvite(ctx context.Context, roomID room.RoomID, invID room.InviteID) (*room.Room, error) {
	var out *room.Room

	lockErr := s.repo.Lock(ctx, roomID, func(ctx context.Context, r *room.Room) error {
		err := r.RemoveInvite(invID, s.nowFunc())

		if err != nil {
			return err
		}

		out = r

		return s.repo.Add(ctx, *r)
	})

	return out, lockErr
}

func (s *Service) RemoveUser(ctx context.Context, roomID room.RoomID, userID room.UserID) (*room.Room, error) {
	var out *room.Room

	lockErr := s.repo.Lock(ctx, roomID, func(ctx context.Context, r *room.Room) error {
		removed := r.RemoveMember(userID, s.nowFunc())

		if !removed {
			return nil
		}

		out = r

		return s.repo.Add(ctx, *r)
	})

	return out, lockErr
}

func (s *Service) ChangeMaster(ctx context.Context, roomID room.RoomID, userID room.UserID) (*room.Room, error) {
	var out *room.Room

	lockErr := s.repo.Lock(ctx, roomID, func(ctx context.Context, r *room.Room) error {
		changed := r.ChangeMaster(userID, s.nowFunc())

		if !changed {
			return nil
		}

		out = r

		return s.repo.Add(ctx, *r)
	})

	return out, lockErr
}
