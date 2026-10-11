package raknet

import "testing"

// BenchmarkHookCheckNoHooks measures what an unused hook costs on the hot
// path: it must be a few nanoseconds and allocate nothing.
func BenchmarkHookCheckNoHooks(b *testing.B) {
	l := &Listener{}
	c := &Conn{listener: l}
	data := []byte{1, 2, 3}
	b.ReportAllocs()
	for b.Loop() {
		if _, keep := c.runPacket(true, data); !keep {
			b.Fatal("dropped")
		}
	}
}

func TestHookListAddRemove(t *testing.T) {
	var h hookList[func() int]
	if h.get() != nil {
		t.Fatal("empty list is not nil")
	}
	r1 := h.add(func() int { return 1 })
	r2 := h.add(func() int { return 2 })
	if n := len(h.get()); n != 2 {
		t.Fatalf("len = %d, want 2", n)
	}
	r1()
	if got := h.get(); len(got) != 1 || got[0].fn() != 2 {
		t.Fatalf("after removing the first hook, got %d hooks", len(got))
	}
	r2()
	if h.get() != nil {
		t.Fatal("list is not nil after removing every hook")
	}
}
