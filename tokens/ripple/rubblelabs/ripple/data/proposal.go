package data

type Proposal struct {
	Hash           Hash256
	LedgerHash     Hash256
	PreviousLedger Hash256
	Sequence       uint32
	CloseTime      RippleTime
	PublicKey      PublicKey
	Signature      VariableLength
}

func (p Proposal) GetType() string                { return "Proposal" }
func (p *Proposal) GetPublicKey() *PublicKey      { return &p.PublicKey }
func (p *Proposal) GetSignature() *VariableLength { return &p.Signature }
func (p *Proposal) Prefix() HashPrefix            { return HP_PROPOSAL }
func (p *Proposal) SigningPrefix() HashPrefix     { return HP_PROPOSAL }
func (p *Proposal) GetHash() *Hash256             { return &p.Hash }
func (p *Proposal) InitialiseForSigning()         {}

func (p Proposal) SigningValues() []interface{} {
	return []interface{}{
		p.Sequence,
		p.CloseTime.Uint32(),
		p.PreviousLedger,
		p.LedgerHash,
	}
}

func (p Proposal) SuppressionId() (Hash256, error) {
	return hashValues([]interface{}{
		p.LedgerHash,
		p.PreviousLedger,
		p.Sequence,
		p.CloseTime.Uint32(),
		p.PublicKey,
		p.Signature,
	})
}
