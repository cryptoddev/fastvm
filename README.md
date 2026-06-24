# fastvm — EVM Implementation in Go

A standalone Ethereum Virtual Machine implementation written from scratch. No go-ethereum dependency.

## Architecture

```
fastvm/
├── types/        — Word (U256), Address, Hash, Gas
├── stack/        — 1024-slot operand stack, array-backed, zero heap per push
├── memory/       — Byte-addressable memory with lazy expansion and gas metering
├── gas/          — Berlin/London gas schedules
├── state/        — In-memory StateDB with journal, snapshot, and revert
├── forkdb/       — Lazy-loading state backend with RPC fallback and caching
├── interpreter/  — Execution loop and all opcodes through London
├── evm/          — Top-level entry point: CALL, DELEGATECALL, STATICCALL, CREATE, CREATE2
├── precompile/   — Precompiled contracts 0x01–0x09
├── tracer/       — Call tree tracer with Foundry-style rendering
└── ethtest/      — Ethereum State Tests runner
```

## Design

- **Zero allocations on hot path** — stack uses a value array, memory doubles capacity, no heap in push/pop
- **No go-ethereum** — uses `holiman/uint256` for 256-bit arithmetic and `golang.org/x/crypto/sha3` for Keccak256
- **ForkDB** — lazy state loading from RPC with account, code, and storage slot caching
- **Berlin/London rules** — EIP-2929 access lists, EIP-3529 refund cap, EIP-1559 base fee

## Quick Start

```go
db := forkdb.New(rpcURL, blockNumber)
vm := evm.New(db, evm.Config{ChainID: 1})

result, err := vm.Call(evm.Message{
    From:  caller,
    To:    &contractAddr,
    Gas:   1_000_000,
    Value: uint256.NewInt(0),
    Data:  calldata,
})
```

## Testing

```bash
go test ./...
go test -bench=. ./...
```

## License

MIT
