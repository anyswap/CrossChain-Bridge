/*
This file provides Solana dev tools like generate key pair, build address, sign and verify tx, call rpc etc.
*/
package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"

	bin "github.com/dfuse-io/binary"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/programs/system"
	"github.com/dfuse-io/solana-go/rpc"
	"github.com/dfuse-io/solana-go/rpc/ws"
	"github.com/mr-tron/base58"
)

func main() {
	//key_test()
	//tx_test()
	//GetLatestBlock()
	//SubscribeAccount()
	SearchTxs()
}

func DecodeTransferData() {
	val, err := base58.Decode("3Bxs45iLYCoeyGyd")
	checkError(err)
	fmt.Printf("val: %v\n", val)
	if val[0] == byte(0x2) {
		fmt.Println("It's a transfer")
	}
	lamports := new(bin.Uint64)
	decoder := bin.NewDecoder(val[4:])
	err = decoder.Decode(lamports)
	checkError(err)
	fmt.Printf("lamports: %v\n", uint64(*lamports))
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func key_test() {
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

func buildUnsignedTx(fromAddress, toAddress string, amount *big.Int) *solana.Transaction {
	from, err := solana.PublicKeyFromBase58(fromAddress)
	checkError(err)
	to, err := solana.PublicKeyFromBase58(toAddress)
	checkError(err)
	lamports := amount.Uint64()

	transfer := &system.Instruction{
		BaseVariant: bin.BaseVariant{
			TypeID: 2, // 0 表示 create account，1 空缺，2 表示 transfer
			Impl: &system.Transfer{
				Lamports: bin.Uint64(lamports),
				Accounts: &system.TransferAccounts{
					From: &solana.AccountMeta{PublicKey: from, IsSigner: true, IsWritable: true},
					To:   &solana.AccountMeta{PublicKey: to, IsSigner: false, IsWritable: true},
				},
			},
		},
	}

	/*ctx := context.Background()
	cli := GetClient()

	resRbt, err := cli.GetRecentBlockhash(ctx, "finalized")
	checkError(err)
	blockHash := resRbt.Value.Blockhash*/
	blockHash, _ := solana.PublicKeyFromBase58("BZWW21AB8Qx2eQRWBcNQNQ4ZRRaQDwmNU1no6iQChTyS")
	fmt.Printf("\nRecent block hash: %v\n", blockHash)

	opt := &solana.Options{
		Payer: from,
	}

	tx, err := solana.TransactionWithInstructions([]solana.TransactionInstruction{transfer}, blockHash, opt)
	checkError(err)
	fmt.Printf("\nTransaction: %+v\n", tx)
	return tx
}

func signTx(tx *solana.Transaction, priv solana.PrivateKey) []byte {
	m := tx.Message
	fmt.Printf("\nMessage: %+v\n", m)

	buf := new(bytes.Buffer)
	err := bin.NewEncoder(buf).Encode(m)
	checkError(err)

	messageCnt := buf.Bytes()
	fmt.Printf("\nMessage bytes: %+v\n", messageCnt)
	signature, err := priv.Sign(messageCnt)
	checkError(err)
	fmt.Printf("\nSignature: %+v\n", signature)
	fmt.Printf("\nSignature bytes: %+v\n", signature[:])
	return signature[:]
}

func makeSignedTx(tx *solana.Transaction, sig []byte) *solana.Transaction {
	var signature [64]byte
	copy(signature[:], sig)
	tx.Signatures = append(tx.Signatures, signature)
	fmt.Printf("\nSigned tx: %+v\n", tx)
	return tx
}

func simulateTx(tx *solana.Transaction) {
	ctx := context.Background()
	cli := GetClient()
	resSmt, err := cli.SimulateTransaction(ctx, tx)
	checkError(err)
	fmt.Printf("\nSimulate transaction result: %+v\n", resSmt)
}

func sendTx(tx *solana.Transaction) {
	ctx := context.Background()
	cli := GetClient()
	txid, err := cli.SendTransaction(ctx, tx)
	checkError(err)
	fmt.Printf("\nSend transaction success: %v\n", txid) // 2Rt9koHr14HL3MKKoq1iqSE1z8vC6a7MCsNih7R4v2XyGSVDzstDJqagicJUfwTmZFD9WHTFtuY3r6qgwd6haWrH*/
}

func tx_test() {
	tx := buildUnsignedTx("7R9zUfmcXPUFGEtWtjuFUjhW5WD2i4G6ZL4TFbDJSozu", "2z55nksdCojo3jDW5reezbZMEvBQmdgPvMa7djMn3vR4", big.NewInt(2333))

	priv, _ := solana.PrivateKeyFromBase58("3tFWtC14qLFNCZjGZHhBjE9Ff78SUtvVrcV13QPz2nRiQV6JpycbYp7oRibUn39jYYm65nHNVA6CSv6rHvEXY3vm")

	sig := signTx(tx, priv)

	signedTx := makeSignedTx(tx, sig)

	// 仿真
	simulateTx(signedTx)

	// 真实发送
	//sendTx(signedTx)
}

func GetClient() *rpc.Client {
	var endpoint = "https://testnet.solana.com"
	cli := rpc.NewClient(endpoint)
	return cli
}

func GetLatestBlock() {
	ctx := context.Background()
	var endpoint = "https://testnet.solana.com"
	cli := rpc.NewClient(endpoint)
	res, err := cli.GetSlot(ctx, "")
	checkError(err)
	fmt.Printf("res: %+v\n", res)

	block, err := cli.GetConfirmedBlock(ctx, uint64(bin.Uint64(res)), "")
	checkError(err)
	fmt.Printf("block: %+v\n", block)
}

func SubscribeAccount() {
	ctx := context.Background()
	var endpoint = "wss://testnet.solana.com"
	cli, err := ws.Dial(ctx, endpoint)
	checkError(err)
	acct, _ := solana.PublicKeyFromBase58("7R9zUfmcXPUFGEtWtjuFUjhW5WD2i4G6ZL4TFbDJSozu")
	sbscrpt, err := cli.AccountSubscribe(acct, "finalized")
	checkError(err)
	fmt.Printf("subscription: %+v\n", sbscrpt)
	for {
		res, err := sbscrpt.Recv()
		checkError(err)
		fmt.Printf("res: %+v\n", res)
	}
}

func SubscribeSlot() {
	ctx := context.Background()
	var endpoint = "wss://testnet.solana.com"
	cli, err := ws.Dial(ctx, endpoint)
	checkError(err)
	sbscrpt, err := cli.SlotSubscribe()
	checkError(err)
	fmt.Printf("subscription: %+v\n", sbscrpt)
	for {
		res, err := sbscrpt.Recv()
		checkError(err)
		fmt.Printf("res: %+v\n", res)
	}
}

func searchTxs(address string, before, until string, limit uint64) (txs []string, err error) {
	acct, err := solana.PublicKeyFromBase58(address)
	checkError(err)

	opts := &rpc.GetConfirmedSignaturesForAddress2Opts{
		Limit: limit,
	}
	if until != "" {
		opts.Until = until
	}
	if before != "" {
		opts.Before = before
	}

	ctx := context.Background()
	var endpoint = "https://testnet.solana.com"
	cli := rpc.NewClient(endpoint)

	res, err := cli.GetConfirmedSignaturesForAddress2(ctx, acct, opts)
	checkError(err)
	txs = make([]string, 0)
	for _, tx := range res {
		txs = append(txs, tx.Signature)
	}
	return txs, nil
}

func searchAllTxs(address string, start, end string) (txs []string, err error) {
	before := end
	util := start
	limit := uint64(5)
	txs = make([]string, 0)
	for {
		txs1, err := searchTxs(address, before, util, limit)
		if err != nil {
			return nil, err
		}
		txs = append(txs, txs1...)
		if len(txs1) == 0 || txs1[len(txs1)-1] == util {
			break
		}
		before = txs[len(txs)-1]
	}
	if end != "" {
		txs = append([]string{end}, txs...)
	}
	if start != "" {
		txs = append(txs, start)
	}
	return txs, nil
}

func SearchTxs() {
	address := "2z55nksdCojo3jDW5reezbZMEvBQmdgPvMa7djMn3vR4"
	start := ""
	//start := "67DUqEMTzRfr9WWrd28Sbdh1tYRhF9AFjSBDbARuaMzkV5GZ46wxnehDyMMGZVXogDQKoqhxspSvGQjWzXpcFT8C"
	end := ""
	//end := "3tvdDrKbRA7XCzc3aKAyKVHLTGuiU71mzXPCwDMWvU4NQtJBX7PNt9iDMbUh54BM9f9gqBSxBtSJpDvbF1wcRreP"
	txs, err := searchAllTxs(address, start, end)
	checkError(err)
	for _, txid := range txs {
		fmt.Printf("tx: %+v\n", txid)
	}
}
