package terra

import (
	"encoding/json"
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgExecuteContract{}

var (
	errWrongSender   = errors.New("wrong sender address")
	errWrongContract = errors.New("wrong contract address")
)

// NewMsgExecuteContract new
func NewMsgExecuteContract(sender, contract, execMsg string) *MsgExecuteContract {
	return &MsgExecuteContract{
		Sender:     sender,
		Contract:   contract,
		ExecuteMsg: execMsg,
	}
}

// String impl sdk.Msg
func (m MsgExecuteContract) String() string {
	jsData, _ := json.Marshal(m)
	return string(jsData)
}

// Route impl sdk.Msg
func (m MsgExecuteContract) Route() string {
	return "wasm"
}

// Type impl sdk.Msg
func (m MsgExecuteContract) Type() string {
	return "exec"
}

// GetSignBytes impl sdk.Msg (legacy)
func (m MsgExecuteContract) GetSignBytes() []byte {
	return nil
}

// ValidateBasic impl sdk.Msg
func (m MsgExecuteContract) ValidateBasic() error {
	if m.Sender == "" {
		return errWrongSender
	}
	if m.Contract == "" {
		return errWrongContract
	}
	return nil
}

// GetSigners impl sdk.Msg interface
func (m MsgExecuteContract) GetSigners() []sdk.AccAddress {
	accSender, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{accSender}
}
