// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package common

import (
	"math"
	"math/big"
)

// Common big integers often used
var (
	Big1   = big.NewInt(1)
	Big2   = big.NewInt(2)
	Big3   = big.NewInt(3)
	Big0   = big.NewInt(0)
	Big32  = big.NewInt(32)
	Big256 = big.NewInt(256)
	Big257 = big.NewInt(257)

	BigMaxUint64 = new(big.Int).SetUint64(math.MaxUint64)
)

// MarshalBigInt marshalls big int into text string for consistent encoding
func MarshalBigInt(i *big.Int) (string, error) {
	bz, err := i.MarshalText()
	if err != nil {
		return "", err
	}
	return string(bz), nil
}

// MustMarshalBigInt marshalls big int into text string for consistent encoding.
// It panics if an error is encountered.
func MustMarshalBigInt(i *big.Int) string {
	str, err := MarshalBigInt(i)
	if err != nil {
		panic(err)
	}
	return str
}

// UnmarshalBigInt unmarshalls string from *big.Int
func UnmarshalBigInt(s string) (*big.Int, error) {
	ret := new(big.Int)
	err := ret.UnmarshalText([]byte(s))
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// MustUnmarshalBigInt unmarshalls string from *big.Int.
// It panics if an error is encountered.
func MustUnmarshalBigInt(s string) *big.Int {
	ret, err := UnmarshalBigInt(s)
	if err != nil {
		panic(err)
	}
	return ret
}
