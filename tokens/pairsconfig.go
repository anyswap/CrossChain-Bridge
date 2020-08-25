package tokens

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
)

var (
	tokenPairsConfigDirectory string

	tokenPairsConfig map[string]*TokenPairConfig
)

// TokenPairConfig pair config
type TokenPairConfig struct {
	PairID    string
	SrcToken  *TokenConfig
	DestToken *TokenConfig
}

// SetTokenPairsDir set token pairs directory
func SetTokenPairsDir(dir string) {
	log.Printf("set token pairs config directory to '%v'\n", dir)
	tokenPairsConfigDirectory = dir
}

// SetTokenPairsConfig set token pairs config
func SetTokenPairsConfig(pairsConfig map[string]*TokenPairConfig, check bool) {
	tokenPairsConfig = pairsConfig
	if !check {
		return
	}
	err := checkTokenPairsConfig()
	if err != nil {
		log.Fatalf("check token pairs config error: %v", err)
	}
}

// GetTokenPairsConfig get token pairs config
func GetTokenPairsConfig() map[string]*TokenPairConfig {
	return tokenPairsConfig
}

// GetTokenPairConfig get token pair config
func GetTokenPairConfig(pairID string) *TokenPairConfig {
	pairCfg, exist := tokenPairsConfig[pairID]
	if !exist {
		return nil
	}
	return pairCfg
}

// IsTokenPairExist is token pair exist
func IsTokenPairExist(pairID string) bool {
	_, exist := tokenPairsConfig[pairID]
	return exist
}

// FindTokenConfig find by (tx to) address
func FindTokenConfig(address string, isSrc bool) (config *TokenConfig, pairID string) {
	var tokenCfg *TokenConfig
	for _, pairCfg := range tokenPairsConfig {
		if isSrc {
			tokenCfg = pairCfg.SrcToken
		} else {
			tokenCfg = pairCfg.DestToken
		}
		if tokenCfg.ContractAddress != "" {
			if strings.EqualFold(tokenCfg.ContractAddress, address) {
				return tokenCfg, pairCfg.PairID
			}
		} else if strings.EqualFold(tokenCfg.DepositAddress, address) {
			return tokenCfg, pairCfg.PairID
		}
	}
	return nil, ""
}

// GetTokenConfig get token config
func GetTokenConfig(pairID string, isSrc bool) *TokenConfig {
	pairCfg, exist := tokenPairsConfig[pairID]
	if !exist {
		return nil
	}
	if isSrc {
		return pairCfg.SrcToken
	}
	return pairCfg.DestToken
}

// GetAllDepositAddresses get all deposit addresses
func GetAllDepositAddresses() []string {
	var addrs []string
	for _, cfg := range tokenPairsConfig {
		addrs = append(addrs, cfg.SrcToken.DepositAddress)
	}
	return addrs
}

func checkTokenPairsConfig() (err error) {
	pairsMap := make(map[string]struct{})
	srcContractsMap := make(map[string]struct{})
	dstContractsMap := make(map[string]struct{})
	for _, tokenPair := range tokenPairsConfig {
		pairID := strings.ToLower(tokenPair.PairID)
		if _, exist := pairsMap[pairID]; exist {
			return fmt.Errorf("duplicate pairID '%v'", pairID)
		}
		pairsMap[pairID] = struct{}{}
		srcContract := strings.ToLower(tokenPair.SrcToken.ContractAddress)
		if srcContract != "" {
			if _, exist := srcContractsMap[srcContract]; exist {
				return fmt.Errorf("duplicate source contract '%v'", srcContract)
			}
			srcContractsMap[srcContract] = struct{}{}
		}
		dstContract := strings.ToLower(tokenPair.DestToken.ContractAddress)
		if _, exist := dstContractsMap[dstContract]; exist {
			return fmt.Errorf("duplicate destinatio contract '%v'", dstContract)
		}
		dstContractsMap[dstContract] = struct{}{}
		err = tokenPair.CheckConfig()
		if err != nil {
			return err
		}
		SrcBridge.VerifyTokenConfig(tokenPair.SrcToken)
		DstBridge.VerifyTokenConfig(tokenPair.DestToken)
	}
	return nil
}

// CheckConfig check token pair config
func (c *TokenPairConfig) CheckConfig() (err error) {
	if c.PairID == "" {
		return errors.New("tokenPair must config nonempty 'PairID'")
	}
	if c.SrcToken == nil {
		return errors.New("tokenPair must config 'SrcToken'")
	}
	if c.DestToken == nil {
		return errors.New("tokenPair must config 'DestToken'")
	}
	err = c.SrcToken.CheckConfig(true)
	if err != nil {
		return err
	}
	err = c.DestToken.CheckConfig(false)
	if err != nil {
		return err
	}
	return nil
}

// LoadTokenPairsConfig load token pairs config
func LoadTokenPairsConfig(check bool) {
	LoadTokenPairsConfigInDir(tokenPairsConfigDirectory, check)
}

// LoadTokenPairsConfigInDir load token pairs config
func LoadTokenPairsConfigInDir(dir string, check bool) {
	fileInfoList, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal("read dir error", "directory", dir, "err", err)
	}
	pairsConfig := make(map[string]*TokenPairConfig)
	for _, info := range fileInfoList {
		if info.IsDir() {
			continue
		}
		fileName := info.Name()
		if !strings.HasSuffix(fileName, ".toml") {
			log.Info("ignore not *.toml file", "file", fileName)
			continue
		}
		var pairConfig *TokenPairConfig
		filePath := common.AbsolutePath(dir, fileName)
		pairConfig, err = loadTokenPairConfig(filePath)
		if err != nil {
			log.Fatal("load token pair config error", "fileName", filePath, "err", err)
		}
		pairsConfig[pairConfig.PairID] = pairConfig
	}
	SetTokenPairsConfig(pairsConfig, check)
}

func loadTokenPairConfig(configFile string) (config *TokenPairConfig, err error) {
	log.Println("start load token pair config file", configFile)
	if !common.FileExist(configFile) {
		return nil, fmt.Errorf("config file '%v' not exist", configFile)
	}
	config = &TokenPairConfig{}
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		return nil, fmt.Errorf("toml decode file error: %v", err)
	}
	var bs []byte
	if log.JSONFormat {
		bs, _ = json.Marshal(config)
	} else {
		bs, _ = json.MarshalIndent(config, "", "  ")
	}
	log.Tracef("load token pair finished. %v", string(bs))
	log.Println("finish load token pair config file", configFile)
	return config, nil
}
