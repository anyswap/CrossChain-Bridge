package cosmos

// account number is associated with account on a cosmos state
var accountNumberCached map[string]uint64

// GetAccountNumberCached get account number cached
func (b *Bridge) GetAccountNumberCached(address string) (uint64, error) {
	if accountNumberCached == nil {
		accountNumberCached = make(map[string]uint64)
	}
	if num, ok := accountNumberCached[address]; ok && num > 0 {
		return num, nil
	}
	num, err := b.GetAccountNumber(address)
	if err != nil {
		return 0, err
	}
	accountNumberCached[address] = num
	return num, err
}
