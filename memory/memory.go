package memory

import (
	"errors"
)

var ErrGasOverflow = errors.New("memory gas overflow")

type Memory struct {
	data     []byte
	lastGas  uint64
}

func New() *Memory {
	return &Memory{data: make([]byte, 0, 4096)}
}

func (m *Memory) Reset() {
	m.data = m.data[:0]
	m.lastGas = 0
}

func (m *Memory) Len() int { return len(m.data) }

func (m *Memory) Data() []byte { return m.data }

func (m *Memory) ExpansionCost(offset, size uint64) (uint64, error) {
	if size == 0 {
		return 0, nil
	}
	newSize, overflow := safeAdd(offset, size)
	if overflow {
		return 0, ErrGasOverflow
	}
	newWords := (newSize + 31) / 32
	newGas := memoryGas(newWords)
	if newGas < m.lastGas {
		return 0, nil
	}
	return newGas - m.lastGas, nil
}

func (m *Memory) Expand(offset, size uint64, gasLeft *uint64) error {
	if size == 0 {
		return nil
	}
	cost, err := m.ExpansionCost(offset, size)
	if err != nil {
		return err
	}
	if cost > *gasLeft {
		return ErrGasOverflow
	}
	*gasLeft -= cost

	newSize := offset + size
	newWords := (newSize + 31) / 32
	m.lastGas = memoryGas(newWords)

	need := int(newWords * 32)
	if need > len(m.data) {
		if need > cap(m.data) {
			newCap := cap(m.data) * 2
			if newCap < need {
				newCap = need
			}
			buf := make([]byte, need, newCap)
			copy(buf, m.data)
			m.data = buf
		} else {
			m.data = m.data[:need]
		}
	}
	return nil
}

func (m *Memory) Set32(offset uint64, val []byte) {
	copy(m.data[offset:offset+32], val)
}

func (m *Memory) Set(offset uint64, val []byte) {
	copy(m.data[offset:], val)
}

func (m *Memory) SetByte(offset uint64, b byte) {
	m.data[offset] = b
}

func (m *Memory) Get(offset, size uint64) []byte {
	if size == 0 {
		return nil
	}
	return m.data[offset : offset+size]
}

func (m *Memory) GetCopy(offset, size uint64) []byte {
	if size == 0 {
		return nil
	}
	dst := make([]byte, size)
	copy(dst, m.data[offset:offset+size])
	return dst
}

func (m *Memory) GetPtr(offset, size uint64) []byte {
	if size == 0 {
		return nil
	}
	return m.data[offset : offset+size : offset+size]
}

func memoryGas(words uint64) uint64 {
	return 3*words + words*words/512
}

func safeAdd(a, b uint64) (uint64, bool) {
	c := a + b
	return c, c < a
}
