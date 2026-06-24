package evm

import (
	"context"

	"github.com/cryptoddev/fastvm/forkdb"
	"github.com/cryptoddev/fastvm/state"
	"github.com/holiman/uint256"
)

type forkDB struct {
	f *forkdb.ForkDB
	s *state.StateDB
}

func newForkDB(rpcURL string, blockNumber uint64) *forkDB {
	f := forkdb.New(rpcURL, blockNumber)
	return &forkDB{f: f, s: f.State()}
}

func (d *forkDB) ForkDB() *forkdb.ForkDB { return d.f }

func (d *forkDB) ensure(addr Address) {
	_ = d.f.EnsureAccount(context.Background(), addr)
}

func (d *forkDB) ensureSlot(addr Address, slot Hash) {
	_ = d.f.EnsureSlot(context.Background(), addr, slot)
}

func (d *forkDB) GetBalance(addr Address) *uint256.Int {
	d.ensure(addr)
	return d.s.GetBalance(addr)
}
func (d *forkDB) GetNonce(addr Address) uint64 {
	d.ensure(addr)
	return d.s.GetNonce(addr)
}
func (d *forkDB) GetCode(addr Address) []byte {
	d.ensure(addr)
	return d.s.GetCode(addr)
}
func (d *forkDB) GetCodeHash(addr Address) Hash {
	d.ensure(addr)
	return d.s.GetCodeHash(addr)
}
func (d *forkDB) GetStorage(addr Address, slot Hash) Hash {
	d.ensureSlot(addr, slot)
	return d.s.GetStorage(addr, slot)
}
func (d *forkDB) GetCommittedStorage(addr Address, slot Hash) Hash {
	d.ensureSlot(addr, slot)
	return d.s.GetCommittedStorage(addr, slot)
}
func (d *forkDB) SetStorage(addr Address, slot, val Hash)      { d.ensure(addr); d.s.SetStorage(addr, slot, val) }
func (d *forkDB) SetBalance(addr Address, v *uint256.Int)      { d.ensure(addr); d.s.SetBalance(addr, v) }
func (d *forkDB) AddBalance(addr Address, v *uint256.Int)      { d.ensure(addr); d.s.AddBalance(addr, v) }
func (d *forkDB) SubBalance(addr Address, v *uint256.Int)      { d.ensure(addr); d.s.SubBalance(addr, v) }
func (d *forkDB) SetNonce(addr Address, n uint64)              { d.ensure(addr); d.s.SetNonce(addr, n) }
func (d *forkDB) SetCode(addr Address, code []byte, h Hash)   { d.ensure(addr); d.s.SetCode(addr, code, h) }
func (d *forkDB) CreateAccount(addr Address)                   { d.ensure(addr); d.s.CreateAccount(addr) }
func (d *forkDB) Exists(addr Address) bool                     { d.ensure(addr); return d.s.Exists(addr) }
func (d *forkDB) Empty(addr Address) bool                      { d.ensure(addr); return d.s.Empty(addr) }
func (d *forkDB) Suicide(addr Address)                         { d.s.Suicide(addr) }
func (d *forkDB) HasSuicided(addr Address) bool                { return d.s.HasSuicided(addr) }
func (d *forkDB) AddLog(l *state.Log)                          { d.s.AddLog(l) }
func (d *forkDB) AddRefund(g uint64)                           { d.s.AddRefund(g) }
func (d *forkDB) SubRefund(g uint64)                           { d.s.SubRefund(g) }
func (d *forkDB) GetRefund() uint64                            { return d.s.GetRefund() }
func (d *forkDB) AddressWarm(addr Address) bool                { return d.s.AddressWarm(addr) }
func (d *forkDB) WarmAddress(addr Address)                     { d.s.WarmAddress(addr) }
func (d *forkDB) SlotWarm(addr Address, slot Hash) bool        { return d.s.SlotWarm(addr, slot) }
func (d *forkDB) WarmSlot(addr Address, slot Hash)             { d.s.WarmSlot(addr, slot) }
func (d *forkDB) Snapshot() int                                { return d.s.Snapshot() }
func (d *forkDB) RevertToSnapshot(id int)                      { d.s.RevertToSnapshot(id) }
func (d *forkDB) Finalise()                                    { d.s.Finalise() }
func (d *forkDB) CreatedInTx(addr Address) bool                { return d.s.CreatedInTx(addr) }
