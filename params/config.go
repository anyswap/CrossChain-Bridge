package params

type TokenConfig struct {
	BlockChain      string
	ID              string `toml:",omitempty" json:",omitempty"`
	Name            string
	Symbol          string
	Decimals        uint8
	Description     string `toml:",omitempty" json:",omitempty"`
	ContractAddress string `toml:",omitempty" json:",omitempty"`
	DcrmAddress     string `toml:",omitempty" json:",omitempty"`
}

type GatewayConfig struct {
	ApiAddress string
}

type ApiServerConfig struct {
	Port int
}
