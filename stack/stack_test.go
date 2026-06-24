package stack

import (
	"testing"

	"github.com/holiman/uint256"
)

func TestReset(t *testing.T) {
	var s Stack
	s.PushRaw(1)
	s.PushRaw(2)
	if s.Len() != 2 {
		t.Fatalf("want len=2, got %d", s.Len())
	}
	s.Reset()
	if s.Len() != 0 {
		t.Fatalf("want len=0 after reset, got %d", s.Len())
	}
}

func TestLen(t *testing.T) {
	var s Stack
	if s.Len() != 0 {
		t.Fatalf("want len=0, got %d", s.Len())
	}
	s.PushRaw(42)
	if s.Len() != 1 {
		t.Fatalf("want len=1, got %d", s.Len())
	}
}

func TestPushRaw(t *testing.T) {
	var s Stack
	if err := s.PushRaw(12345); err != nil {
		t.Fatal(err)
	}
	top, _ := s.Peek()
	if top.Uint64() != 12345 {
		t.Fatalf("want 12345, got %d", top.Uint64())
	}
}

func TestPushRawOverflow(t *testing.T) {
	var s Stack
	for i := 0; i < MaxDepth; i++ {
		if err := s.PushRaw(1); err != nil {
			t.Fatalf("unexpected overflow at %d", i)
		}
	}
	if err := s.PushRaw(1); err != ErrStackOverflow {
		t.Fatalf("want ErrStackOverflow, got %v", err)
	}
}

func TestPushOverflow(t *testing.T) {
	var s Stack
	v := uint256.NewInt(1)
	for i := 0; i < MaxDepth; i++ {
		if err := s.Push(v); err != nil {
			t.Fatalf("unexpected overflow at %d", i)
		}
	}
	if err := s.Push(v); err != ErrStackOverflow {
		t.Fatalf("want ErrStackOverflow, got %v", err)
	}
}

func TestPeek(t *testing.T) {
	var s Stack
	if _, err := s.Peek(); err != ErrStackUnderflow {
		t.Fatalf("want ErrStackUnderflow, got %v", err)
	}
	s.PushRaw(42)
	v, err := s.Peek()
	if err != nil {
		t.Fatal(err)
	}
	if v.Uint64() != 42 {
		t.Fatalf("want 42, got %d", v.Uint64())
	}
	if s.Len() != 1 {
		t.Fatalf("Peek should not change len, got %d", s.Len())
	}
}

func TestPeekN(t *testing.T) {
	var s Stack
	s.PushRaw(10)
	s.PushRaw(20)
	s.PushRaw(30)

	v0, err := s.PeekN(0)
	if err != nil {
		t.Fatal(err)
	}
	if v0.Uint64() != 30 {
		t.Fatalf("want 30, got %d", v0.Uint64())
	}

	v1, err := s.PeekN(1)
	if err != nil {
		t.Fatal(err)
	}
	if v1.Uint64() != 20 {
		t.Fatalf("want 20, got %d", v1.Uint64())
	}

	v2, err := s.PeekN(2)
	if err != nil {
		t.Fatal(err)
	}
	if v2.Uint64() != 10 {
		t.Fatalf("want 10, got %d", v2.Uint64())
	}

	_, err = s.PeekN(3)
	if err != ErrStackUnderflow {
		t.Fatalf("want ErrStackUnderflow, got %v", err)
	}

	_, err = s.PeekN(100)
	if err != ErrStackUnderflow {
		t.Fatalf("want ErrStackUnderflow for large n, got %v", err)
	}
}

func TestPeekNEmpty(t *testing.T) {
	var s Stack
	_, err := s.PeekN(0)
	if err != ErrStackUnderflow {
		t.Fatalf("want ErrStackUnderflow on empty stack, got %v", err)
	}
}

func TestSwapNUnderflow(t *testing.T) {
	var s Stack
	s.PushRaw(1)

	if err := s.SwapN(1); err != ErrStackUnderflow {
		t.Fatalf("want ErrStackUnderflow, got %v", err)
	}

	if err := s.SwapN(100); err != ErrStackUnderflow {
		t.Fatalf("want ErrStackUnderflow for large n, got %v", err)
	}
}

func TestSwapN(t *testing.T) {
	var s Stack
	s.PushRaw(1)
	s.PushRaw(2)
	s.PushRaw(3)

	if err := s.SwapN(1); err != nil {
		t.Fatal(err)
	}
	top, _ := s.Peek()
	if top.Uint64() != 2 {
		t.Fatalf("after SWAP1 want top=2, got %d", top.Uint64())
	}
}

func TestDupN(t *testing.T) {
	var s Stack
	s.PushRaw(10)
	s.PushRaw(20)
	if err := s.DupN(2); err != nil {
		t.Fatal(err)
	}
	if s.Len() != 3 {
		t.Fatalf("want 3, got %d", s.Len())
	}
	top, _ := s.Peek()
	if top.Uint64() != 10 {
		t.Fatalf("want 10, got %d", top.Uint64())
	}
}

func TestDupNUnderflow(t *testing.T) {
	var s Stack
	s.PushRaw(10)

	if err := s.DupN(2); err != ErrStackUnderflow {
		t.Fatalf("want ErrStackUnderflow, got %v", err)
	}
}

func TestDupNEmpty(t *testing.T) {
	var s Stack
	if err := s.DupN(1); err != ErrStackUnderflow {
		t.Fatalf("want ErrStackUnderflow on empty stack, got %v", err)
	}
}

func TestDupNOverflow(t *testing.T) {
	var s Stack
	v := uint256.NewInt(1)
	for i := 0; i < MaxDepth; i++ {
		if err := s.Push(v); err != nil {
			t.Fatalf("unexpected overflow at %d", i)
		}
	}
	if err := s.DupN(1); err != ErrStackOverflow {
		t.Fatalf("want ErrStackOverflow, got %v", err)
	}
}

func TestBack(t *testing.T) {
	var s Stack
	s.PushRaw(10)
	s.PushRaw(20)
	s.PushRaw(30)

	v0 := s.Back(0)
	if v0.Uint64() != 30 {
		t.Fatalf("want 30, got %d", v0.Uint64())
	}

	v1 := s.Back(1)
	if v1.Uint64() != 20 {
		t.Fatalf("want 20, got %d", v1.Uint64())
	}

	v2 := s.Back(2)
	if v2.Uint64() != 10 {
		t.Fatalf("want 10, got %d", v2.Uint64())
	}
}

func TestRequireN(t *testing.T) {
	var s Stack
	s.PushRaw(1)
	s.PushRaw(2)

	if err := s.RequireN(2); err != nil {
		t.Fatalf("want nil, got %v", err)
	}

	if err := s.RequireN(3); err != ErrStackUnderflow {
		t.Fatalf("want ErrStackUnderflow, got %v", err)
	}

	if err := s.RequireN(0); err != nil {
		t.Fatalf("RequireN(0) should always succeed, got %v", err)
	}
}

func TestData(t *testing.T) {
	var s Stack
	s.PushRaw(10)
	s.PushRaw(20)

	data := s.Data()
	if len(data) != 2 {
		t.Fatalf("want len=2, got %d", len(data))
	}
	if data[0].Uint64() != 10 {
		t.Fatalf("want data[0]=10, got %d", data[0].Uint64())
	}
	if data[1].Uint64() != 20 {
		t.Fatalf("want data[1]=20, got %d", data[1].Uint64())
	}
}

func TestDataEmpty(t *testing.T) {
	var s Stack
	data := s.Data()
	if len(data) != 0 {
		t.Fatalf("want len=0, got %d", len(data))
	}
}

func TestErrorMessages(t *testing.T) {
	if ErrStackOverflow.Error() != "stack overflow" {
		t.Fatalf("unexpected ErrStackOverflow message: %s", ErrStackOverflow.Error())
	}
	if ErrStackUnderflow.Error() != "stack underflow" {
		t.Fatalf("unexpected ErrStackUnderflow message: %s", ErrStackUnderflow.Error())
	}
}

func TestPop(t *testing.T) {
	var s Stack
	s.PushRaw(42)
	v, err := s.Pop()
	if err != nil {
		t.Fatal(err)
	}
	if v.Uint64() != 42 {
		t.Fatalf("want 42, got %d", v.Uint64())
	}
	if s.Len() != 0 {
		t.Fatalf("want len=0 after pop, got %d", s.Len())
	}
}

func TestPopUnderflow(t *testing.T) {
	var s Stack
	_, err := s.Pop()
	if err != ErrStackUnderflow {
		t.Fatalf("want ErrStackUnderflow, got %v", err)
	}
}

func TestPush(t *testing.T) {
	var s Stack
	v := uint256.NewInt(42)
	if err := s.Push(v); err != nil {
		t.Fatal(err)
	}
	if s.Len() != 1 {
		t.Fatalf("want len=1, got %d", s.Len())
	}
	top, _ := s.Peek()
	if !top.Eq(v) {
		t.Fatalf("want %s, got %s", v, top)
	}
}

func TestPushCopyIndependence(t *testing.T) {
	var s Stack
	v := uint256.NewInt(42)
	if err := s.Push(v); err != nil {
		t.Fatal(err)
	}
	v.SetUint64(999)
	top, _ := s.Peek()
	if top.Uint64() != 42 {
		t.Fatalf("Push should copy; modifying original should not affect stack, got %d", top.Uint64())
	}
}
