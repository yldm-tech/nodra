package core

import "testing"

func TestCombinePublishesOnceForAtomic2(t *testing.T) {
	first, second := New(0), New(0)
	combined := Combine(first, second, func(a, b int) int { return a + b }, func(a, b int) bool { return a == b })
	defer func() { _ = combined.Close() }()
	var values []int
	stop := combined.Subscribe(func(value int) { values = append(values, value) })
	defer stop()
	if err := Atomic2(first, second, func(a, b *int) error { *a, *b = 1, 2; return nil }); err != nil {
		t.Fatal(err)
	}
	if got := combined.Get(); got != 3 {
		t.Fatalf("combined = %d, want 3", got)
	}
	if len(values) != 1 || values[0] != 3 {
		t.Fatalf("notifications = %v, want [3]", values)
	}
}

func TestDiamondDerivedPublishesOnlyFinalAtomicValue(t *testing.T) {
	first, second := New(0), New(0)
	left := Combine(first, second, func(a, b int) int { return a + b }, func(a, b int) bool { return a == b })
	right := Combine(first, second, func(a, b int) int { return a * b }, func(a, b int) bool { return a == b })
	final := Combine(left, right, func(sum, product int) int { return sum*100 + product }, func(a, b int) bool { return a == b })
	defer func() { _ = final.Close(); _ = right.Close(); _ = left.Close() }()
	var values []int
	stop := final.Subscribe(func(value int) { values = append(values, value) })
	defer stop()
	if err := Atomic2(first, second, func(a, b *int) error { *a, *b = 1, 2; return nil }); err != nil {
		t.Fatal(err)
	}
	if got := final.Get(); got != 302 {
		t.Fatalf("final = %d, want 302", got)
	}
	if len(values) != 1 || values[0] != 302 {
		t.Fatalf("diamond notifications = %v, want [302]", values)
	}
}
