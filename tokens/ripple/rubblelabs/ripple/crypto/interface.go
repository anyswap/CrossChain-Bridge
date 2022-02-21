package crypto

import "math/big"

type Key interface {
	Private(*uint32) []byte
	Id(*uint32) []byte
	Public(*uint32) []byte
}

type Hash interface {
	Version() HashVersion
	Payload() []byte
	PayloadTrimmed() []byte
	Value() *big.Int
	String() string
	Clone() Hash
	MarshalText() ([]byte, error)
}
