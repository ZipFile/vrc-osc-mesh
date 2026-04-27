package room

import (
	"slices"
	"time"
)

func (r *Room) AddMember(m Member, now time.Time) error {
	for _, member := range r.Members {
		if member.UserID == m.UserID {
			return ErrAlreadyMember
		}
	}

	r.LastActivity = now
	r.Members = append(r.Members, m)
	r.Invites = slices.DeleteFunc(r.Invites, func(invite Invite) bool {
		return invite.UserID == m.UserID
	})

	return nil
}

func (r *Room) IsMember(id UserID) bool {
	for _, member := range r.Members {
		if member.UserID == id {
			return true
		}
	}
	return false
}

func (r *Room) RemoveMember(id UserID, now time.Time) {
	for i, member := range r.Members {
		if member.UserID == id {
			r.LastActivity = now
			r.Members = slices.Delete(r.Members, i, i+1)
			return
		}
	}
}

func (r *Room) Copy() *Room {
	if r == nil {
		return nil
	}

	members := make([]Member, len(r.Members))
	invites := make([]Invite, len(r.Invites))

	copy(members, r.Members)
	copy(invites, r.Invites)

	c := *r
	c.Invites = invites
	c.Members = members

	return &c
}
