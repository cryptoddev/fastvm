package stack

import (
	"errors"

	"github.com/holiman/uint256"
)

const (
	MaxDepth = 1024
)

var (
	ErrStackOverflow  = errors.New("stack overflow")
	ErrStackUnderflow = errors.New("stack underflow")
)

type Stack struct {
	data [MaxDepth]uint256.Int
	top  int // index of next free slot (0 = empty)
}

func (s *Stack) Reset() {
	s.top = 0
}

func (s *Stack) Len() int { return s.top }

func (s *Stack) Push(v *uint256.Int) error {
	if s.top == MaxDepth {
		return ErrStackOverflow
	}
	s.data[s.top].Set(v)
	s.top++
	return nil
}

func (s *Stack) PushRaw(v uint64) error {
	if s.top == MaxDepth {
		return ErrStackOverflow
	}
	s.data[s.top].SetUint64(v)
	s.top++
	return nil
}

func (s *Stack) Pop() (*uint256.Int, error) {
	if s.top == 0 {
		return nil, ErrStackUnderflow
	}
	s.top--
	return &s.data[s.top], nil
}

func (s *Stack) Peek() (*uint256.Int, error) {
	if s.top == 0 {
		return nil, ErrStackUnderflow
	}
	return &s.data[s.top-1], nil
}

func (s *Stack) PeekN(n int) (*uint256.Int, error) {
	idx := s.top - 1 - n
	if idx < 0 {
		return nil, ErrStackUnderflow
	}
	return &s.data[idx], nil
}

func (s *Stack) SwapN(n int) error {
	// n is 1-based: SWAP1 swaps top with index top-2
	if s.top < n+1 {
		return ErrStackUnderflow
	}
	s.data[s.top-1], s.data[s.top-1-n] = s.data[s.top-1-n], s.data[s.top-1]
	return nil
}

func (s *Stack) DupN(n int) error {
	if s.top < n {
		return ErrStackUnderflow
	}
	if s.top == MaxDepth {
		return ErrStackOverflow
	}
	s.data[s.top].Set(&s.data[s.top-n])
	s.top++
	return nil
}

// Back is an alias for PeekN used by opcode implementations (no bounds check).
func (s *Stack) Back(n int) *uint256.Int {
	return &s.data[s.top-1-n]
}

func (s *Stack) RequireN(n int) error {
	if s.top < n {
		return ErrStackUnderflow
	}
	return nil
}

func (s *Stack) Data() []uint256.Int {
	return s.data[:s.top]
}
