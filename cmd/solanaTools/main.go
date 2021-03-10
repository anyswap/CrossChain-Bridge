/*
This file provides Solana dev tools like generate key pair, build address, sign and verify tx, call rpc etc.
*/
package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"

	bin "github.com/dfuse-io/binary"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/program/system"
)

func main() {
	//keyTest()
	newTransferInstruction()
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func keyTest() {
	// 1. Generate new key pair
	fmt.Printf("\n\n======== 1. Generate new key pair ========\n\n")
	GenerateKeyPair()

	// private Base58: 3tFWtC14qLFNCZjGZHhBjE9Ff78SUtvVrcV13QPz2nRiQV6JpycbYp7oRibUn39jYYm65nHNVA6CSv6rHvEXY3vm
	// public Base58: 7R9zUfmcXPUFGEtWtjuFUjhW5WD2i4G6ZL4TFbDJSozu
	// private hex: 903AF3796D5A9ADB3E2936C4ED5A7349DA29E01ED80EB8850A5570DAAFB38CE15F563A2419B55C64AFB8565CD8883E6EDC184AAAC9197180490725ECEE6F248E
	// public hex: 5F563A2419B55C64AFB8565CD8883E6EDC184AAAC9197180490725ECEE6F248E

	// 2.Read private key Base58
	fmt.Printf("\n\n======== 2. Read private key base58 ========\n\n")
	priv2, err := solana.PrivateKeyFromBase58("3tFWtC14qLFNCZjGZHhBjE9Ff78SUtvVrcV13QPz2nRiQV6JpycbYp7oRibUn39jYYm65nHNVA6CSv6rHvEXY3vm")
	checkError(err)
	pub2 := priv2.PublicKey()
	fmt.Printf("Private key:\n%s\nPublic key:\n%s\n", priv2, pub2)
	fmt.Printf("\nPrivate key hex:\n%X\nPublic key hex:\n%X\n", []byte(priv2), pub2[:])

	// 3. Read public key Base58, convert to hex
	fmt.Printf("\n\n======== 3. Read public key Base58, convert to hex ========\n\n")
	pub3, err := solana.PublicKeyFromBase58("7R9zUfmcXPUFGEtWtjuFUjhW5WD2i4G6ZL4TFbDJSozu")
	checkError(err)
	fmt.Printf("Public key hex:\n%X\n", pub3[:])

	// 4. Read public key hex, convert to Base58
	fmt.Printf("\n\n======== 4. Read public key hex, convert to Base58 ========\n\n")
	pub4, err := PubkeyHexToAddress("5F563A2419B55C64AFB8565CD8883E6EDC184AAAC9197180490725ECEE6F248E")
	checkError(err)
	fmt.Printf("Public key Base58 (address):\n%v\n", pub4)
}

// GenerateKeyPair returns a new ed25519 key pair
func GenerateKeyPair() {
	pub, priv, err := solana.NewRandomPrivateKey()
	checkError(err)

	// Base58 format
	fmt.Printf("Private key:\n%s\nPublic key:\n%s\n", priv, pub)

	// Hex format
	// private key has a 64 bytes including 32 bytes suffix, which is the public key
	// public key has 32 bytes
	fmt.Printf("\nPrivate key hex:\n%X\nPublic key hex:\n%X\n", []byte(priv), pub[:])
}

// PubkeyHexToAddress returns public key address, that is just the public key encoded in base58
func PubkeyHexToAddress(pubkeyHex string) (string, error) {
	bz, err := hex.DecodeString(pubkeyHex)
	if err != nil {
		return "", errors.New("Decode pubkey hex error")
	}
	pub := PublicKeyFromBytes(bz)
	return fmt.Sprintf("%s", pub), nil
}

func PublicKeyFromBytes(in []byte) (out solana.PublicKey) {
	byteCount := len(in)
	if byteCount == 0 {
		return
	}

	max := 32
	if byteCount < max {
		max = byteCount
	}

	copy(out[:], in[0:max])
	return
}

func newTransferInstruction() {
	/*
		type Transfer struct {
			// Prefixed with byte 0x02
			Lamports bin.Uint64
			Accounts *TransferAccounts `bin:"-"`
		}
		type TransferAccounts struct {
			From *solana.AccountMeta `text:"linear,notype"`
			To   *solana.AccountMeta `text:"linear,notype"`
		}
	*/
	From, err := solana.PublicKeyFromBase58("7R9zUfmcXPUFGEtWtjuFUjhW5WD2i4G6ZL4TFbDJSozu")
	checkError(err)
	To, err := solana.PublicKeyFromBase58("2z55nksdCojo3jDW5reezbZMEvBQmdgPvMa7djMn3vR4")
	checkError(err)
	lamports := 2333

	transfer := &system.Instruction{
		BaseVariant: bin.BaseVariant{
			TypeID: 0,
			Impl: &system.Transfer{
				Lamports: bin.Uint64(lamports),
				Accounts: &system.TransferAccounts{
					From: &solana.AccountMeta{PublicKey: from, IsSigner: true, IsWritable: true},
					To:   &solana.AccountMeta{PublicKey: to, IsSigner: false, IsWritable: true},
				},
			},
		},
	}
	fmt.Printf("Transfer instruction:\n%+v\n", transfer)
	data := transfer.Data()
	fmt.Printf("Transfer data:\n%v\n", data)
	programID := transfer.ProgramID()
	fmt.Printf("Transfer program ID:\n%v\n", programID)
}
