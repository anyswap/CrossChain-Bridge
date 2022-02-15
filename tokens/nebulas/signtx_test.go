package nebulas

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/nebulasio/go-nebulas/crypto/keystore"
	"github.com/nebulasio/go-nebulas/crypto/keystore/secp256k1"
	"github.com/stretchr/testify/assert"
)

func mockAddress() *Address {
	ks := keystore.DefaultKS
	priv1 := secp256k1.GeneratePrivateKey()
	pubdata1, _ := priv1.PublicKey().Encoded()
	addr, _ := NewAddressFromPublicKey(pubdata1)
	ks.SetKey(addr.String(), priv1, []byte("passphrase"))
	ks.Unlock(addr.String(), []byte("passphrase"), time.Second*60*60*24*365)
	return addr
}

func TestTransaction_Verify(t *testing.T) {
	bridge := NewCrossChainBridge(true)
	testCount := 1
	type testTx struct {
		name   string
		tx     *Transaction
		signer *ecdsa.PrivateKey
		count  int
	}

	tests := []testTx{}

	for index := 0; index < testCount; index++ {

		to := mockAddress()

		privKey, err := secp256k1.NewECDSAPrivateKey()
		assert.Nil(t, err)
		pbytes, err := secp256k1.FromECDSAPrivateKey(privKey)
		assert.Nil(t, err)
		nprivKey := new(secp256k1.PrivateKey)
		err = nprivKey.Decode(pbytes)
		assert.Nil(t, err)
		pubdata1, _ := nprivKey.PublicKey().Encoded()
		from, _ := NewAddressFromPublicKey(pubdata1)

		tx, _ := NewTransaction(1, from, to, &big.Int{}, 10, TxPayloadCallType, []byte("datadata"), big.NewInt(20000000000), 2000000)

		test := testTx{fmt.Sprintf("%d", index), tx, privKey, 1}
		tests = append(tests, test)
	}
	for _, tt := range tests {
		for index := 0; index < tt.count; index++ {
			t.Run(tt.name, func(t *testing.T) {
				tx, _, err := bridge.SignTransactionWithPrivateKey(tt.tx, tt.signer)
				assert.Nil(t, err)
				assert.NotNil(t, tx)
				_, err = bridge.signTxWithSignature(tx.(*Transaction), tt.tx.sign, tt.tx.from)
				assert.Nil(t, err)
			})
		}
	}
}
