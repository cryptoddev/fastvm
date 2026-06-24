package evm

import (
	"errors"

	"github.com/cryptoddev/fastvm/gas"
	"github.com/cryptoddev/fastvm/interpreter"
	"github.com/cryptoddev/fastvm/precompile"
	"github.com/cryptoddev/fastvm/state"
	"github.com/cryptoddev/fastvm/tracer"
	"github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"
)

type Address = state.Address
type Hash = state.Hash

type Config struct {
	ChainID     uint64
	BlockNumber uint64
	Time        uint64
	GasLimit    uint64
	BaseFee     uint64
	Coinbase    Address
	Difficulty  uint64
	Trace       bool
}

type AccessListEntry struct {
	Address Address
	Slots   []Hash
}

type Message struct {
	From              Address
	To                *Address
	Gas               uint64
	Value             *uint256.Int
	Data              []byte
	GasPrice          uint64
	ApplyIntrinsicGas bool
	AccessList        []AccessListEntry
}

type Result struct {
	ReturnData      []byte
	GasUsed         uint64
	GasLeft         uint64
	Err             error
	ContractAddress *Address
}

type DB interface {
	interpreter.StateDB
	GetBalance(Address) *uint256.Int
	GetNonce(Address) uint64
	GetCode(Address) []byte
	GetCodeHash(Address) Hash
	SetBalance(Address, *uint256.Int)
	SetNonce(Address, uint64)
	SetCode(Address, []byte, Hash)
	CreateAccount(Address)
	Exists(Address) bool
	HasSuicided(Address) bool
	GetRefund() uint64
	Snapshot() int
	RevertToSnapshot(int)
	Finalise()
}

type TracerIface = interpreter.TracerIface

type CallTracerIface = interpreter.CallTracerIface

type EVM struct {
	db          DB
	config      Config
	interp      *interpreter.Interpreter
	block       interpreter.BlockContext
	tx          interpreter.TxContext
	tracer      TracerIface
	callTracer  CallTracerIface
}

func New(db DB, cfg Config) *EVM {
	e := &EVM{db: db, config: cfg}
	e.interp = interpreter.New(e)
	e.block = interpreter.BlockContext{
		Coinbase:    cfg.Coinbase,
		BlockNumber: cfg.BlockNumber,
		Time:        cfg.Time,
		GasLimit:    cfg.GasLimit,
	}
	e.block.Difficulty.SetUint64(cfg.Difficulty)
	e.block.BaseFee.SetUint64(cfg.BaseFee)
	e.block.ChainID.SetUint64(cfg.ChainID)
	if cfg.Trace {
		e.callTracer = tracer.New()
	}
	return e
}

func (e *EVM) Tracer() *tracer.CallTracer {
	if t, ok := e.callTracer.(*tracer.CallTracer); ok {
		return t
	}
	return nil
}

func NewFromFork(rpcURL string, blockNumber uint64, cfg Config) (*EVM, *forkDB) {
	f := newForkDB(rpcURL, blockNumber)
	return New(f, cfg), f
}

func (e *EVM) Execute(msg Message) Result {
	// Deduct upfront gas cost (gasLimit * gasPrice) from sender.
	upfront := new(uint256.Int).SetUint64(msg.Gas)
	upfront.Mul(upfront, new(uint256.Int).SetUint64(msg.GasPrice))
	senderBal := e.db.GetBalance(msg.From)
	if senderBal.Lt(upfront) {
		return Result{Err: interpreter.ErrInsufficientFunds, GasUsed: msg.Gas}
	}
	e.db.SubBalance(msg.From, upfront)
	e.db.SetNonce(msg.From, e.db.GetNonce(msg.From)+1)

	e.tx.Origin = msg.From
	e.tx.GasPrice.SetUint64(msg.GasPrice)

	// EIP-2929: precompile addresses are always warm.
	for i := byte(1); i <= 9; i++ {
		var pa Address
		pa[19] = i
		e.db.WarmAddress(pa)
	}

	origGas := msg.Gas

	for _, entry := range msg.AccessList {
		e.db.WarmAddress(entry.Address)
		for _, slot := range entry.Slots {
			e.db.WarmSlot(entry.Address, slot)
		}
	}

	if msg.ApplyIntrinsicGas {
		ig := intrinsicGas(msg.Data, msg.To == nil, msg.AccessList)
		if msg.Gas < ig {
			e.db.AddBalance(msg.From, upfront)
			return Result{Err: ErrIntrinsicGas, GasUsed: msg.Gas}
		}
		msg.Gas -= ig
	}

	var result Result
	if msg.To == nil {
		result = e.create(msg, nil)
	} else {
		result = e.execCall(msg, interpreter.CallTypeCall, *msg.To, msg.From)
	}

	// Refund unused gas (gasLeft * gasPrice) to sender.
	refund := new(uint256.Int).SetUint64(result.GasLeft)
	refund.Mul(refund, new(uint256.Int).SetUint64(msg.GasPrice))
	e.db.AddBalance(msg.From, refund)

	// EIP-3529: cap storage refund at gasUsed/5.
	if result.Err == nil || result.Err == interpreter.ErrRevert {
		totalGasUsed := origGas - result.GasLeft
		maxRefund := totalGasUsed / gas.MaxRefundQuotient
		r := e.db.GetRefund()
		if r > maxRefund {
			r = maxRefund
		}
		if r > 0 {
			bonus := new(uint256.Int).SetUint64(r * msg.GasPrice)
			e.db.AddBalance(msg.From, bonus)
		}
	}

	e.db.Finalise()

	// Pay coinbase the effective tip (gasPrice - baseFee).
	if msg.GasPrice > e.config.BaseFee {
		tip := msg.GasPrice - e.config.BaseFee
		totalGasUsed := origGas - result.GasLeft
		fee := new(uint256.Int).SetUint64(totalGasUsed)
		fee.Mul(fee, new(uint256.Int).SetUint64(tip))
		if !fee.IsZero() {
			e.db.AddBalance(e.config.Coinbase, fee)
		}
	}

	return result
}

func (e *EVM) ExecuteWithTrace(msg Message, tr TracerIface) Result {
	e.tracer = tr
	defer func() { e.tracer = nil }()
	return e.Execute(msg)
}

func (e *EVM) ExecuteWithCallTrace(msg Message, tr CallTracerIface) Result {
	e.callTracer = tr
	defer func() { e.callTracer = nil }()
	return e.Execute(msg)
}

func (e *EVM) execCall(msg Message, ct interpreter.CallType, contract, caller Address) Result {
	// Fast path: precompile check before snapshot/state touch.
	// Precompiles work for CALL, CALLCODE, STATICCALL (not DELEGATECALL).
	if ct != interpreter.CallTypeDelegateCall {
		if out, gasLeft, isPrecompile := precompile.Run(contract, msg.Data, msg.Gas); isPrecompile {
			return Result{ReturnData: out, GasLeft: gasLeft, GasUsed: msg.Gas - gasLeft}
		}
	}

	snap := e.db.Snapshot()

	e.db.WarmAddress(caller)
	e.db.WarmAddress(contract)

	// CALL/CALLCODE: value transfer from caller to contract. DELEGATECALL/STATICCALL: no transfer.
	if (ct == interpreter.CallTypeCall || ct == interpreter.CallTypeCallCode) && msg.Value != nil && !msg.Value.IsZero() {
		bal := e.db.GetBalance(caller)
		if bal.Lt(msg.Value) {
			e.db.RevertToSnapshot(snap)
			return Result{Err: interpreter.ErrInsufficientFunds, GasUsed: msg.Gas}
		}
		e.db.SubBalance(caller, msg.Value)
		e.db.AddBalance(contract, msg.Value)
	}

	code := e.db.GetCode(contract)
	if len(code) == 0 {
		return Result{GasLeft: msg.Gas}
	}

	if e.callTracer != nil && e.callTracer.Enabled() {
		e.callTracer.OnEnter(interpreter.CallType(ct), caller, contract, msg.Data, msg.Gas, msg.Value)
	}

	f := e.newFrame(code, contract, caller, msg)
	result := e.interp.Run(f)
	interpreter.ReleaseFrame(f)

	if e.callTracer != nil && e.callTracer.Enabled() {
		e.callTracer.OnExit(result.ReturnData, msg.Gas-result.GasLeft, result.Err)
	}

	if result.Err != nil {
		e.db.RevertToSnapshot(snap)
	}
	return Result{
		ReturnData: result.ReturnData,
		GasUsed:    msg.Gas - result.GasLeft,
		GasLeft:    result.GasLeft,
		Err:        result.Err,
	}
}

func (e *EVM) create(msg Message, salt []byte) Result {
	snap := e.db.Snapshot()

	var contractAddr Address
	if salt == nil {
		contractAddr = createAddress(msg.From, e.db.GetNonce(msg.From))
	} else {
		contractAddr = create2Address(msg.From, salt, msg.Data)
	}

	e.db.SetNonce(msg.From, e.db.GetNonce(msg.From)+1)
	e.db.CreateAccount(contractAddr)
	e.db.SetNonce(contractAddr, 1)
	e.db.WarmAddress(contractAddr)

	if msg.Value != nil && !msg.Value.IsZero() {
		bal := e.db.GetBalance(msg.From)
		if bal.Lt(msg.Value) {
			e.db.RevertToSnapshot(snap)
			return Result{Err: interpreter.ErrInsufficientFunds, GasUsed: msg.Gas}
		}
		e.db.SubBalance(msg.From, msg.Value)
		e.db.AddBalance(contractAddr, msg.Value)
	}

	f := e.newFrame(msg.Data, contractAddr, msg.From, msg)
	result := e.interp.Run(f)
	interpreter.ReleaseFrame(f)

	if result.Err != nil && result.Err != interpreter.ErrRevert {
		e.db.RevertToSnapshot(snap)
		return Result{Err: result.Err, GasUsed: msg.Gas - result.GasLeft}
	}
	if result.Err == interpreter.ErrRevert {
		e.db.RevertToSnapshot(snap)
		return Result{Err: result.Err, ReturnData: result.ReturnData, GasUsed: msg.Gas - result.GasLeft}
	}

	codeHash := keccak256Hash(result.ReturnData)
	e.db.SetCode(contractAddr, result.ReturnData, codeHash)

	return Result{
		ReturnData:      result.ReturnData,
		GasUsed:         msg.Gas - result.GasLeft,
		GasLeft:         result.GasLeft,
		ContractAddress: &contractAddr,
	}
}

func (e *EVM) newFrame(code []byte, contract, caller Address, msg Message) *interpreter.Frame {
	f := interpreter.AcquireFrame()
	f.Code = code
	f.Gas = msg.Gas
	f.Contract = contract
	f.Caller = caller
	f.Input = msg.Data
	f.State = e.db
	f.Block = &e.block
	// e.tx.Origin is set in Execute(); preserved across all sub-calls via pointer.
	f.Tx = &e.tx
	if msg.Value != nil {
		f.Value.Set(msg.Value)
	} else {
		f.Value.Clear()
	}
	f.Tracer = e.tracer
	return f
}

func (e *EVM) Call(parent *interpreter.Frame, ct interpreter.CallType, addr Address, value, gasIn uint64, input []byte) ([]byte, uint64, error) {
	if parent.Depth+1 >= interpreter.MaxCallDepth {
		return nil, gasIn, interpreter.ErrDepthLimit
	}

	var caller, contract, codeAddr Address
	switch ct {
	case interpreter.CallTypeDelegateCall:
		caller = parent.Caller
		contract = parent.Contract
		codeAddr = addr
	case interpreter.CallTypeCallCode:
		caller = parent.Contract
		contract = parent.Contract
		codeAddr = addr
	default:
		caller = parent.Contract
		contract = addr
		codeAddr = addr
	}

	val := uint256.NewInt(value)
	msg := Message{From: caller, To: &contract, Gas: gasIn, Value: val, Data: input}

	var result Result
	if ct == interpreter.CallTypeStaticCall {
		snap := e.db.Snapshot()
		code := e.db.GetCode(codeAddr)
		f := e.newFrame(code, contract, caller, msg)
		f.Static = true
		f.Depth = parent.Depth + 1
		r := e.interp.Run(f)
		interpreter.ReleaseFrame(f)
		if r.Err != nil && r.Err != interpreter.ErrRevert {
			e.db.RevertToSnapshot(snap)
		}
		result = Result{ReturnData: r.ReturnData, GasLeft: r.GasLeft, Err: r.Err}
	} else if ct == interpreter.CallTypeCallCode || ct == interpreter.CallTypeDelegateCall {
		snap := e.db.Snapshot()
		code := e.db.GetCode(codeAddr)

		if ct == interpreter.CallTypeCallCode && value > 0 {
			bal := e.db.GetBalance(caller)
			if bal.Lt(val) {
				return nil, gasIn, interpreter.ErrInsufficientFunds
			}
		}

		if len(code) == 0 {
			result = Result{GasLeft: gasIn}
		} else {
			if e.callTracer != nil && e.callTracer.Enabled() {
				e.callTracer.OnEnter(ct, caller, contract, input, gasIn, val)
			}
			f := e.newFrame(code, contract, caller, msg)
			f.Depth = parent.Depth + 1
			r := e.interp.Run(f)
			interpreter.ReleaseFrame(f)
			if e.callTracer != nil && e.callTracer.Enabled() {
				e.callTracer.OnExit(r.ReturnData, gasIn-r.GasLeft, r.Err)
			}
			if r.Err != nil {
				e.db.RevertToSnapshot(snap)
			}
			result = Result{ReturnData: r.ReturnData, GasLeft: r.GasLeft, Err: r.Err}
		}
	} else {
		result = e.execCall(msg, ct, contract, caller)
	}
	return result.ReturnData, result.GasLeft, result.Err
}

func (e *EVM) Create(parent *interpreter.Frame, value uint64, initCode []byte, salt []byte) (Address, uint64, error) {
	if parent.Depth+1 >= interpreter.MaxCallDepth {
		return Address{}, 0, interpreter.ErrDepthLimit
	}
	val := uint256.NewInt(value)
	msg := Message{From: parent.Contract, Gas: parent.Gas, Value: val, Data: initCode}
	result := e.create(msg, salt)
	var addr Address
	if result.ContractAddress != nil {
		addr = *result.ContractAddress
	}
	return addr, result.GasLeft, result.Err
}

func createAddress(deployer Address, nonce uint64) Address {
	var rlp []byte
	addrEncoded := append([]byte{0x94}, deployer[:]...)
	var nonceEncoded []byte
	switch {
	case nonce == 0:
		nonceEncoded = []byte{0x80}
	case nonce < 0x80:
		nonceEncoded = []byte{byte(nonce)}
	default:
		var b [8]byte
		n := nonce
		i := 7
		for n > 0 {
			b[i] = byte(n)
			n >>= 8
			i--
		}
		byteLen := 7 - i
		nonceEncoded = append([]byte{0x80 + byte(byteLen)}, b[i+1:]...)
	}
	listLen := len(addrEncoded) + len(nonceEncoded)
	rlp = append([]byte{0xc0 + byte(listLen)}, addrEncoded...)
	rlp = append(rlp, nonceEncoded...)
	h := keccak256Hash(rlp)
	var addr Address
	copy(addr[:], h[12:])
	return addr
}

func create2Address(deployer Address, salt []byte, initCode []byte) Address {
	codeHash := keccak256Hash(initCode)
	var buf [85]byte
	buf[0] = 0xff
	copy(buf[1:21], deployer[:])
	copy(buf[21:53], salt)
	copy(buf[53:85], codeHash[:])
	h := keccak256Hash(buf[:])
	var addr Address
	copy(addr[:], h[12:])
	return addr
}

func keccak256Hash(data []byte) Hash {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	var out Hash
	h.Sum(out[:0])
	return out
}

var ErrIntrinsicGas = errors.New("intrinsic gas too low")

func intrinsicGas(data []byte, isCreate bool, accessList []AccessListEntry) uint64 {
	g := uint64(21000)
	if isCreate {
		g += 32000
	}
	for _, b := range data {
		if b == 0 {
			g += 4
		} else {
			g += 16
		}
	}
	for _, entry := range accessList {
		g += 2400
		g += uint64(len(entry.Slots)) * 1900
	}
	return g
}
