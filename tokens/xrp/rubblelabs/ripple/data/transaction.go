package data

type TxBase struct {
	TransactionType    TransactionType
	Flags              *TransactionFlag `json:",omitempty"`
	SourceTag          *uint32          `json:",omitempty"`
	Account            Account
	Sequence           uint32
	Fee                Value
	AccountTxnID       *Hash256        `json:",omitempty"`
	SigningPubKey      *PublicKey      `json:",omitempty"`
	TxnSignature       *VariableLength `json:",omitempty"`
	Memos              Memos           `json:",omitempty"`
	PreviousTxnID      *Hash256        `json:",omitempty"`
	LastLedgerSequence *uint32         `json:",omitempty"`
	Hash               Hash256         `json:"hash"`
}

type Payment struct {
	TxBase
	Destination    Account
	Amount         Amount
	SendMax        *Amount  `json:",omitempty"`
	DeliverMin     *Amount  `json:",omitempty"`
	Paths          *PathSet `json:",omitempty"`
	DestinationTag *uint32  `json:",omitempty"`
	InvoiceID      *Hash256 `json:",omitempty"`
}

type AccountSet struct {
	TxBase
	EmailHash     *Hash128        `json:",omitempty"`
	WalletLocator *Hash256        `json:",omitempty"`
	WalletSize    *uint32         `json:",omitempty"`
	MessageKey    *VariableLength `json:",omitempty"`
	Domain        *VariableLength `json:",omitempty"`
	TransferRate  *uint32         `json:",omitempty"`
	TickSize      *uint8          `json:",omitempty"`
	SetFlag       *uint32         `json:",omitempty"`
	ClearFlag     *uint32         `json:",omitempty"`
}

type AccountDelete struct {
	TxBase
	Destination    Account
	DestinationTag *uint32 `json:",omitempty"`
}

type SetRegularKey struct {
	TxBase
	RegularKey *RegularKey `json:",omitempty"`
}

type OfferCreate struct {
	TxBase
	OfferSequence *uint32 `json:",omitempty"`
	TakerPays     Amount
	TakerGets     Amount
	Expiration    *uint32 `json:",omitempty"`
}

type OfferCancel struct {
	TxBase
	OfferSequence uint32
}

type TrustSet struct {
	TxBase
	LimitAmount Amount
	QualityIn   *uint32 `json:",omitempty"`
	QualityOut  *uint32 `json:",omitempty"`
}

type SetFee struct {
	TxBase
	BaseFee           Uint64Hex
	ReferenceFeeUnits uint32
	ReserveBase       uint32
	ReserveIncrement  uint32
}

type Amendment struct {
	TxBase
	Amendment Hash256
}

type EscrowCreate struct {
	TxBase
	Destination    Account
	Amount         Amount
	Digest         *Hash256 `json:",omitempty"`
	CancelAfter    *uint32  `json:",omitempty"`
	FinishAfter    *uint32  `json:",omitempty"`
	DestinationTag *uint32  `json:",omitempty"`
}

type EscrowFinish struct {
	TxBase
	Owner         Account
	OfferSequence uint32
	Method        *uint8   `json:",omitempty"`
	Digest        *Hash256 `json:",omitempty"`
	Proof         *Hash256 `json:",omitempty"`
}

type EscrowCancel struct {
	TxBase
	Owner         Account
	OfferSequence uint32
}

type PaymentChannelCreate struct {
	TxBase
	Amount         Amount
	Destination    Account
	SettleDelay    uint32
	PublicKey      PublicKey
	CancelAfter    *uint32 `json:",omitempty"`
	DestinationTag *uint32 `json:",omitempty"`
	SourceTag      *uint32 `json:",omitempty"`
}

type PaymentChannelFund struct {
	TxBase
	Channel    Hash256
	Amount     Amount
	Expiration *uint32 `json:",omitempty"`
}

type PaymentChannelClaim struct {
	TxBase
	Channel   Hash256
	Balance   *Amount         `json:",omitempty"`
	Amount    *Amount         `json:",omitempty"`
	Signature *VariableLength `json:",omitempty"`
	PublicKey *PublicKey      `json:",omitempty"`
}

// CheckCreate, CheckCash, CheckCancel enabled by amendment 157D2D480E006395B76F948E3E07A45A05FE10230D88A7993C71F97AE4B1F2D1

// https://ripple.com/build/transactions/#checkcreate
type CheckCreate struct {
	TxBase
	Destination    Account
	SendMax        Amount
	DestinationTag *uint32  `json:",omitempty"`
	Expiration     *uint32  `json:",omitempty"`
	InvoiceID      *Hash256 `json:",omitempty"`
}

// https://ripple.com/build/transactions/#checkcash
// Must include one of Amount or DeliverMin
type CheckCash struct {
	TxBase
	CheckID    Hash256
	Amount     *Amount `json:",omitempty"`
	DeliverMin *Amount `json:",omitempty"`
}

// https://ripple.com/build/transactions/#checkcancel
type CheckCancel struct {
	TxBase
	CheckID Hash256
}

type TicketCreate struct {
	Target     *Account `json:",omitempty"`
	Expiration *uint32  `json:",omitempty"`
}

type TicketCancel struct {
	TicketID Hash256
}

type SignerListSet struct {
	TxBase
	SignerQuorum  uint32        `json:",omitempty"`
	SignerEntries []SignerEntry `json:",omitempty"`
}

func (t *TxBase) GetBase() *TxBase                    { return t }
func (t *TxBase) GetType() string                     { return txNames[t.TransactionType] }
func (t *TxBase) GetTransactionType() TransactionType { return t.TransactionType }
func (t *TxBase) Prefix() HashPrefix                  { return HP_TRANSACTION_ID }
func (t *TxBase) GetPublicKey() *PublicKey            { return t.SigningPubKey }
func (t *TxBase) GetSignature() *VariableLength       { return t.TxnSignature }
func (t *TxBase) SigningPrefix() HashPrefix           { return HP_TRANSACTION_SIGN }
func (t *TxBase) PathSet() PathSet                    { return PathSet(nil) }
func (t *TxBase) GetHash() *Hash256                   { return &t.Hash }

func (t *TxBase) Compare(other *TxBase) int {
	switch {
	case t.Account.Equals(other.Account):
		switch {
		case t.Sequence == other.Sequence:
			return t.GetHash().Compare(*other.GetHash())
		case t.Sequence < other.Sequence:
			return -1
		default:
			return 1
		}
	case t.Account.Less(other.Account):
		return -1
	default:
		return 1
	}
}

func (t *TxBase) InitialiseForSigning() {
	if t.SigningPubKey == nil {
		t.SigningPubKey = new(PublicKey)
	}
	if t.TxnSignature == nil {
		t.TxnSignature = new(VariableLength)
	}
}

func (o *OfferCreate) Ratio() *Value {
	return o.TakerPays.Ratio(o.TakerGets)
}

func (p *Payment) PathSet() PathSet {
	if p.Paths == nil {
		return PathSet(nil)
	}
	return *p.Paths
}
