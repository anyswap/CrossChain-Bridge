package mongodb

type SwapTxType uint16

const (
	SwapinTx     SwapTxType = iota // 0
	SwapoutTx                      // 1
	P2shSwapinTx                   // 2
)

func (txtype SwapTxType) String() string {
	switch txtype {
	case SwapinTx:
		return "SwapinTx"
	case SwapoutTx:
		return "SwapoutTx"
	case P2shSwapinTx:
		return "P2shSwapinTx"
	default:
		panic("unknown swap tx type")
	}
}
