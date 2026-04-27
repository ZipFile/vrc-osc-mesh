package room

import (
	"slices"
	"time"
)

func (i *Invite) Member() Member {
	return Member{UserID: i.UserID, Name: i.Name}
}

func (i *Invite) IsJoinRequest() bool {
	return i.Direction == FromUser
}

func (r *Room) AddInvite(i Invite, now time.Time) error {
	if r.IsMember(i.UserID) {
		return ErrAlreadyMember
	}

	for _, invite := range r.Invites {
		if invite.UserID == i.UserID {
			if invite.Direction == i.Direction {
				return ErrAlreadyInvited
			}

			return r.AddMember(i.Member(), now)
		}
	}

	r.LastActivity = now
	r.Invites = append(r.Invites, i)

	return nil
}

func (r *Room) isInvited(id UserID, direction InviteDirection) bool {
	for _, invite := range r.Invites {
		if invite.Direction == direction && invite.UserID == id {
			return true
		}
	}
	return false
}

func (r *Room) IsInvited(id UserID) bool {
	return r.isInvited(id, ToUser)
}

func (r *Room) IsRequested(id UserID) bool {
	return r.isInvited(id, FromUser)
}

func (r *Room) RemoveInvite(id InviteID, now time.Time) error {
	for i, invite := range r.Invites {
		if invite.ID == id {
			r.LastActivity = now
			r.Invites = slices.Delete(r.Invites, i, i+1)
			return nil
		}
	}
	return ErrInviteNotFound
}

func (r *Room) AcceptInvite(id InviteID, now time.Time) error {
	for _, invite := range r.Invites {
		if invite.ID == id {
			return r.AddMember(invite.Member(), now)
		}
	}
	return ErrInviteNotFound
}
