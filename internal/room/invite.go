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

func (i *Invite) IsAcceptable(by, master UserID) bool {
	switch {
	case i == nil:
		return false
	case by == i.UserID:
		return i.Direction == ToUser
	case by == master:
		return i.Direction == FromUser
	default:
		return false
	}
}

func (i *Invite) IsRejectable(by, master UserID) bool {
	if i == nil {
		return false
	}

	return by == i.UserID || by == master
}

func (r *Room) GetInvite(id InviteID) *Invite {
	for _, invite := range r.Invites {
		if invite.ID == id {
			return &invite
		}
	}
	return nil
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
