package room

import (
	"slices"
	"time"
)

func (r *Room) ChangeMaster(id UserID, now time.Time) bool {
	if r == nil || r.MasterID == id {
		return false
	}

	for _, member := range r.Members {
		if member.ID == id {
			r.LastActivity = now
			r.MasterID = id
			return true
		}
	}

	return false
}

func (r *Room) addMember(m User, now time.Time) {
	r.LastActivity = now
	r.Members = append(r.Members, m)
	r.Invites = slices.DeleteFunc(r.Invites, func(invite Invite) bool {
		return invite.UserID == m.ID
	})
}

func (r *Room) AddMember(m User, now time.Time) error {
	if r.IsMember(m.ID) {
		return ErrAlreadyMember
	}

	r.addMember(m, now)

	return nil
}

func (r *Room) IsMember(id UserID) bool {
	for _, member := range r.Members {
		if member.ID == id {
			return true
		}
	}
	return false
}

func (r *Room) RemoveMember(id UserID, now time.Time) bool {
	for i, member := range r.Members {
		if member.ID == id && member.ID != r.MasterID {
			r.LastActivity = now
			r.Members = slices.Delete(r.Members, i, i+1)
			return true
		}
	}

	return false
}

func (r *Room) Copy() *Room {
	if r == nil {
		return nil
	}

	members := make([]User, len(r.Members))
	invites := make([]Invite, len(r.Invites))

	copy(members, r.Members)
	copy(invites, r.Invites)

	c := *r
	c.Invites = invites
	c.Members = members

	return &c
}
