package ckb

import "strings"

type CKBSwapinCommitment struct {
	FromAddress   string
	DepositAddres string
}

func (b *Bridge) NewCKBSwapinCommitment(fromAddress string) *CKBSwapinCommitment {
	var depositAddress string // TODO, get deposit address from bridge config
	return &CKBSwapinCommitment{
		FromAddress:    strings.ToLower(fromAddress),
		DepositAddress: strings.ToLower(depositAddress),
	}
}

func (c CKBSwapinCommitment) Key() string {
	return strings.ToLower(c.FromAddress)
}

func (c CKBSwapinCommitment) Validate(event interface{}) bool {
	// swapin 认定规则
	// 1. event 是 types.Transaction
	// 2. Outputs 包含至少 1 个 cell 锁定脚本与 c.DepositAddress 对应
	// 3. type 脚本是 nil，data 不检查
	// 4. Inputs 包含至少一个 cell，锁定脚本与 c.FromAddress 对应
	// 5. Inputs 中属于 c.FromAddress 的 cell 的 capacity 总和大于等于 outputs 中属于 c.DepositAddress 的 cell 的 capacity
	return false
}
