package ethtest

import (
	"os"
	"testing"
)

const testDir = "/tmp/ethereum-tests/GeneralStateTests"
const fork = "Cancun"

func requireEthereumTests(t *testing.T) {
	t.Helper()
	if os.Getenv("RUN_ETHEREUM_TESTS") != "1" {
		t.Skip("set RUN_ETHEREUM_TESTS=1 to run ethereum/tests GeneralStateTests")
	}
	if _, err := os.Stat(testDir); err != nil {
		t.Skip("ethereum/tests not found at " + testDir)
	}
}

func TestGeneralStateTests_PreCompiled(t *testing.T) {
	requireEthereumTests(t)
	RunDir(t, testDir+"/stPreCompiledContracts", fork)
}

func TestGeneralStateTests_Arithmetic(t *testing.T) {
	requireEthereumTests(t)
	RunDir(t, testDir+"/stArith", fork)
}

func TestGeneralStateTests_Memory(t *testing.T) {
	requireEthereumTests(t)
	RunDir(t, testDir+"/stMemory", fork)
}

func TestGeneralStateTests_Calls(t *testing.T) {
	requireEthereumTests(t)
	RunDir(t, testDir+"/stCallCodes", fork)
}
