package ethtest

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cryptoddev/fastvm/evm"
	"github.com/cryptoddev/fastvm/state"
	"github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"
)

type stateTest map[string]testCase

type testCase struct {
	Env  envJSON            `json:"env"`
	Pre  map[string]accJSON `json:"pre"`
	Tx   txJSON             `json:"transaction"`
	Post map[string][]postEntry `json:"post"`
}

type envJSON struct {
	Coinbase    string `json:"currentCoinbase"`
	Difficulty  string `json:"currentDifficulty"`
	GasLimit    string `json:"currentGasLimit"`
	Number      string `json:"currentNumber"`
	Timestamp   string `json:"currentTimestamp"`
	BaseFee     string `json:"currentBaseFee"`
}

type accJSON struct {
	Balance string            `json:"balance"`
	Code    string            `json:"code"`
	Nonce   string            `json:"nonce"`
	Storage map[string]string `json:"storage"`
}

type txJSON struct {
	Data       []string          `json:"data"`
	GasLimit   []string          `json:"gasLimit"`
	GasPrice   string            `json:"gasPrice"`
	Nonce      string            `json:"nonce"`
	To         string            `json:"to"`
	Value      []string          `json:"value"`
	From       string            `json:"sender"`
	AccessList []accessListEntry `json:"accessList"`
}

type accessListEntry struct {
	Address     string   `json:"address"`
	StorageKeys []string `json:"storageKeys"`
}

type postEntry struct {
	Indexes struct {
		Data  int `json:"data"`
		Gas   int `json:"gas"`
		Value int `json:"value"`
	} `json:"indexes"`
	State map[string]accJSON `json:"state"`
}

func RunFile(t *testing.T, path string, fork string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var tests stateTest
	if err := json.Unmarshal(data, &tests); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	for name, tc := range tests {
		posts, ok := tc.Post[fork]
		if !ok {
			continue
		}
		for i, post := range posts {
			t.Run(fmt.Sprintf("%s/%d", name, i), func(t *testing.T) {
				runCase(t, tc, post)
			})
		}
	}
}

func runCase(t *testing.T, tc testCase, post postEntry) {
	t.Helper()

	db := state.New()

	for addrHex, acc := range tc.Pre {
		addr := mustAddr(addrHex)
		bal := mustBig(acc.Balance)
		var u uint256.Int
		u.SetFromBig(bal)
		nonce := mustUint64(acc.Nonce)
		code := mustBytes(acc.Code)
		var codeHash [32]byte
		if len(code) > 0 {
			codeHash = keccak256Hash(code)
		}
		a := state.Account{}
		a.Balance.SetFromBig(bal)
		a.Nonce = nonce
		a.Code = code
		a.CodeHash = codeHash
		db.SetAccount(addr, a)
		for slot, val := range acc.Storage {
			db.SetStorageDirect(addr, mustHash(slot), mustHash(val))
		}
	}

	cfg := evm.Config{
		ChainID:     1,
		BlockNumber: mustUint64(tc.Env.Number),
		Time:        mustUint64(tc.Env.Timestamp),
		GasLimit:    mustUint64(tc.Env.GasLimit),
		BaseFee:     mustUint64(tc.Env.BaseFee),
		Coinbase:    mustAddr(tc.Env.Coinbase),
	}
	if d := mustBig(tc.Env.Difficulty); d != nil {
		cfg.Difficulty = d.Uint64()
	}

	vm := evm.New(db, cfg)

	di := post.Indexes.Data
	gi := post.Indexes.Gas
	vi := post.Indexes.Value
	calldata := mustBytes(tc.Tx.Data[di])
	gasLimit := mustUint64(tc.Tx.GasLimit[gi])
	value := mustU256(tc.Tx.Value[vi])
	from := mustAddr(tc.Tx.From)

	var toPtr *evm.Address
	if tc.Tx.To != "" {
		a := mustAddr(tc.Tx.To)
		toPtr = &a
	}

	var accessList []evm.AccessListEntry
	for _, e := range tc.Tx.AccessList {
		entry := evm.AccessListEntry{Address: mustAddr(e.Address)}
		for _, s := range e.StorageKeys {
			entry.Slots = append(entry.Slots, mustHash(s))
		}
		accessList = append(accessList, entry)
	}

	_ = vm.Execute(evm.Message{
		From:              from,
		To:                toPtr,
		Gas:               gasLimit,
		Value:             value,
		Data:              calldata,
		GasPrice:          mustUint64(tc.Tx.GasPrice),
		ApplyIntrinsicGas: true,
		AccessList:        accessList,
	})

	// Verify post-state.
	for addrHex, expected := range post.State {
		addr := mustAddr(addrHex)
		gotBal := db.GetBalance(addr)
		wantBal := mustU256(expected.Balance)
		if !gotBal.Eq(wantBal) {
			t.Errorf("addr %s balance: got %s want %s", addrHex, gotBal, wantBal)
		}
		gotNonce := db.GetNonce(addr)
		wantNonce := mustUint64(expected.Nonce)
		if gotNonce != wantNonce {
			t.Errorf("addr %s nonce: got %d want %d", addrHex, gotNonce, wantNonce)
		}
		for slot, wantVal := range expected.Storage {
			got := db.GetStorage(addr, mustHash(slot))
			want := mustHash(wantVal)
			if got != want {
				t.Errorf("addr %s slot %s: got %x want %x", addrHex, slot, got, want)
			}
		}
	}
}

func mustAddr(s string) evm.Address {
	s = strings.TrimPrefix(s, "0x")
	b, _ := hex.DecodeString(fmt.Sprintf("%040s", s))
	var a evm.Address
	copy(a[:], b[len(b)-20:])
	return a
}

func mustHash(s string) [32]byte {
	s = strings.TrimPrefix(s, "0x")
	b, _ := hex.DecodeString(fmt.Sprintf("%064s", s))
	var h [32]byte
	copy(h[:], b[len(b)-32:])
	return h
}

func mustBytes(s string) []byte {
	s = strings.TrimPrefix(s, "0x")
	if s == "" {
		return nil
	}
	b, _ := hex.DecodeString(s)
	return b
}

func mustBig(s string) *big.Int {
	s = strings.TrimPrefix(s, "0x")
	if s == "" {
		return new(big.Int)
	}
	n := new(big.Int)
	n.SetString(s, 16)
	return n
}

func mustUint64(s string) uint64 {
	return mustBig(s).Uint64()
}

func mustU256(s string) *uint256.Int {
	b := mustBig(s)
	var u uint256.Int
	u.SetFromBig(b)
	return &u
}

func keccak256Hash(data []byte) [32]byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	var out [32]byte
	h.Sum(out[:0])
	return out
}

func RunDir(t *testing.T, dir string, fork string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("test dir not found: %s", dir)
		return
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		t.Run(e.Name(), func(t *testing.T) {
			RunFile(t, path, fork)
		})
	}
}
