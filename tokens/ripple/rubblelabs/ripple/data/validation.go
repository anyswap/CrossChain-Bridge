package data

type Validation struct {
	Hash             Hash256
	Flags            uint32
	LedgerHash       Hash256
	LedgerSequence   uint32
	Amendments       Vector256
	SigningTime      RippleTime
	SigningPubKey    PublicKey
	Signature        VariableLength
	CloseTime        *uint32
	LoadFee          *uint32
	BaseFee          *uint64
	ReserveBase      *uint32
	ReserveIncrement *uint32
}

func (v Validation) GetType() string                 { return "Validation" }
func (v Validation) GetPublicKey() *PublicKey        { return &v.SigningPubKey }
func (v Validation) GetSignature() *VariableLength   { return &v.Signature }
func (v Validation) Prefix() HashPrefix              { return HP_VALIDATION }
func (v Validation) SigningPrefix() HashPrefix       { return HP_VALIDATION }
func (v Validation) SuppressionId() (Hash256, error) { return NodeId(&v) }
func (v Validation) GetHash() *Hash256               { return &v.Hash }
func (v Validation) InitialiseForSigning()           {}
