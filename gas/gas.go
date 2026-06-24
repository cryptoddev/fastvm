package gas

const (
	Zero          uint64 = 0
	JumpDest      uint64 = 1
	Base          uint64 = 2
	VeryLow       uint64 = 3
	Low           uint64 = 5
	Mid           uint64 = 8
	High          uint64 = 10
	WarmStorageRead uint64 = 100
	ColdAccountAccess uint64 = 2600
	ColdSload       uint64 = 2100
	SstoreSet       uint64 = 20000
	SstoreReset     uint64 = 2900
	SstoreClear     uint64 = 4800
	SstoreNoopGas   uint64 = 100
	SstoreInitGas   uint64 = 20000
	SstoreInitRefund uint64 = 19900
	SstoreCleanRefund uint64 = 4800
	SstoreDirtyRefund uint64 = 0
	Exp             uint64 = 10
	ExpByte         uint64 = 50
	Sha3            uint64 = 30
	Sha3Word        uint64 = 6
	Copy            uint64 = 3
	BlockHash       uint64 = 20
	Log             uint64 = 375
	LogData         uint64 = 8
	LogTopic        uint64 = 375
	Create          uint64 = 32000
	Create2         uint64 = 32000
	Create2Word     uint64 = 6
	Call            uint64 = 0
	CallValue       uint64 = 9000
	CallStipend     uint64 = 2300
	NewAccount      uint64 = 25000
	SelfDestruct    uint64 = 5000
	SelfDestructNewAccount uint64 = 25000
	Return          uint64 = 0
	Revert          uint64 = 0
	Stop            uint64 = 0
	Invalid         uint64 = 0
	ExtCodeSize     uint64 = 0
	ExtCodeCopy     uint64 = 0
	ExtCodeHash     uint64 = 0
	Balance         uint64 = 0
	TxDataZero      uint64 = 4
	TxDataNonZero   uint64 = 16
	TxBase          uint64 = 21000
	TxCreate        uint64 = 53000
	TxAccessListAddress uint64 = 2400
	TxAccessListStorage uint64 = 1900
)

const MaxRefundQuotient uint64 = 5

func CallGas(available uint64) uint64 {
	return available - available/64
}

func CopyWords(n uint64) uint64 {
	return ((n + 31) / 32) * Copy
}

func Sha3Words(n uint64) uint64 {
	return ((n + 31) / 32) * Sha3Word
}

func InitCodeCost(_ uint64) uint64 {
	return 0
}

func SstoreCost(current, original, newVal [32]byte, warm bool) (cost, refund uint64) {
	if current == newVal {
		if warm {
			return SstoreNoopGas, 0
		}
		return ColdSload + SstoreNoopGas, 0
	}
	if original == current {
		if isZero(original) {
			return SstoreSet, 0
		}
		if isZero(newVal) {
			return SstoreReset, SstoreClear
		}
		return SstoreReset, 0
	}
	cost = SstoreNoopGas
	if !isZero(original) {
		if isZero(current) {
		} else if isZero(newVal) {
			refund += SstoreClear
		}
	}
	if original == newVal {
		if isZero(original) {
			refund += SstoreInitRefund
		} else {
			refund += SstoreCleanRefund
		}
	}
	return cost, refund
}

func isZero(b [32]byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}
