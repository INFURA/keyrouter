package consistent

import (
	wrapped "github.com/buraksezer/consistent"
	"github.com/cespare/xxhash"
	"github.com/pkg/errors"
)

// HashRing is our representation of a consistent hash ring (https://en.wikipedia.org/wiki/Consistent_hashing#Technique)
type HashRing struct {
	inner *wrapped.Consistent
}

// Member represents a destination member or bucket on the hash ring
type Member string
type Members []Member

func (m Member) String() string {
	return string(m)
}

// converts our Member type to the internal wrapped member type
func (members Members) asWrappedMembers() []wrapped.Member {
	if len(members) == 0 {
		return nil
	}

	wrappedMembers := make([]wrapped.Member, len(members))
	for i, m := range members {
		wrappedMembers[i] = &m
	}

	return wrappedMembers
}

// finds the Set-wise difference: {members} - {others}
func (members Members) difference(others Members) (diff Members) {
	m := make(map[Member]bool)

	for _, item := range others {
		m[item] = true
	}

	for _, item := range members {
		if _, ok := m[item]; !ok {
			diff = append(diff, item)
		}
	}
	return
}

type hasher struct{}

func (h hasher) Sum64(data []byte) uint64 {
	return xxhash.Sum64(data)
}

//NewHashRing returns a new HashRing
func NewHashRing(members ...Member) *HashRing {
	cfg := wrapped.Config{
		PartitionCount:    15739,
		ReplicationFactor: 51,
		Load:              1.25,
		Hasher:            hasher{},
	}

	return &HashRing{
		inner: wrapped.New(Members(members).asWrappedMembers(), cfg),
	}
}

//Set updates the list of members on a hash ring
func (h *HashRing) Set(members Members) (added Members, removed Members, err error) {
	var existing Members
	for _, m := range h.inner.GetMembers() {
		existing = append(existing, Member(m.String()))
	}

	added = members.difference(existing)
	removed = existing.difference(members)

	for _, a := range added {
		if err := h.Add(a); err != nil {
			return nil, nil, errors.Wrap(err, "could not add new member")
		}
	}

	for _, r := range removed {
		if err := h.Remove(r); err != nil {
			return nil, nil, errors.Wrap(err, "could not remove removed member")
		}
	}

	return
}

//Add adds a single member to a hash ring
func (h *HashRing) Add(member Member) error {
	h.inner.Add(member)
	return nil
}

//Remove removes a single member from a hash ring
func (h *HashRing) Remove(member Member) error {
	h.inner.Remove(member.String())
	return nil
}

//Get returns the count closest members to key on the hash ring
func (h *HashRing) Get(key string, count int) (Members, error) {
	innerMembers, err := h.inner.GetClosestN([]byte(key), count)
	if err != nil {
		return nil, errors.Wrap(err, "could not get closest members")
	}

	members := make(Members, len(innerMembers))
	for i, m := range innerMembers {
		members[i] = Member(m.String())
	}

	return members, nil
}
