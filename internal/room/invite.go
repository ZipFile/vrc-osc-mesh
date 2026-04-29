package room

import (
	"slices"
	"time"
)

func (i *Invite) Member() User {
	return User{ID: i.UserID, Name: i.Name}
}

func (i *Invite) IsJoinRequest() bool {
	return i.Direction == FromUser
}

func (r *Room) AddInvite(inv Invite, now time.Time) (bool, error) {
	if r.IsMember(inv.UserID) {
		return false, ErrAlreadyMember
	}

	for i := range r.Invites {
		if r.Invites[i].UserID == inv.UserID {
			if r.Invites[i].Direction == inv.Direction {
				return false, ErrAlreadyInvited
			}

			r.addMember(inv.Member(), now) // updates r.Invites

			return true, nil
		}
	}

	r.LastActivity = now
	r.Invites = append(r.Invites, inv)

	return false, nil
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
