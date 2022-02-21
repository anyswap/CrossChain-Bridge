package crypto

type HashVersion byte

const (
	ACCOUNT_ZERO = "rrrrrrrrrrrrrrrrrrrrrhoLvTp"
	ACCOUNT_ONE  = "rrrrrrrrrrrrrrrrrrrrBZbvji"
	NaN          = "rrrrrrrrrrrrrrrrrrrn5RM1rHd"
	ROOT         = "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh"
)

const (
	ALPHABET = "rpshnaf39wBUDNEGHJKLM4PQRST7VWXYZ2bcdeCg65jkm8oFqi1tuvAxyz"

	RIPPLE_ACCOUNT_ID      HashVersion = 0
	RIPPLE_NODE_PUBLIC     HashVersion = 28
	RIPPLE_NODE_PRIVATE    HashVersion = 32
	RIPPLE_FAMILY_SEED     HashVersion = 33
	RIPPLE_ACCOUNT_PRIVATE HashVersion = 34
	RIPPLE_ACCOUNT_PUBLIC  HashVersion = 35
)

var hashTypes = [...]struct {
	Description       string
	Prefix            byte
	Payload           int
	MaximumCharacters int
}{
	RIPPLE_ACCOUNT_ID:      {"Short name for sending funds to an account.", 'r', 20, 35},
	RIPPLE_NODE_PUBLIC:     {"Validation public key for node.", 'n', 33, 53},
	RIPPLE_NODE_PRIVATE:    {"Validation private key for node.", 'p', 32, 52},
	RIPPLE_FAMILY_SEED:     {"Family seed.", 's', 16, 29},
	RIPPLE_ACCOUNT_PRIVATE: {"Account private key.", 'p', 32, 52},
	RIPPLE_ACCOUNT_PUBLIC:  {"Account public key.", 'a', 33, 53},
}
