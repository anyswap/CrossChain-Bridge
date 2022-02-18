package data

import (
	"fmt"
)

type TransactionFlag uint32
type LedgerEntryFlag uint32

// Transaction Flags
const (
	//Universal flags
	TxCanonicalSignature TransactionFlag = 0x80000000

	// Payment flags
	TxNoDirectRipple TransactionFlag = 0x00010000
	TxPartialPayment TransactionFlag = 0x00020000
	TxLimitQuality   TransactionFlag = 0x00040000
	TxCircle         TransactionFlag = 0x00080000 // Not implemented

	// AccountSet flags
	TxSetRequireDest   TransactionFlag = 0x00000001
	TxSetRequireAuth   TransactionFlag = 0x00000002
	TxSetDisallowXRP   TransactionFlag = 0x00000003
	TxSetDisableMaster TransactionFlag = 0x00000004
	TxSetAccountTxnID  TransactionFlag = 0x00000005
	TxNoFreeze         TransactionFlag = 0x00000006
	TxGlobalFreeze     TransactionFlag = 0x00000007
	TxDefaultRipple    TransactionFlag = 0x00000008
	TxRequireDestTag   TransactionFlag = 0x00010000
	TxOptionalDestTag  TransactionFlag = 0x00020000
	TxRequireAuth      TransactionFlag = 0x00040000
	TxOptionalAuth     TransactionFlag = 0x00080000
	TxDisallowXRP      TransactionFlag = 0x00100000
	TxAllowXRP         TransactionFlag = 0x00200000

	// OfferCreate flags
	TxPassive           TransactionFlag = 0x00010000
	TxImmediateOrCancel TransactionFlag = 0x00020000
	TxFillOrKill        TransactionFlag = 0x00040000
	TxSell              TransactionFlag = 0x00080000

	// TrustSet flags
	TxSetAuth       TransactionFlag = 0x00010000
	TxSetNoRipple   TransactionFlag = 0x00020000
	TxClearNoRipple TransactionFlag = 0x00040000
	TxSetFreeze     TransactionFlag = 0x00100000
	TxClearFreeze   TransactionFlag = 0x00200000

	// EnableAmendments flags
	TxGotMajority  TransactionFlag = 0x00010000
	TxLostMajority TransactionFlag = 0x00020000

	// PaymentChannelClaim flags
	TxRenew TransactionFlag = 0x00010000
	TxClose TransactionFlag = 0x00020000
)

// Ledger entry flags
const (
	// AccountRoot flags
	LsPasswordSpent  LedgerEntryFlag = 0x00010000
	LsRequireDestTag LedgerEntryFlag = 0x00020000
	LsRequireAuth    LedgerEntryFlag = 0x00040000
	LsDisallowXRP    LedgerEntryFlag = 0x00080000
	LsDisableMaster  LedgerEntryFlag = 0x00100000
	LsNoFreeze       LedgerEntryFlag = 0x00200000
	LsGlobalFreeze   LedgerEntryFlag = 0x00400000
	LsDefaultRipple  LedgerEntryFlag = 0x00800000

	// Offer flags
	LsPassive LedgerEntryFlag = 0x00010000
	LsSell    LedgerEntryFlag = 0x00020000

	// RippleState flags
	LsLowReserve   LedgerEntryFlag = 0x00010000
	LsHighReserve  LedgerEntryFlag = 0x00020000
	LsLowAuth      LedgerEntryFlag = 0x00040000
	LsHighAuth     LedgerEntryFlag = 0x00080000
	LsLowNoRipple  LedgerEntryFlag = 0x00100000
	LsHighNoRipple LedgerEntryFlag = 0x00200000
	LsLowFreeze    LedgerEntryFlag = 0x00400000
	LsHighFreeze   LedgerEntryFlag = 0x00800000
)

var txFlagNames = map[TransactionType][]struct {
	Flag TransactionFlag
	Name string
}{
	PAYMENT: {
		{TxNoDirectRipple, "NoDirectRipple"},
		{TxPartialPayment, "PartialPayment"},
		{TxLimitQuality, "LimitQuality"},
		{TxCircle, "Circle"},
	},
	ACCOUNT_SET: {
		{TxSetRequireDest, "SetRequireDest"},
		{TxSetRequireAuth, "SetRequireAuth"},
		{TxSetDisallowXRP, "SetDisallowXRP"},
		{TxSetDisableMaster, "SetDisableMaster"},
		{TxNoFreeze, "NoFreeze"},
		{TxGlobalFreeze, "GlobalFreeze"},
		{TxRequireDestTag, "RequireDestTag"},
		{TxOptionalDestTag, "OptionalDestTag"},
		{TxRequireAuth, "RequireAuth"},
		{TxDisallowXRP, "DisallowXRP"},
		{TxAllowXRP, "AllowXRP"},
	},
	OFFER_CREATE: {
		{TxPassive, "Passive"},
		{TxImmediateOrCancel, "ImmediateOrCancel"},
		{TxFillOrKill, "FillOrKill"},
		{TxSell, "Sell"},
	},
	TRUST_SET: {
		{TxSetAuth, "SetAuth"},
		{TxSetNoRipple, "SetNoRipple"},
		{TxClearNoRipple, "ClearNoRipple"},
		{TxSetFreeze, "SetFreeze"},
		{TxClearFreeze, "ClearFreeze"},
	},
}

var leFlagNames = map[LedgerEntryType][]struct {
	Flag LedgerEntryFlag
	Name string
}{
	ACCOUNT_ROOT: {
		{LsPasswordSpent, "PasswordSpent"},
		{LsRequireDestTag, "RequireDestTag"},
		{LsRequireAuth, "RequireAuth"},
		{LsDisallowXRP, "DisallowXRP"},
		{LsDisableMaster, "DisableMaster"},
		{LsNoFreeze, "NoFreeze"},
	},
	OFFER: {
		{LsPassive, "Passive"},
		{LsSell, "Sell"},
	},
	RIPPLE_STATE: {
		{LsLowReserve, "LowReserve"},
		{LsHighReserve, "HighReserve"},
		{LsLowAuth, "LowAuth"},
		{LsHighAuth, "HighAuth"},
		{LsLowNoRipple, "LowNoRipple"},
		{LsHighNoRipple, "HighNoRipple"},
		{LsLowFreeze, "LowFreeze"},
		{LsHighFreeze, "HighFreeze"},
	},
}

func (f TransactionFlag) String() string {
	return fmt.Sprintf("%08X", uint32(f))
}

func (f LedgerEntryFlag) String() string {
	return fmt.Sprintf("%08X", uint32(f))
}

func (f TransactionFlag) Explain(tx Transaction) []string {
	var flags []string
	if f&TxCanonicalSignature > 0 {
		flags = append(flags, "CanonicalSignature")
	}
	for _, n := range txFlagNames[tx.GetTransactionType()] {
		if f&n.Flag > 0 {
			flags = append(flags, n.Name)
		}
	}
	return flags
}

func (f LedgerEntryFlag) Explain(le LedgerEntry) []string {
	var flags []string
	for _, n := range leFlagNames[le.GetLedgerEntryType()] {
		if f&n.Flag > 0 {
			flags = append(flags, n.Name)
		}
	}
	return flags
}
