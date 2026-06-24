package memory

import (
	"math"
	"testing"
)

func TestNew(t *testing.T) {
	m := New()
	if m == nil {
		t.Fatal("New() returned nil")
	}
	if m.Len() != 0 {
		t.Fatalf("new memory should have len=0, got %d", m.Len())
	}
	if cap(m.data) != 4096 {
		t.Fatalf("new memory should have cap=4096, got %d", cap(m.data))
	}
}

func TestReset(t *testing.T) {
	m := New()
	gas := uint64(10000)
	_ = m.Expand(0, 64, &gas)
	if m.Len() != 64 {
		t.Fatalf("want len=64, got %d", m.Len())
	}
	m.Reset()
	if m.Len() != 0 {
		t.Fatalf("want len=0 after reset, got %d", m.Len())
	}
	if cap(m.data) < 64 {
		t.Fatalf("reset should keep backing allocation, cap=%d", cap(m.data))
	}
}

func TestLen(t *testing.T) {
	m := New()
	if m.Len() != 0 {
		t.Fatalf("want len=0, got %d", m.Len())
	}
	gas := uint64(10000)
	_ = m.Expand(0, 100, &gas)
	if m.Len() != 128 {
		t.Fatalf("want len=128, got %d", m.Len())
	}
}

func TestData(t *testing.T) {
	m := New()
	gas := uint64(10000)
	_ = m.Expand(0, 32, &gas)
	m.data[0] = 0xAB
	m.data[31] = 0xCD

	data := m.Data()
	if data[0] != 0xAB {
		t.Fatalf("want 0xAB, got 0x%X", data[0])
	}
	if data[31] != 0xCD {
		t.Fatalf("want 0xCD, got 0x%X", data[31])
	}
}

func TestExpand(t *testing.T) {
	m := New()
	gas := uint64(10000)
	if err := m.Expand(0, 32, &gas); err != nil {
		t.Fatal(err)
	}
	if m.Len() != 32 {
		t.Fatalf("want 32, got %d", m.Len())
	}
	before := gas
	if err := m.Expand(0, 32, &gas); err != nil {
		t.Fatal(err)
	}
	if gas != before {
		t.Fatalf("re-expand should cost 0 gas, spent %d", before-gas)
	}
}

func TestExpandRoundsUp(t *testing.T) {
	m := New()
	gas := uint64(10000)
	if err := m.Expand(0, 33, &gas); err != nil {
		t.Fatal(err)
	}
	if m.Len() != 64 {
		t.Fatalf("want 64, got %d", m.Len())
	}
}

func TestExpandZeroSize(t *testing.T) {
	m := New()
	gas := uint64(100)
	if err := m.Expand(0, 0, &gas); err != nil {
		t.Fatal(err)
	}
	if m.Len() != 0 {
		t.Fatalf("expand with size=0 should not grow memory, got %d", m.Len())
	}
	if gas != 100 {
		t.Fatalf("expand with size=0 should not consume gas, got %d", gas)
	}
}

func TestExpandCapacityGrowth(t *testing.T) {
	m := New()
	gas := uint64(1000000)
	_ = m.Expand(0, 8192, &gas)
	if m.Len() != 8192 {
		t.Fatalf("want len=8192, got %d", m.Len())
	}
	if cap(m.data) < 8192 {
		t.Fatalf("cap should be >= 8192, got %d", cap(m.data))
	}
}

func TestExpandLargeGrowth(t *testing.T) {
	m := New()
	gas := uint64(10000000)
	_ = m.Expand(0, 100000, &gas)
	if m.Len() != 100000 {
		t.Fatalf("want len=100000, got %d", m.Len())
	}
	if cap(m.data) < 100000 {
		t.Fatalf("cap should be >= 100000, got %d", cap(m.data))
	}
}

func TestExpandGasOverflow(t *testing.T) {
	m := New()
	gas := uint64(0)
	err := m.Expand(0, 32, &gas)
	if err == nil {
		t.Fatal("expected gas overflow error")
	}
}

func TestExpansionCost(t *testing.T) {
	m := New()
	cost, err := m.ExpansionCost(0, 32)
	if err != nil {
		t.Fatal(err)
	}
	if cost != 3 {
		t.Fatalf("want 3, got %d", cost)
	}
}

func TestExpansionCostZeroSize(t *testing.T) {
	m := New()
	cost, err := m.ExpansionCost(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if cost != 0 {
		t.Fatalf("want 0, got %d", cost)
	}
}

func TestExpansionCostOverflow(t *testing.T) {
	m := New()
	_, err := m.ExpansionCost(math.MaxUint64, 1)
	if err != ErrGasOverflow {
		t.Fatalf("want ErrGasOverflow, got %v", err)
	}
}

func TestExpandOverflow(t *testing.T) {
	m := New()
	gas := uint64(10000)
	err := m.Expand(math.MaxUint64, 1, &gas)
	if err != ErrGasOverflow {
		t.Fatalf("want ErrGasOverflow, got %v", err)
	}
}

func TestExpansionCostAlreadyExpanded(t *testing.T) {
	m := New()
	gas := uint64(10000)
	_ = m.Expand(0, 64, &gas)
	cost, err := m.ExpansionCost(0, 32)
	if err != nil {
		t.Fatal(err)
	}
	if cost != 0 {
		t.Fatalf("already expanded, want cost=0, got %d", cost)
	}
}

func TestSetGet(t *testing.T) {
	m := New()
	gas := uint64(10000)
	_ = m.Expand(0, 64, &gas)

	data := []byte{0xde, 0xad, 0xbe, 0xef}
	m.Set(4, data)
	got := m.Get(4, 4)
	for i, b := range data {
		if got[i] != b {
			t.Fatalf("byte %d: want %x, got %x", i, b, got[i])
		}
	}
}

func TestSet32(t *testing.T) {
	m := New()
	gas := uint64(10000)
	_ = m.Expand(0, 64, &gas)

	val := make([]byte, 32)
	val[0] = 0xAB
	val[31] = 0xCD
	m.Set32(0, val)

	got := m.Get(0, 32)
	if got[0] != 0xAB {
		t.Fatalf("want 0xAB, got 0x%X", got[0])
	}
	if got[31] != 0xCD {
		t.Fatalf("want 0xCD, got 0x%X", got[31])
	}
}

func TestSetByte(t *testing.T) {
	m := New()
	gas := uint64(10000)
	_ = m.Expand(0, 32, &gas)

	m.SetByte(10, 0x42)
	if m.data[10] != 0x42 {
		t.Fatalf("want 0x42, got 0x%X", m.data[10])
	}
}

func TestGetZeroSize(t *testing.T) {
	m := New()
	gas := uint64(10000)
	_ = m.Expand(0, 32, &gas)

	result := m.Get(0, 0)
	if result != nil {
		t.Fatalf("Get with size=0 should return nil, got %v", result)
	}
}

func TestGetCopy(t *testing.T) {
	m := New()
	gas := uint64(10000)
	_ = m.Expand(0, 32, &gas)
	m.data[0] = 0xAB

	got := m.GetCopy(0, 1)
	if got[0] != 0xAB {
		t.Fatalf("want 0xAB, got 0x%X", got[0])
	}
	got[0] = 0xFF
	if m.data[0] != 0xAB {
		t.Fatal("GetCopy should return a copy, not a slice into memory")
	}
}

func TestGetCopyZeroSize(t *testing.T) {
	m := New()
	gas := uint64(10000)
	_ = m.Expand(0, 32, &gas)

	result := m.GetCopy(0, 0)
	if result != nil {
		t.Fatalf("GetCopy with size=0 should return nil, got %v", result)
	}
}

func TestGetPtr(t *testing.T) {
	m := New()
	gas := uint64(10000)
	_ = m.Expand(0, 32, &gas)
	m.data[0] = 0xAB

	ptr := m.GetPtr(0, 1)
	if ptr[0] != 0xAB {
		t.Fatalf("want 0xAB, got 0x%X", ptr[0])
	}
	ptr[0] = 0xFF
	if m.data[0] != 0xFF {
		t.Fatal("GetPtr should return a slice into memory (zero-copy)")
	}
}

func TestGetPtrZeroSize(t *testing.T) {
	m := New()
	gas := uint64(10000)
	_ = m.Expand(0, 32, &gas)

	result := m.GetPtr(0, 0)
	if result != nil {
		t.Fatalf("GetPtr with size=0 should return nil, got %v", result)
	}
}

func TestMemoryGasFormula(t *testing.T) {
	// memoryGas(words) = 3*words + words*words/512
	tests := []struct {
		words uint64
		want  uint64
	}{
		{0, 0},
		{1, 3},
		{2, 6},
		{32, 96 + 2},  // 3*32 + 1024/512 = 96 + 2
		{256, 768 + 128}, // 3*256 + 65536/512 = 768 + 128
	}
	for _, tt := range tests {
		got := memoryGas(tt.words)
		if got != tt.want {
			t.Errorf("memoryGas(%d) = %d, want %d", tt.words, got, tt.want)
		}
	}
}

func TestSafeAdd(t *testing.T) {
	result, overflow := safeAdd(10, 20)
	if result != 30 {
		t.Fatalf("want 30, got %d", result)
	}
	if overflow {
		t.Fatal("expected no overflow")
	}
}

func TestSafeAddOverflow(t *testing.T) {
	result, overflow := safeAdd(math.MaxUint64, 1)
	if result != 0 {
		t.Fatalf("want 0, got %d", result)
	}
	if !overflow {
		t.Fatal("expected overflow")
	}
}

func TestSafeAddMaxValues(t *testing.T) {
	result, overflow := safeAdd(math.MaxUint64, math.MaxUint64)
	if !overflow {
		t.Fatal("expected overflow")
	}
	_ = result
}


