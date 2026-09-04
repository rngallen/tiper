package main

import (
	"sync"
	"testing"
)

func TestClampCopyPool(t *testing.T) {
	w, b := clampCopyPool(0, 0)
	if w != 1 || b != 1 {
		t.Fatalf("low clamp %d %d", w, b)
	}
	w, b = clampCopyPool(99, 5000)
	if w != 32 || b != 1000 {
		t.Fatalf("high clamp %d %d", w, b)
	}
}

func TestBatchWriter_FlushesOnSize(t *testing.T) {
	var mu sync.Mutex
	var got [][]int
	w := newBatchWriter("t", 2, 2, func(batch []int) {
		cp := append([]int(nil), batch...)
		mu.Lock()
		got = append(got, cp)
		mu.Unlock()
	})
	w.add(1)
	w.add(2)
	w.add(3)
	if n := w.close(); n != 3 {
		t.Fatalf("added %d", n)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 {
		t.Fatalf("batches %d", len(got))
	}
	if len(got[0])+len(got[1]) != 3 {
		t.Fatalf("items %v", got)
	}
}
