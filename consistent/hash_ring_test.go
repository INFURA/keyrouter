package consistent_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/INFURA/keyrouter/consistent"
)

func TestHashRing_Get(t *testing.T) {
	members := consistent.Members{
		consistent.Member("10.a.a.a:8xxx"),
		consistent.Member("10.b.b.b:8xxx"),
		consistent.Member("10.c.c.c:8xxx"),
	}

	ring := consistent.NewHashRing()
	_, _, err := ring.Set(members)

	require.NoError(t, err)

	targets, err := ring.Get("/", 3)

	require.NoError(t, err)
	require.Equal(t, 3, len(targets))

	// Make sure the three results are unique
	require.False(t, targets[0].String() == targets[1].String())
	require.False(t, targets[1].String() == targets[2].String())
	require.False(t, targets[0].String() == targets[2].String())
}

var targets = consistent.Members{}

func BenchmarkHashRing_Get(b *testing.B) {
	members := consistent.Members{
		consistent.Member("10.a.a.a:8xxx"),
		consistent.Member("10.b.b.b:8xxx"),
		consistent.Member("10.c.c.c:8xxx"),
	}

	ring := consistent.NewHashRing()
	_, _, err := ring.Set(members)

	require.NoError(b, err)
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		targets, _ = ring.Get("/", 3)
	}
}
