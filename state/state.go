package state

import (
	"github.com/holiman/uint256"
)

type Address = [20]byte

type Hash = [32]byte

type Account struct {
	Balance  uint256.Int
	Nonce    uint64
	CodeHash Hash
	Code     []byte
	Suicide  bool
}

type Storage map[Hash]Hash

func (s Storage) Clone() Storage {
	c := make(Storage, len(s))
	for k, v := range s {
		c[k] = v
	}
	return c
}

type journalEntry struct {
	kind     uint8
	addr     Address
	slot     Hash
	oldVal   Hash
	oldBal   uint256.Int
	oldNonce uint64
	oldCode  []byte
	oldCodeHash Hash
	oldSuicide  bool
}

const (
	jBalance uint8 = iota
	jNonce
	jCode
	jStorage
	jCreate
	jSuicide
)

type StateDB struct {
	accounts     map[Address]*Account
	storage      map[Address]Storage
	origStorage  map[Address]Storage
	accessedAddrs  map[Address]struct{}
	accessedSlots  map[Address]map[Hash]struct{}
	journal      []journalEntry
	snapshots    []int
	Logs         []*Log
	Refund       uint64
	createdInTx  map[Address]struct{}
}

type Log struct {
	Address Address
	Topics  []Hash
	Data    []byte
}

func New() *StateDB {
	return &StateDB{
		accounts:      make(map[Address]*Account),
		storage:       make(map[Address]Storage),
		origStorage:   make(map[Address]Storage),
		accessedAddrs: make(map[Address]struct{}),
		accessedSlots: make(map[Address]map[Hash]struct{}),
	}
}

func (s *StateDB) Reset() {
	for k := range s.accounts {
		delete(s.accounts, k)
	}
	for k := range s.storage {
		delete(s.storage, k)
	}
	for k := range s.origStorage {
		delete(s.origStorage, k)
	}
	for k := range s.accessedAddrs {
		delete(s.accessedAddrs, k)
	}
	for k := range s.accessedSlots {
		delete(s.accessedSlots, k)
	}
	s.journal = s.journal[:0]
	s.snapshots = s.snapshots[:0]
	s.Logs = s.Logs[:0]
	s.Refund = 0
}

func (s *StateDB) getOrCreate(addr Address) *Account {
	a, ok := s.accounts[addr]
	if !ok {
		a = &Account{}
		s.accounts[addr] = a
	}
	return a
}

func (s *StateDB) Exists(addr Address) bool {
	_, ok := s.accounts[addr]
	return ok
}

func (s *StateDB) Empty(addr Address) bool {
	a, ok := s.accounts[addr]
	if !ok {
		return true
	}
	return a.Balance.IsZero() && a.Nonce == 0 && len(a.Code) == 0
}

func (s *StateDB) GetBalance(addr Address) *uint256.Int {
	a, ok := s.accounts[addr]
	if !ok {
		return new(uint256.Int)
	}
	result := new(uint256.Int)
	result.Set(&a.Balance)
	return result
}

func (s *StateDB) SetBalance(addr Address, val *uint256.Int) {
	a := s.getOrCreate(addr)
	s.journal = append(s.journal, journalEntry{kind: jBalance, addr: addr, oldBal: a.Balance})
	a.Balance.Set(val)
}

func (s *StateDB) AddBalance(addr Address, amount *uint256.Int) {
	a := s.getOrCreate(addr)
	s.journal = append(s.journal, journalEntry{kind: jBalance, addr: addr, oldBal: a.Balance})
	a.Balance.Add(&a.Balance, amount)
}

func (s *StateDB) SubBalance(addr Address, amount *uint256.Int) {
	a := s.getOrCreate(addr)
	s.journal = append(s.journal, journalEntry{kind: jBalance, addr: addr, oldBal: a.Balance})
	a.Balance.Sub(&a.Balance, amount)
}

func (s *StateDB) GetNonce(addr Address) uint64 {
	a, ok := s.accounts[addr]
	if !ok {
		return 0
	}
	return a.Nonce
}

func (s *StateDB) SetNonce(addr Address, nonce uint64) {
	a := s.getOrCreate(addr)
	s.journal = append(s.journal, journalEntry{kind: jNonce, addr: addr, oldNonce: a.Nonce})
	a.Nonce = nonce
}

func (s *StateDB) GetCode(addr Address) []byte {
	a, ok := s.accounts[addr]
	if !ok {
		return nil
	}
	return a.Code
}

func (s *StateDB) GetCodeHash(addr Address) Hash {
	a, ok := s.accounts[addr]
	if !ok {
		return Hash{}
	}
	return a.CodeHash
}

func (s *StateDB) SetCode(addr Address, code []byte, codeHash Hash) {
	a := s.getOrCreate(addr)
	s.journal = append(s.journal, journalEntry{
		kind: jCode, addr: addr,
		oldCode: a.Code, oldCodeHash: a.CodeHash,
	})
	a.Code = code
	a.CodeHash = codeHash
}

func (s *StateDB) CreateAccount(addr Address) {
	prev, exists := s.accounts[addr]
	newAcc := &Account{}
	if exists {
		newAcc.Balance.Set(&prev.Balance)
	}
	s.journal = append(s.journal, journalEntry{kind: jCreate, addr: addr})
	s.accounts[addr] = newAcc
	if s.createdInTx == nil {
		s.createdInTx = make(map[Address]struct{})
	}
	s.createdInTx[addr] = struct{}{}
}

func (s *StateDB) CreatedInTx(addr Address) bool {
	_, ok := s.createdInTx[addr]
	return ok
}

func (s *StateDB) storageOf(addr Address) Storage {
	st, ok := s.storage[addr]
	if !ok {
		st = make(Storage)
		s.storage[addr] = st
	}
	return st
}

func (s *StateDB) origStorageOf(addr Address) Storage {
	st, ok := s.origStorage[addr]
	if !ok {
		st = make(Storage)
		s.origStorage[addr] = st
	}
	return st
}

func (s *StateDB) GetStorage(addr Address, slot Hash) Hash {
	st, ok := s.storage[addr]
	if !ok {
		return Hash{}
	}
	return st[slot]
}

func (s *StateDB) GetCommittedStorage(addr Address, slot Hash) Hash {
	return s.origStorageOf(addr)[slot]
}

func (s *StateDB) SetStorage(addr Address, slot, value Hash) {
	st := s.storageOf(addr)
	old := st[slot]
	orig := s.origStorageOf(addr)
	if _, seen := orig[slot]; !seen {
		orig[slot] = old
	}
	s.journal = append(s.journal, journalEntry{kind: jStorage, addr: addr, slot: slot, oldVal: old})
	if value == (Hash{}) {
		delete(st, slot)
	} else {
		st[slot] = value
	}
}

func (s *StateDB) SetStorageDirect(addr Address, slot, value Hash) {
	st := s.storageOf(addr)
	orig := s.origStorageOf(addr)
	if _, seen := orig[slot]; !seen {
		orig[slot] = value
	}
	st[slot] = value
}

func (s *StateDB) AddressWarm(addr Address) bool {
	_, ok := s.accessedAddrs[addr]
	return ok
}

func (s *StateDB) WarmAddress(addr Address) {
	s.accessedAddrs[addr] = struct{}{}
}

func (s *StateDB) SlotWarm(addr Address, slot Hash) bool {
	m, ok := s.accessedSlots[addr]
	if !ok {
		return false
	}
	_, ok = m[slot]
	return ok
}

func (s *StateDB) WarmSlot(addr Address, slot Hash) {
	m, ok := s.accessedSlots[addr]
	if !ok {
		m = make(map[Hash]struct{})
		s.accessedSlots[addr] = m
	}
	m[slot] = struct{}{}
}

func (s *StateDB) Snapshot() int {
	id := len(s.snapshots)
	s.snapshots = append(s.snapshots, len(s.journal))
	return id
}

func (s *StateDB) RevertToSnapshot(id int) {
	if id >= len(s.snapshots) {
		return
	}
	journalLen := s.snapshots[id]
	for i := len(s.journal) - 1; i >= journalLen; i-- {
		e := s.journal[i]
		switch e.kind {
		case jBalance:
			if a := s.accounts[e.addr]; a != nil {
				a.Balance.Set(&e.oldBal)
			}
		case jNonce:
			if a := s.accounts[e.addr]; a != nil {
				a.Nonce = e.oldNonce
			}
		case jCode:
			if a := s.accounts[e.addr]; a != nil {
				a.Code = e.oldCode
				a.CodeHash = e.oldCodeHash
			}
		case jStorage:
			st := s.storageOf(e.addr)
			if e.oldVal == (Hash{}) {
				delete(st, e.slot)
			} else {
				st[e.slot] = e.oldVal
			}
		case jCreate:
			delete(s.accounts, e.addr)
			delete(s.createdInTx, e.addr)
		case jSuicide:
			if a := s.accounts[e.addr]; a != nil {
				a.Suicide = e.oldSuicide
			}
		}
	}
	s.journal = s.journal[:journalLen]
	s.snapshots = s.snapshots[:id]
}

func (s *StateDB) Suicide(addr Address) {
	a := s.getOrCreate(addr)
	s.journal = append(s.journal, journalEntry{kind: jSuicide, addr: addr, oldSuicide: a.Suicide})
	a.Suicide = true
}

func (s *StateDB) HasSuicided(addr Address) bool {
	a, ok := s.accounts[addr]
	return ok && a.Suicide
}

func (s *StateDB) Finalise() {
	for addr, acc := range s.accounts {
		if acc.Suicide {
			delete(s.accounts, addr)
			delete(s.storage, addr)
		}
	}
	s.journal = s.journal[:0]
	s.snapshots = s.snapshots[:0]
	for k := range s.createdInTx {
		delete(s.createdInTx, k)
	}
}

func (s *StateDB) AddLog(l *Log) {
	s.Logs = append(s.Logs, l)
}

func (s *StateDB) AddRefund(gas uint64) {
	s.Refund += gas
}

func (s *StateDB) SubRefund(gas uint64) {
	if gas > s.Refund {
		s.Refund = 0
		return
	}
	s.Refund -= gas
}

func (s *StateDB) GetRefund() uint64 {
	return s.Refund
}

func (s *StateDB) SetAccount(addr Address, acc Account) {
	a := s.getOrCreate(addr)
	*a = acc
}
