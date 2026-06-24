package state

import (
	"testing"

	"github.com/holiman/uint256"
)

func addr(b byte) Address {
	var a Address
	a[19] = b
	return a
}

func TestBalanceAddSub(t *testing.T) {
	s := New()
	a := addr(1)
	s.AddBalance(a, uint256.NewInt(1000))
	if s.GetBalance(a).Uint64() != 1000 {
		t.Fatal("balance mismatch")
	}
	s.SubBalance(a, uint256.NewInt(300))
	if s.GetBalance(a).Uint64() != 700 {
		t.Fatal("balance after sub mismatch")
	}
}

func TestSnapshotRevert(t *testing.T) {
	s := New()
	a := addr(1)
	s.AddBalance(a, uint256.NewInt(100))

	snap := s.Snapshot()
	s.AddBalance(a, uint256.NewInt(50))
	if s.GetBalance(a).Uint64() != 150 {
		t.Fatal("expected 150 after add")
	}

	s.RevertToSnapshot(snap)
	if s.GetBalance(a).Uint64() != 100 {
		t.Fatalf("expected 100 after revert, got %d", s.GetBalance(a).Uint64())
	}
}

func TestStorageSetGet(t *testing.T) {
	s := New()
	a := addr(2)
	var slot, val Hash
	slot[31] = 1
	val[31] = 42
	s.SetStorage(a, slot, val)
	got := s.GetStorage(a, slot)
	if got != val {
		t.Fatalf("storage mismatch: want %x, got %x", val, got)
	}
}

func TestStorageRevert(t *testing.T) {
	s := New()
	a := addr(3)
	var slot, val1, val2 Hash
	slot[31] = 1
	val1[31] = 10
	val2[31] = 20

	s.SetStorage(a, slot, val1)
	snap := s.Snapshot()
	s.SetStorage(a, slot, val2)
	s.RevertToSnapshot(snap)

	got := s.GetStorage(a, slot)
	if got != val1 {
		t.Fatalf("want val1, got %x", got)
	}
}

func TestAccessList(t *testing.T) {
	s := New()
	a := addr(4)
	if s.AddressWarm(a) {
		t.Fatal("should be cold initially")
	}
	s.WarmAddress(a)
	if !s.AddressWarm(a) {
		t.Fatal("should be warm after WarmAddress")
	}

	var slot Hash
	slot[31] = 5
	if s.SlotWarm(a, slot) {
		t.Fatal("slot should be cold")
	}
	s.WarmSlot(a, slot)
	if !s.SlotWarm(a, slot) {
		t.Fatal("slot should be warm")
	}
}

func TestSuicide(t *testing.T) {
	s := New()
	a := addr(5)
	s.CreateAccount(a)
	s.Suicide(a)
	if !s.HasSuicided(a) {
		t.Fatal("should be suicided")
	}
	s.Finalise()
	if s.Exists(a) {
		t.Fatal("should be deleted after finalise")
	}
}

func TestNonce(t *testing.T) {
	s := New()
	a := addr(6)
	s.SetNonce(a, 5)
	if s.GetNonce(a) != 5 {
		t.Fatal("nonce mismatch")
	}
	snap := s.Snapshot()
	s.SetNonce(a, 10)
	s.RevertToSnapshot(snap)
	if s.GetNonce(a) != 5 {
		t.Fatal("nonce after revert mismatch")
	}
}

func TestStorageClone(t *testing.T) {
	st := Storage{
		Hash{1}: Hash{2},
		Hash{3}: Hash{4},
	}
	clone := st.Clone()
	if len(clone) != len(st) {
		t.Fatalf("clone len mismatch: want %d, got %d", len(st), len(clone))
	}
	for k, v := range st {
		if clone[k] != v {
			t.Fatalf("clone value mismatch for key %x", k)
		}
	}
	// Verify deep copy
	clone[Hash{1}] = Hash{99}
	if st[Hash{1}] == (Hash{99}) {
		t.Fatal("clone should be a deep copy")
	}
}

func TestStorageCloneEmpty(t *testing.T) {
	st := Storage{}
	clone := st.Clone()
	if len(clone) != 0 {
		t.Fatalf("clone of empty storage should be empty")
	}
}

func TestEmptyNonExistent(t *testing.T) {
	s := New()
	a := addr(99)
	if !s.Empty(a) {
		t.Fatal("non-existent account should be empty")
	}
}

func TestEmptyWithBalance(t *testing.T) {
	s := New()
	a := addr(10)
	s.SetBalance(a, uint256.NewInt(100))
	if s.Empty(a) {
		t.Fatal("account with balance should not be empty")
	}
}

func TestEmptyWithNonce(t *testing.T) {
	s := New()
	a := addr(11)
	s.SetNonce(a, 1)
	if s.Empty(a) {
		t.Fatal("account with nonce should not be empty")
	}
}

func TestEmptyWithCode(t *testing.T) {
	s := New()
	a := addr(12)
	s.SetCode(a, []byte{0x01}, Hash{1})
	if s.Empty(a) {
		t.Fatal("account with code should not be empty")
	}
}

func TestEmptyTrulyEmpty(t *testing.T) {
	s := New()
	a := addr(13)
	s.CreateAccount(a)
	if !s.Empty(a) {
		t.Fatal("created account with no balance/nonce/code should be empty")
	}
}

func TestGetBalanceNonExistent(t *testing.T) {
	s := New()
	a := addr(99)
	bal := s.GetBalance(a)
	if !bal.IsZero() {
		t.Fatalf("non-existent account balance should be 0, got %s", bal)
	}
}

func TestSetBalance(t *testing.T) {
	s := New()
	a := addr(20)
	val := uint256.NewInt(500)
	s.SetBalance(a, val)
	got := s.GetBalance(a)
	if !got.Eq(val) {
		t.Fatalf("want %s, got %s", val, got)
	}
}

func TestSetBalanceRevert(t *testing.T) {
	s := New()
	a := addr(21)
	s.SetBalance(a, uint256.NewInt(100))
	snap := s.Snapshot()
	s.SetBalance(a, uint256.NewInt(200))
	s.RevertToSnapshot(snap)
	if s.GetBalance(a).Uint64() != 100 {
		t.Fatalf("want 100 after revert, got %d", s.GetBalance(a).Uint64())
	}
}

func TestGetCodeNonExistent(t *testing.T) {
	s := New()
	a := addr(99)
	code := s.GetCode(a)
	if code != nil {
		t.Fatalf("non-existent account code should be nil, got %v", code)
	}
}

func TestGetCodeHashNonExistent(t *testing.T) {
	s := New()
	a := addr(99)
	h := s.GetCodeHash(a)
	if h != (Hash{}) {
		t.Fatalf("non-existent account code hash should be zero, got %x", h)
	}
}

func TestCreateAccountExistingWithBalance(t *testing.T) {
	s := New()
	a := addr(30)
	s.SetBalance(a, uint256.NewInt(500))
	s.CreateAccount(a)
	// Balance should be preserved
	if s.GetBalance(a).Uint64() != 500 {
		t.Fatalf("CreateAccount should preserve balance, got %d", s.GetBalance(a).Uint64())
	}
}

func TestCreateAccount(t *testing.T) {
	s := New()
	a := addr(31)
	s.CreateAccount(a)
	if !s.Exists(a) {
		t.Fatal("account should exist after CreateAccount")
	}
	if !s.Empty(a) {
		t.Fatal("newly created account should be empty")
	}
}

func TestCreatedInTx(t *testing.T) {
	s := New()
	a := addr(32)
	if s.CreatedInTx(a) {
		t.Fatal("account should not be marked as created yet")
	}
	s.CreateAccount(a)
	if !s.CreatedInTx(a) {
		t.Fatal("account should be marked as created")
	}
}

func TestCreatedInTxClearedAfterFinalise(t *testing.T) {
	s := New()
	a := addr(33)
	s.CreateAccount(a)
	s.Finalise()
	if s.CreatedInTx(a) {
		t.Fatal("createdInTx should be cleared after Finalise")
	}
}

func TestSetStorageDirect(t *testing.T) {
	s := New()
	a := addr(40)
	var slot, val Hash
	slot[31] = 1
	val[31] = 42
	s.SetStorageDirect(a, slot, val)
	got := s.GetStorage(a, slot)
	if got != val {
		t.Fatalf("want %x, got %x", val, got)
	}
	// Verify origStorage is also set
	orig := s.GetCommittedStorage(a, slot)
	if orig != val {
		t.Fatalf("origStorage should also be set, want %x, got %x", val, orig)
	}
}

func TestSetStorageDirectNoOverwriteOrig(t *testing.T) {
	s := New()
	a := addr(41)
	var slot, val1, val2 Hash
	slot[31] = 1
	val1[31] = 10
	val2[31] = 20
	s.SetStorageDirect(a, slot, val1)
	s.SetStorageDirect(a, slot, val2)
	// Current should be val2
	if s.GetStorage(a, slot) != val2 {
		t.Fatalf("current storage should be val2")
	}
	// Orig should still be val1 (first write)
	if s.GetCommittedStorage(a, slot) != val1 {
		t.Fatalf("orig storage should be val1")
	}
}

func TestSetStorageDeleteZero(t *testing.T) {
	s := New()
	a := addr(42)
	var slot, val Hash
	slot[31] = 1
	val[31] = 42
	s.SetStorage(a, slot, val)
	if s.GetStorage(a, slot) != val {
		t.Fatal("storage should be set")
	}
	// Set to zero should delete
	var zero Hash
	s.SetStorage(a, slot, zero)
	// Check that the slot is deleted from the map
	st := s.storage[a]
	if _, ok := st[slot]; ok {
		t.Fatal("slot should be deleted when set to zero")
	}
}

func TestRevertToSnapshotInvalidID(t *testing.T) {
	s := New()
	a := addr(50)
	s.SetBalance(a, uint256.NewInt(100))
	// Revert to a non-existent snapshot ID
	s.RevertToSnapshot(999)
	// Should not panic and balance should remain
	if s.GetBalance(a).Uint64() != 100 {
		t.Fatalf("balance should remain after invalid revert, got %d", s.GetBalance(a).Uint64())
	}
}

func TestRevertToSnapshotCodeRevert(t *testing.T) {
	s := New()
	a := addr(51)
	code1 := []byte{0x01, 0x02}
	hash1 := Hash{1}
	s.SetCode(a, code1, hash1)

	snap := s.Snapshot()

	code2 := []byte{0x03, 0x04, 0x05}
	hash2 := Hash{2}
	s.SetCode(a, code2, hash2)

	s.RevertToSnapshot(snap)

	got := s.GetCode(a)
	if string(got) != string(code1) {
		t.Fatalf("code should revert, want %x, got %x", code1, got)
	}
	if s.GetCodeHash(a) != hash1 {
		t.Fatalf("code hash should revert")
	}
}

func TestRevertToSnapshotSuicideRevert(t *testing.T) {
	s := New()
	a := addr(52)
	s.CreateAccount(a)

	snap := s.Snapshot()
	s.Suicide(a)
	if !s.HasSuicided(a) {
		t.Fatal("should be suicided")
	}

	s.RevertToSnapshot(snap)
	if s.HasSuicided(a) {
		t.Fatal("suicide should be reverted")
	}
}

func TestRevertToSnapshotCreateRevert(t *testing.T) {
	s := New()
	a := addr(53)
	s.CreateAccount(a)
	if !s.Exists(a) {
		t.Fatal("account should exist")
	}

	snap := s.Snapshot()
	s.CreateAccount(a)

	s.RevertToSnapshot(snap)
	if s.Exists(a) {
		t.Fatal("account should not exist after reverting creation")
	}
	if s.CreatedInTx(a) {
		t.Fatal("createdInTx should be cleared after reverting creation")
	}
}

func TestRevertToSnapshotStorageRevert(t *testing.T) {
	s := New()
	a := addr(54)
	var slot, val1, val2 Hash
	slot[31] = 1
	val1[31] = 10
	val2[31] = 20

	s.SetStorage(a, slot, val1)
	snap := s.Snapshot()
	s.SetStorage(a, slot, val2)

	s.RevertToSnapshot(snap)
	if s.GetStorage(a, slot) != val1 {
		t.Fatalf("storage should revert to val1")
	}
}

func TestRevertToSnapshotStorageDeleteRevert(t *testing.T) {
	s := New()
	a := addr(55)
	var slot, val Hash
	slot[31] = 1
	val[31] = 42

	s.SetStorage(a, slot, val)
	snap := s.Snapshot()
	var zero Hash
	s.SetStorage(a, slot, zero)

	s.RevertToSnapshot(snap)
	if s.GetStorage(a, slot) != val {
		t.Fatalf("storage should revert to val after delete revert")
	}
}

func TestRevertToSnapshotStorageNewSlotRevert(t *testing.T) {
	s := New()
	a := addr(56)
	var slot, val Hash
	slot[31] = 1
	val[31] = 42

	snap := s.Snapshot()
	// Set a new slot (oldVal is zero)
	s.SetStorage(a, slot, val)

	s.RevertToSnapshot(snap)
	// Slot should be deleted after revert
	st := s.storage[a]
	if st == nil {
		return // storage map doesn't exist
	}
	if _, ok := st[slot]; ok {
		t.Fatal("new slot should be deleted after revert")
	}
}

func TestSubRefundExceedsRefund(t *testing.T) {
	s := New()
	s.AddRefund(100)
	s.SubRefund(200)
	if s.GetRefund() != 0 {
		t.Fatalf("refund should be 0 after exceeding, got %d", s.GetRefund())
	}
}

func TestSubRefundExact(t *testing.T) {
	s := New()
	s.AddRefund(100)
	s.SubRefund(100)
	if s.GetRefund() != 0 {
		t.Fatalf("refund should be 0 after exact sub, got %d", s.GetRefund())
	}
}

func TestGetRefund(t *testing.T) {
	s := New()
	if s.GetRefund() != 0 {
		t.Fatalf("initial refund should be 0, got %d", s.GetRefund())
	}
	s.AddRefund(500)
	if s.GetRefund() != 500 {
		t.Fatalf("want 500, got %d", s.GetRefund())
	}
}

func TestFinaliseWithNoSuicides(t *testing.T) {
	s := New()
	a := addr(60)
	s.SetBalance(a, uint256.NewInt(100))
	s.Finalise()
	// Account should still exist (not suicided)
	if !s.Exists(a) {
		t.Fatal("non-suicided account should still exist after Finalise")
	}
	if s.GetBalance(a).Uint64() != 100 {
		t.Fatalf("balance should remain, got %d", s.GetBalance(a).Uint64())
	}
}

func TestFinaliseClearsJournal(t *testing.T) {
	s := New()
	a := addr(61)
	s.SetBalance(a, uint256.NewInt(100))
	s.Finalise()
	if len(s.journal) != 0 {
		t.Fatal("journal should be cleared after Finalise")
	}
	if len(s.snapshots) != 0 {
		t.Fatal("snapshots should be cleared after Finalise")
	}
}

func TestReset(t *testing.T) {
	s := New()
	a := addr(70)
	s.SetBalance(a, uint256.NewInt(100))
	s.AddRefund(50)
	s.WarmAddress(a)

	var slot, val Hash
	slot[31] = 1
	val[31] = 42
	s.SetStorage(a, slot, val)

	s.SetNonce(a, 5)

	// Warm a slot to test accessedSlots cleanup
	var slot2 Hash
	slot2[31] = 2
	s.WarmSlot(a, slot2)

	l := &Log{Address: a}
	s.AddLog(l)

	s.Reset()

	if s.Exists(a) {
		t.Fatal("account should not exist after Reset")
	}
	if s.GetRefund() != 0 {
		t.Fatal("refund should be 0 after Reset")
	}
	if s.AddressWarm(a) {
		t.Fatal("address should not be warm after Reset")
	}
	if len(s.Logs) != 0 {
		t.Fatal("Logs should be empty after Reset")
	}
	if len(s.journal) != 0 {
		t.Fatal("journal should be empty after Reset")
	}
	if len(s.snapshots) != 0 {
		t.Fatal("snapshots should be empty after Reset")
	}
	if s.SlotWarm(a, slot2) {
		t.Fatal("slot should not be warm after Reset")
	}
}

func TestAddLog(t *testing.T) {
	s := New()
	l := &Log{
		Address: addr(80),
		Topics:  []Hash{{1}, {2}},
		Data:    []byte{0xAB, 0xCD},
	}
	s.AddLog(l)
	if len(s.Logs) != 1 {
		t.Fatalf("want 1 log, got %d", len(s.Logs))
	}
	if s.Logs[0].Address != addr(80) {
		t.Fatal("log address mismatch")
	}
	if len(s.Logs[0].Topics) != 2 {
		t.Fatal("log topics mismatch")
	}
	if string(s.Logs[0].Data) != string([]byte{0xAB, 0xCD}) {
		t.Fatal("log data mismatch")
	}
}

func TestLogStruct(t *testing.T) {
	l := Log{
		Address: addr(90),
		Topics:  []Hash{{3}},
		Data:    []byte{0x01},
	}
	if l.Address != addr(90) {
		t.Fatal("log address mismatch")
	}
	if len(l.Topics) != 1 {
		t.Fatal("log topics length mismatch")
	}
	if len(l.Data) != 1 {
		t.Fatal("log data length mismatch")
	}
}

func TestExists(t *testing.T) {
	s := New()
	a := addr(85)
	if s.Exists(a) {
		t.Fatal("non-existent account should not exist")
	}
	s.CreateAccount(a)
	if !s.Exists(a) {
		t.Fatal("created account should exist")
	}
}

func TestGetNonceNonExistent(t *testing.T) {
	s := New()
	a := addr(86)
	if s.GetNonce(a) != 0 {
		t.Fatal("non-existent account nonce should be 0")
	}
}

func TestGetStorageNonExistent(t *testing.T) {
	s := New()
	a := addr(87)
	var slot Hash
	slot[31] = 1
	val := s.GetStorage(a, slot)
	if val != (Hash{}) {
		t.Fatalf("non-existent account storage should be zero, got %x", val)
	}
}

func TestGetCommittedStorageNonExistent(t *testing.T) {
	s := New()
	a := addr(88)
	var slot Hash
	slot[31] = 1
	val := s.GetCommittedStorage(a, slot)
	if val != (Hash{}) {
		t.Fatalf("non-existent account committed storage should be zero, got %x", val)
	}
}

func TestSnapshotIDSequence(t *testing.T) {
	s := New()
	id1 := s.Snapshot()
	id2 := s.Snapshot()
	if id1 != 0 {
		t.Fatalf("first snapshot id should be 0, got %d", id1)
	}
	if id2 != 1 {
		t.Fatalf("second snapshot id should be 1, got %d", id2)
	}
}

func TestRevertToSnapshotClearsLaterSnapshots(t *testing.T) {
	s := New()
	a := addr(89)
	s.SetBalance(a, uint256.NewInt(100))

	id1 := s.Snapshot()
	s.SetBalance(a, uint256.NewInt(200))
	id2 := s.Snapshot()
	s.SetBalance(a, uint256.NewInt(300))

	s.RevertToSnapshot(id1)

	// id2 snapshot should be gone
	s.RevertToSnapshot(id2)
	// Balance should still be 100 (from id1 revert)
	if s.GetBalance(a).Uint64() != 100 {
		t.Fatalf("balance should be 100, got %d", s.GetBalance(a).Uint64())
	}
}

func TestSuicideRevert(t *testing.T) {
	s := New()
	a := addr(91)
	s.CreateAccount(a)

	snap := s.Snapshot()
	s.Suicide(a)
	s.RevertToSnapshot(snap)

	if s.HasSuicided(a) {
		t.Fatal("suicide should be reverted")
	}
}

func TestFinaliseRemovesStorage(t *testing.T) {
	s := New()
	a := addr(92)
	s.CreateAccount(a)
	var slot, val Hash
	slot[31] = 1
	val[31] = 42
	s.SetStorage(a, slot, val)
	s.Suicide(a)
	s.Finalise()

	if s.Exists(a) {
		t.Fatal("suicided account should be removed")
	}
	// Storage should also be removed
	st := s.storage[a]
	if st != nil {
		t.Fatal("storage should be removed for suicided account")
	}
}

func TestSetAccount(t *testing.T) {
	s := New()
	a := addr(93)
	acc := Account{
		Balance:  *uint256.NewInt(500),
		Nonce:    10,
		Code:     []byte{0x01, 0x02},
		CodeHash: Hash{1},
	}
	s.SetAccount(a, acc)

	if s.GetBalance(a).Uint64() != 500 {
		t.Fatalf("want balance=500, got %d", s.GetBalance(a).Uint64())
	}
	if s.GetNonce(a) != 10 {
		t.Fatalf("want nonce=10, got %d", s.GetNonce(a))
	}
	if string(s.GetCode(a)) != string([]byte{0x01, 0x02}) {
		t.Fatalf("code mismatch")
	}
	if s.GetCodeHash(a) != (Hash{1}) {
		t.Fatalf("code hash mismatch")
	}
}

func TestRevertNonceNonExistentAccount(t *testing.T) {
	s := New()
	a := addr(94)
	s.SetNonce(a, 5)
	snap := s.Snapshot()
	s.SetNonce(a, 10)
	s.RevertToSnapshot(snap)
	if s.GetNonce(a) != 5 {
		t.Fatalf("nonce should be reverted to 5, got %d", s.GetNonce(a))
	}
}

func TestRevertCodeNonExistentAccount(t *testing.T) {
	s := New()
	a := addr(95)
	s.SetCode(a, []byte{0x01}, Hash{1})
	snap := s.Snapshot()
	s.SetCode(a, []byte{0x02, 0x03}, Hash{2})
	s.RevertToSnapshot(snap)
	got := s.GetCode(a)
	if string(got) != string([]byte{0x01}) {
		t.Fatalf("code should be reverted, got %x", got)
	}
}

func TestRevertBalanceNonExistentAccount(t *testing.T) {
	s := New()
	a := addr(96)
	s.SetBalance(a, uint256.NewInt(100))
	snap := s.Snapshot()
	s.SetBalance(a, uint256.NewInt(200))
	s.RevertToSnapshot(snap)
	if s.GetBalance(a).Uint64() != 100 {
		t.Fatalf("balance should be reverted to 100, got %d", s.GetBalance(a).Uint64())
	}
}

func TestRevertSuicideNonExistentAccount(t *testing.T) {
	s := New()
	a := addr(97)
	s.CreateAccount(a)
	s.Suicide(a)
	snap := s.Snapshot()
	s.Suicide(a) // Already suicided, just testing revert doesn't panic
	s.RevertToSnapshot(snap)
	if !s.HasSuicided(a) {
		t.Fatal("should still be suicided after revert (was suicided before snapshot)")
	}
}
