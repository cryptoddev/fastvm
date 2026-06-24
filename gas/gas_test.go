package gas

import "testing"

func TestCallGas(t *testing.T) {
	avail := uint64(64000)
	got := CallGas(avail)
	want := avail - avail/64
	if got != want {
		t.Fatalf("want %d, got %d", want, got)
	}
}

func TestCallGasZero(t *testing.T) {
	if got := CallGas(0); got != 0 {
		t.Fatalf("CallGas(0) = %d, want 0", got)
	}
}

func TestCallGasSmall(t *testing.T) {
	if got := CallGas(63); got != 63 {
		t.Fatalf("CallGas(63) = %d, want 63", got)
	}
}

func TestCopyWords(t *testing.T) {
	tests := []struct {
		n    uint64
		want uint64
	}{
		{0, 0},
		{1, 3},
		{32, 3},
		{33, 6},
		{64, 6},
	}
	for _, tt := range tests {
		got := CopyWords(tt.n)
		if got != tt.want {
			t.Errorf("CopyWords(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}
}

func TestSha3Words(t *testing.T) {
	tests := []struct {
		n    uint64
		want uint64
	}{
		{0, 0},
		{1, 6},
		{32, 6},
		{33, 12},
		{64, 12},
	}
	for _, tt := range tests {
		got := Sha3Words(tt.n)
		if got != tt.want {
			t.Errorf("Sha3Words(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}
}

func TestInitCodeCost(t *testing.T) {
	if got := InitCodeCost(0); got != 0 {
		t.Fatalf("InitCodeCost(0) = %d, want 0", got)
	}
	if got := InitCodeCost(1000); got != 0 {
		t.Fatalf("InitCodeCost(1000) = %d, want 0", got)
	}
}

func TestSstoreCostNoOp(t *testing.T) {
	var slot [32]byte
	slot[31] = 1
	cost, refund := SstoreCost(slot, slot, slot, true)
	if cost != SstoreNoopGas {
		t.Fatalf("want %d, got %d", SstoreNoopGas, cost)
	}
	if refund != 0 {
		t.Fatalf("want 0 refund, got %d", refund)
	}
}

func TestSstoreCostNoOpCold(t *testing.T) {
	var slot [32]byte
	slot[31] = 1
	cost, refund := SstoreCost(slot, slot, slot, false)
	if cost != ColdSload+SstoreNoopGas {
		t.Fatalf("want %d, got %d", ColdSload+SstoreNoopGas, cost)
	}
	if refund != 0 {
		t.Fatalf("want 0 refund, got %d", refund)
	}
}

func TestSstoreCostSet(t *testing.T) {
	var zero, one [32]byte
	one[31] = 1
	// Setting a zero slot to non-zero: SstoreSet
	cost, _ := SstoreCost(zero, zero, one, true)
	if cost != SstoreSet {
		t.Fatalf("want %d, got %d", SstoreSet, cost)
	}
}

func TestSstoreCostClear(t *testing.T) {
	var zero, one [32]byte
	one[31] = 1
	// Clearing a non-zero slot: SstoreReset + SstoreClear refund
	cost, refund := SstoreCost(one, one, zero, true)
	if cost != SstoreReset {
		t.Fatalf("cost: want %d, got %d", SstoreReset, cost)
	}
	if refund != SstoreClear {
		t.Fatalf("refund: want %d, got %d", SstoreClear, refund)
	}
}

func TestSstoreCostResetNoRefund(t *testing.T) {
	var one, two [32]byte
	one[31] = 1
	two[31] = 2
	// Changing non-zero to different non-zero: SstoreReset, no refund
	cost, refund := SstoreCost(one, one, two, true)
	if cost != SstoreReset {
		t.Fatalf("cost: want %d, got %d", SstoreReset, cost)
	}
	if refund != 0 {
		t.Fatalf("refund: want 0, got %d", refund)
	}
}

func TestSstoreCostDirtySetAgain(t *testing.T) {
	var zero, one, two [32]byte
	one[31] = 1
	two[31] = 2
	// original=0, current=1 (dirty), new=2
	// Was set from 0 to 1, now setting to 2
	cost, refund := SstoreCost(one, zero, two, true)
	if cost != SstoreNoopGas {
		t.Fatalf("cost: want %d, got %d", SstoreNoopGas, cost)
	}
	if refund != 0 {
		t.Fatalf("refund: want 0, got %d", refund)
	}
}

func TestSstoreCostDirtyRevertToOriginal(t *testing.T) {
	var zero, one [32]byte
	one[31] = 1
	// original=0, current=1 (dirty), new=0 (back to original)
	cost, refund := SstoreCost(one, zero, zero, true)
	if cost != SstoreNoopGas {
		t.Fatalf("cost: want %d, got %d", SstoreNoopGas, cost)
	}
	if refund != SstoreInitRefund {
		t.Fatalf("refund: want %d, got %d", SstoreInitRefund, refund)
	}
}

func TestSstoreCostDirtyClearRestoreOriginal(t *testing.T) {
	var one, zero [32]byte
	one[31] = 1
	// original=1, current=2 (dirty), new=0 (clearing)
	cost, refund := SstoreCost(two, one, zero, true)
	if cost != SstoreNoopGas {
		t.Fatalf("cost: want %d, got %d", SstoreNoopGas, cost)
	}
	if refund != SstoreClear {
		t.Fatalf("refund: want %d, got %d", SstoreClear, refund)
	}
}

func TestSstoreCostDirtyRestoreOriginal(t *testing.T) {
	var one, two [32]byte
	one[31] = 1
	two[31] = 2
	// original=1, current=2 (dirty), new=1 (back to original)
	cost, refund := SstoreCost(two, one, one, true)
	if cost != SstoreNoopGas {
		t.Fatalf("cost: want %d, got %d", SstoreNoopGas, cost)
	}
	if refund != SstoreCleanRefund {
		t.Fatalf("refund: want %d, got %d", SstoreCleanRefund, refund)
	}
}

func TestSstoreCostDirtyClearThenSetAgain(t *testing.T) {
	var zero, one [32]byte
	one[31] = 1
	// original=1, current=0 (was cleared), new=1 (set again)
	// This removes the clear refund
	cost, refund := SstoreCost(zero, one, one, true)
	if cost != SstoreNoopGas {
		t.Fatalf("cost: want %d, got %d", SstoreNoopGas, cost)
	}
	if refund != SstoreCleanRefund {
		t.Fatalf("refund: want %d, got %d", SstoreCleanRefund, refund)
	}
}

func TestMaxRefundQuotient(t *testing.T) {
	if MaxRefundQuotient != 5 {
		t.Fatalf("MaxRefundQuotient = %d, want 5", MaxRefundQuotient)
	}
}

func TestGasConstants(t *testing.T) {
	if Zero != 0 {
		t.Fatalf("Zero = %d, want 0", Zero)
	}
	if JumpDest != 1 {
		t.Fatalf("JumpDest = %d, want 1", JumpDest)
	}
	if Base != 2 {
		t.Fatalf("Base = %d, want 2", Base)
	}
	if VeryLow != 3 {
		t.Fatalf("VeryLow = %d, want 3", VeryLow)
	}
	if Low != 5 {
		t.Fatalf("Low = %d, want 5", Low)
	}
	if Mid != 8 {
		t.Fatalf("Mid = %d, want 8", Mid)
	}
	if High != 10 {
		t.Fatalf("High = %d, want 10", High)
	}
	if WarmStorageRead != 100 {
		t.Fatalf("WarmStorageRead = %d, want 100", WarmStorageRead)
	}
	if ColdAccountAccess != 2600 {
		t.Fatalf("ColdAccountAccess = %d, want 2600", ColdAccountAccess)
	}
	if ColdSload != 2100 {
		t.Fatalf("ColdSload = %d, want 2100", ColdSload)
	}
	if SstoreSet != 20000 {
		t.Fatalf("SstoreSet = %d, want 20000", SstoreSet)
	}
	if SstoreReset != 2900 {
		t.Fatalf("SstoreReset = %d, want 2900", SstoreReset)
	}
	if SstoreClear != 4800 {
		t.Fatalf("SstoreClear = %d, want 4800", SstoreClear)
	}
	if SstoreNoopGas != 100 {
		t.Fatalf("SstoreNoopGas = %d, want 100", SstoreNoopGas)
	}
	if SstoreInitGas != 20000 {
		t.Fatalf("SstoreInitGas = %d, want 20000", SstoreInitGas)
	}
	if SstoreInitRefund != 19900 {
		t.Fatalf("SstoreInitRefund = %d, want 19900", SstoreInitRefund)
	}
	if SstoreCleanRefund != 4800 {
		t.Fatalf("SstoreCleanRefund = %d, want 4800", SstoreCleanRefund)
	}
	if SstoreDirtyRefund != 0 {
		t.Fatalf("SstoreDirtyRefund = %d, want 0", SstoreDirtyRefund)
	}
	if Exp != 10 {
		t.Fatalf("Exp = %d, want 10", Exp)
	}
	if ExpByte != 50 {
		t.Fatalf("ExpByte = %d, want 50", ExpByte)
	}
	if Sha3 != 30 {
		t.Fatalf("Sha3 = %d, want 30", Sha3)
	}
	if Sha3Word != 6 {
		t.Fatalf("Sha3Word = %d, want 6", Sha3Word)
	}
	if Copy != 3 {
		t.Fatalf("Copy = %d, want 3", Copy)
	}
	if BlockHash != 20 {
		t.Fatalf("BlockHash = %d, want 20", BlockHash)
	}
	if Log != 375 {
		t.Fatalf("Log = %d, want 375", Log)
	}
	if LogData != 8 {
		t.Fatalf("LogData = %d, want 8", LogData)
	}
	if LogTopic != 375 {
		t.Fatalf("LogTopic = %d, want 375", LogTopic)
	}
	if Create != 32000 {
		t.Fatalf("Create = %d, want 32000", Create)
	}
	if Create2 != 32000 {
		t.Fatalf("Create2 = %d, want 32000", Create2)
	}
	if Create2Word != 6 {
		t.Fatalf("Create2Word = %d, want 6", Create2Word)
	}
	if CallValue != 9000 {
		t.Fatalf("CallValue = %d, want 9000", CallValue)
	}
	if CallStipend != 2300 {
		t.Fatalf("CallStipend = %d, want 2300", CallStipend)
	}
	if NewAccount != 25000 {
		t.Fatalf("NewAccount = %d, want 25000", NewAccount)
	}
	if SelfDestruct != 5000 {
		t.Fatalf("SelfDestruct = %d, want 5000", SelfDestruct)
	}
	if SelfDestructNewAccount != 25000 {
		t.Fatalf("SelfDestructNewAccount = %d, want 25000", SelfDestructNewAccount)
	}
	if TxDataZero != 4 {
		t.Fatalf("TxDataZero = %d, want 4", TxDataZero)
	}
	if TxDataNonZero != 16 {
		t.Fatalf("TxDataNonZero = %d, want 16", TxDataNonZero)
	}
	if TxBase != 21000 {
		t.Fatalf("TxBase = %d, want 21000", TxBase)
	}
	if TxCreate != 53000 {
		t.Fatalf("TxCreate = %d, want 53000", TxCreate)
	}
	if TxAccessListAddress != 2400 {
		t.Fatalf("TxAccessListAddress = %d, want 2400", TxAccessListAddress)
	}
	if TxAccessListStorage != 1900 {
		t.Fatalf("TxAccessListStorage = %d, want 1900", TxAccessListStorage)
	}
}

var two = [32]byte{31: 2}
