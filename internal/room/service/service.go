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

func (s *Service) CreateRoom(ctx context.Context, name string) (*room.Room, error) {
	user, err := room.UserFromContext(ctx)

	if err != nil {
		return nil, err
	}

	id, err := s.uuidFactory()

	if err != nil {
		return nil, err
	}

	roomId := room.RoomID(id.String())
	now := s.nowFunc()
	r := user.NewRoom(roomId, name, now)
	err = s.repo.Add(ctx, *r)

	if err != nil {
		return nil, err
	}

	return r, nil
}

func (s *Service) DestroyRoom(ctx context.Context, roomID room.RoomID) error {
	return s.userRoomAction(ctx, roomID, func(ctx context.Context, u *room.User, r *room.Room) error {
		if r.MasterID != u.ID {
			return room.ErrNotMaster
		}

		return s.repo.Delete(ctx, r.ID)
	})
}

func (s *Service) RequestRoomJoin(ctx context.Context, roomID room.RoomID) (out *room.Room, joined bool, actionErr error) {
	actionErr = s.userRoomAction(ctx, roomID, func(ctx context.Context, u *room.User, r *room.Room) error {
		now := s.nowFunc()
		id, err := s.uuidFactory()

		if err != nil {
			return err
		}

		inv := u.NewJoinRequest(room.InviteID(id.String()), now)
		joined, err = r.AddInvite(*inv, now)

		if err != nil {
			return err
		}

		out = r

		return s.repo.Add(ctx, *r)
	})

	return
}

func (s *Service) SendInvite(ctx context.Context, roomID room.RoomID, to room.User) (out *room.Room, joined bool, actionErr error) {
	actionErr = s.userRoomAction(ctx, roomID, func(ctx context.Context, u *room.User, r *room.Room) error {
		if r.MasterID != u.ID {
			return room.ErrNotMaster
		}

		now := s.nowFunc()
		id, err := s.uuidFactory()

		if err != nil {
			return err
		}

		inv := to.NewInvite(room.InviteID(id.String()), now)
		joined, err = r.AddInvite(*inv, now)

		if err != nil {
			return err
		}

		out = r

		return s.repo.Add(ctx, *r)
	})

	return
}

func (s *Service) AcceptInvite(ctx context.Context, roomID room.RoomID, invID room.InviteID) (out *room.Room, actionErr error) {
	actionErr = s.userRoomAction(ctx, roomID, func(ctx context.Context, u *room.User, r *room.Room) error {
		inv := r.GetInvite(invID)

		if inv == nil {
			return room.ErrInviteNotFound
		}

		if !inv.IsAcceptable(u.ID, r.MasterID) {
			return room.ErrInviteNotAcceptable
		}

		_ = r.AcceptInvite(invID, s.nowFunc())

		out = r

		return s.repo.Add(ctx, *r)
	})

	return
}

func (s *Service) RejectInvite(ctx context.Context, roomID room.RoomID, invID room.InviteID) (out *room.Room, actionErr error) {
	actionErr = s.userRoomAction(ctx, roomID, func(ctx context.Context, u *room.User, r *room.Room) error {
		inv := r.GetInvite(invID)

		if !inv.IsRejectable(u.ID, r.MasterID) {
			return room.ErrInviteNotFound
		}

		_ = r.RemoveInvite(invID, s.nowFunc())

		out = r

		return s.repo.Add(ctx, *r)
	})

	return
}

func (s *Service) RemoveUser(ctx context.Context, roomID room.RoomID, userID room.UserID) (out *room.Room, actionErr error) {
	actionErr = s.userRoomAction(ctx, roomID, func(ctx context.Context, u *room.User, r *room.Room) error {
		if !(r.MasterID == u.ID || userID == u.ID) {
			return room.ErrNotMaster
		}

		err := r.RemoveMember(userID, s.nowFunc())

		if err != nil {
			return err
		}

		out = r

		return s.repo.Add(ctx, *r)
	})

	return
}

func (s *Service) ChangeMaster(ctx context.Context, roomID room.RoomID, userID room.UserID) (out *room.Room, actionErr error) {
	actionErr = s.userRoomAction(ctx, roomID, func(ctx context.Context, u *room.User, r *room.Room) error {
		if r.MasterID != u.ID {
			return room.ErrNotMaster
		}

		err := r.ChangeMaster(userID, s.nowFunc())

		if err != nil {
			return err
		}

		out = r

		return s.repo.Add(ctx, *r)
	})

	return
}

func (s *Service) userRoomAction(ctx context.Context, id room.RoomID, action func(ctx context.Context, u *room.User, r *room.Room) error) error {
	u, err := room.UserFromContext(ctx)

	if err != nil {
		return err
	}

	return s.repo.Lock(ctx, id, func(ctx context.Context, r *room.Room) error {
		return action(ctx, u, r)
	})
}
