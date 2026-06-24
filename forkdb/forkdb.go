package forkdb

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cryptoddev/fastvm/state"
	"github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"
)

type Address = state.Address
type Hash = state.Hash

type ForkDB struct {
	mu              sync.RWMutex
	inner           *state.StateDB
	rpcURL          string
	blockNumber     string
	httpClient      *http.Client
	fetchedAccounts map[Address]struct{}
	fetchedSlots    map[Address]map[Hash]struct{}
}

func New(rpcURL string, blockNumber uint64) *ForkDB {
	var blockStr string
	if blockNumber == 0 {
		blockStr = "latest"
	} else {
		blockStr = fmt.Sprintf("0x%x", blockNumber)
	}
	return &ForkDB{
		inner:           state.New(),
		rpcURL:          rpcURL,
		blockNumber:     blockStr,
		httpClient:      &http.Client{Timeout: 10 * time.Second},
		fetchedAccounts: make(map[Address]struct{}),
		fetchedSlots:    make(map[Address]map[Hash]struct{}),
	}
}

func (f *ForkDB) State() *state.StateDB { return f.inner }

func (f *ForkDB) EnsureAccount(ctx context.Context, addr Address) error {
	f.mu.RLock()
	_, fetched := f.fetchedAccounts[addr]
	f.mu.RUnlock()
	if fetched {
		return nil
	}
	return f.fetchAccount(ctx, addr)
}

func (f *ForkDB) fetchAccount(ctx context.Context, addr Address) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, fetched := f.fetchedAccounts[addr]; fetched {
		return nil
	}

	addrHex := "0x" + hex.EncodeToString(addr[:])

	type rpcReq struct {
		JSONRPC string        `json:"jsonrpc"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
		ID      int           `json:"id"`
	}
	// Batch: balance + nonce + code in one RPC call.
	batch := []rpcReq{
		{JSONRPC: "2.0", Method: "eth_getBalance", Params: []interface{}{addrHex, f.blockNumber}, ID: 1},
		{JSONRPC: "2.0", Method: "eth_getTransactionCount", Params: []interface{}{addrHex, f.blockNumber}, ID: 2},
		{JSONRPC: "2.0", Method: "eth_getCode", Params: []interface{}{addrHex, f.blockNumber}, ID: 3},
	}

	results, err := f.batchCall(ctx, batch)
	if err != nil {
		return fmt.Errorf("fetchAccount %x: %w", addr, err)
	}

	var acc state.Account

	if balHex, ok := results[1].(string); ok {
		balHex = strings.TrimPrefix(balHex, "0x")
		if balHex == "" {
			balHex = "0"
		}
		b, _ := hex.DecodeString(padEven(balHex))
		acc.Balance.SetBytes(b)
	}

	if nonceHex, ok := results[2].(string); ok {
		nonceHex = strings.TrimPrefix(nonceHex, "0x")
		if nonceHex == "" {
			nonceHex = "0"
		}
		var n uint256.Int
		b, _ := hex.DecodeString(padEven(nonceHex))
		n.SetBytes(b)
		acc.Nonce = n.Uint64()
	}

	if codeHex, ok := results[3].(string); ok {
		codeHex = strings.TrimPrefix(codeHex, "0x")
		if len(codeHex) > 0 {
			code, err := hex.DecodeString(codeHex)
			if err == nil && len(code) > 0 {
				acc.Code = code
				acc.CodeHash = keccak256Hash(code)
			}
		}
	}

	f.inner.SetAccount(addr, acc)
	f.fetchedAccounts[addr] = struct{}{}
	return nil
}

func (f *ForkDB) EnsureSlot(ctx context.Context, addr Address, slot Hash) error {
	f.mu.RLock()
	m, ok := f.fetchedSlots[addr]
	if ok {
		_, fetched := m[slot]
		f.mu.RUnlock()
		if fetched {
			return nil
		}
	} else {
		f.mu.RUnlock()
	}
	return f.fetchSlot(ctx, addr, slot)
}

func (f *ForkDB) fetchSlot(ctx context.Context, addr Address, slot Hash) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.fetchedSlots[addr]
	if ok {
		if _, fetched := m[slot]; fetched {
			return nil
		}
	} else {
		m = make(map[Hash]struct{})
		f.fetchedSlots[addr] = m
	}

	addrHex := "0x" + hex.EncodeToString(addr[:])
	slotHex := "0x" + hex.EncodeToString(slot[:])

	type rpcReq struct {
		JSONRPC string        `json:"jsonrpc"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
		ID      int           `json:"id"`
	}
	req := rpcReq{
		JSONRPC: "2.0",
		Method:  "eth_getStorageAt",
		Params:  []interface{}{addrHex, slotHex, f.blockNumber},
		ID:      1,
	}

	var resp struct {
		Result string `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := f.call(ctx, req, &resp); err != nil {
		return fmt.Errorf("fetchSlot %x[%x]: %w", addr, slot, err)
	}
	if resp.Error != nil {
		return fmt.Errorf("fetchSlot RPC error: %s", resp.Error.Message)
	}

	valHex := strings.TrimPrefix(resp.Result, "0x")
	var val Hash
	if len(valHex) > 0 {
		b, err := hex.DecodeString(padEven(valHex))
		if err == nil {
			copy(val[32-len(b):], b)
		}
	}

	f.inner.SetStorageDirect(addr, slot, val)
	m[slot] = struct{}{}
	return nil
}

type rpcResponse struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (f *ForkDB) batchCall(ctx context.Context, reqs interface{}) (map[int]interface{}, error) {
	body, err := json.Marshal(reqs)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, f.rpcURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := f.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var responses []rpcResponse
	if err := json.Unmarshal(data, &responses); err != nil {
		return nil, fmt.Errorf("batch unmarshal: %w (body: %.200s)", err, data)
	}

	result := make(map[int]interface{}, len(responses))
	for _, r := range responses {
		if r.Error != nil {
			return nil, fmt.Errorf("RPC error id=%d: %s", r.ID, r.Error.Message)
		}
		var v interface{}
		if err := json.Unmarshal(r.Result, &v); err == nil {
			result[r.ID] = v
		}
	}
	return result, nil
}

func (f *ForkDB) call(ctx context.Context, req interface{}, out interface{}) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, f.rpcURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := f.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

func padEven(s string) string {
	if len(s)%2 != 0 {
		return "0" + s
	}
	return s
}

func keccak256Hash(data []byte) Hash {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	var out Hash
	h.Sum(out[:0])
	return out
}
