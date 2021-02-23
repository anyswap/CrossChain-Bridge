package cosmos

var accountNumberCached map[string]uint64

func (b *Bridge) GetAccountNumberCached(address string) (uint64, error) {
	if num, ok := accountNumberCached[address]; ok {
		return num, nil
	} else {
		num, err := b.GetAccountNumber(address)
		if err != nil {
			return 0, err
		}
		accountNumberCached[address] = num
		return num, err
	}
}
