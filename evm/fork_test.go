package evm_test

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/cryptoddev/fastvm/evm"
	"github.com/holiman/uint256"
)

func rpcURL() string { return os.Getenv("RPC_URL") }

func TestForkWBNBBalance(t *testing.T) {
	if rpcURL() == "" { t.Skip("RPC_URL not set") }
	vm, _ := evm.NewFromFork(rpcURL(), 0, evm.Config{ChainID: 56, GasLimit: 30_000_000})
	var wbnb evm.Address
	b, _ := hex.DecodeString("bb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c")
	copy(wbnb[:], b)
	calldata, _ := hex.DecodeString("70a08231" + "0000000000000000000000000000000000000000000000000000000000000000")
	var caller evm.Address; caller[19] = 1
	r := vm.Execute(evm.Message{From: caller, To: &wbnb, Gas: 100_000, Value: uint256.NewInt(0), Data: calldata})
	if r.Err != nil { t.Fatal(r.Err) }
	t.Logf("WBNB.balanceOf(0x0) = 0x%x (gasUsed=%d)", r.ReturnData, r.GasUsed)
}

func TestForkPancakeV2Reserves(t *testing.T) {
	if rpcURL() == "" { t.Skip("RPC_URL not set") }
	vm, _ := evm.NewFromFork(rpcURL(), 0, evm.Config{ChainID: 56, GasLimit: 30_000_000})

	// PancakeSwap V2 WBNB/BUSD pair
	var pair evm.Address
	b, _ := hex.DecodeString("58F876857a02D6762E0101bb5C46A8c1ED44Dc16")
	copy(pair[:], b)

	// getReserves() selector: 0x0902f1ac
	calldata, _ := hex.DecodeString("0902f1ac")
	var caller evm.Address; caller[19] = 1

	r := vm.Execute(evm.Message{From: caller, To: &pair, Gas: 100_000, Value: uint256.NewInt(0), Data: calldata})
	if r.Err != nil { t.Fatalf("getReserves failed: %v", r.Err) }
	if len(r.ReturnData) < 64 { t.Fatalf("short return: %d bytes", len(r.ReturnData)) }

	r0 := new(uint256.Int).SetBytes32(r.ReturnData[0:32])
	r1 := new(uint256.Int).SetBytes32(r.ReturnData[32:64])
	ts := binary.BigEndian.Uint32(r.ReturnData[92:96])

	t.Logf("WBNB/BUSD reserves: r0=%s r1=%s ts=%d gasUsed=%d", r0, r1, ts, r.GasUsed)
	if r0.IsZero() || r1.IsZero() { t.Error("reserves should not be zero") }
}

func TestForkERC20TotalSupply(t *testing.T) {
	if rpcURL() == "" { t.Skip("RPC_URL not set") }
	vm, _ := evm.NewFromFork(rpcURL(), 0, evm.Config{ChainID: 56, GasLimit: 30_000_000})

	// CAKE token
	var cake evm.Address
	b, _ := hex.DecodeString("0E09FaBB73Bd3Ade0a17ECC321fD13a19e81cE82")
	copy(cake[:], b)

	calldata, _ := hex.DecodeString("18160ddd") // totalSupply()
	var caller evm.Address; caller[19] = 1

	r := vm.Execute(evm.Message{From: caller, To: &cake, Gas: 100_000, Value: uint256.NewInt(0), Data: calldata})
	if r.Err != nil { t.Fatalf("totalSupply failed: %v", r.Err) }

	supply := new(uint256.Int).SetBytes32(r.ReturnData[0:32])
	t.Logf("CAKE totalSupply = %s (gasUsed=%d)", supply, r.GasUsed)
	fmt.Printf("CAKE totalSupply = %s\n", supply)
}
