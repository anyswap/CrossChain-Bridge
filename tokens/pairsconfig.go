package tokens

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
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
	log.Printf("set token pairs config directory to '%v'", dir)
	fileStat, err := os.Stat(dir)
	if err != nil {
		log.Fatal("wrong token pairs dir", "dir", dir, "err", err)
	}
	if !fileStat.IsDir() {
		log.Fatal("token pairs dir is not directory", "dir", dir)
	}
	tokenPairsConfigDirectory = dir
}

// GetTokenPairsDir get token pairs directory
func GetTokenPairsDir() string {
	return tokenPairsConfigDirectory
}

// SetTokenPairsConfig set token pairs config
func SetTokenPairsConfig(pairsConfig map[string]*TokenPairConfig, check bool) {
	if check {
		err := checkTokenPairsConfig(pairsConfig)
		if err != nil {
			log.Fatalf("check token pairs config error: %v", err)
		}
	}
	tokenPairsConfig = pairsConfig
}

// GetTokenPairsConfig get token pairs config
func GetTokenPairsConfig() map[string]*TokenPairConfig {
	return tokenPairsConfig
}

// GetTokenPairConfig get token pair config
func GetTokenPairConfig(pairID string) *TokenPairConfig {
	pairCfg, exist := tokenPairsConfig[strings.ToLower(pairID)]
	if !exist {
		log.Warn("GetTokenPairConfig: pairID not exist", "pairID", pairID)
		return nil
	}
	return pairCfg
}

// IsTokenPairExist is token pair exist
func IsTokenPairExist(pairID string) bool {
	_, exist := tokenPairsConfig[strings.ToLower(pairID)]
	return exist
}

// GetAllPairIDs get all pairIDs
func GetAllPairIDs() []string {
	pairIDs := make([]string, 0, len(tokenPairsConfig))
	for _, pairCfg := range tokenPairsConfig {
		pairIDs = append(pairIDs, strings.ToLower(pairCfg.PairID))
	}
	return pairIDs
}

// FindTokenConfig find by (tx to) address
func FindTokenConfig(address string, isSrc bool) (configs []*TokenConfig, pairIDs []string) {
	for _, pairCfg := range tokenPairsConfig {
		var tokenCfg *TokenConfig
		if isSrc {
			tokenCfg = pairCfg.SrcToken
		} else {
			tokenCfg = pairCfg.DestToken
		}
		match := false
		if tokenCfg.ContractAddress != "" {
			if strings.EqualFold(tokenCfg.ContractAddress, address) {
				match = true
			}
		} else if isSrc && strings.EqualFold(tokenCfg.DepositAddress, address) {
			match = true
		}
		if match {
			configs = append(configs, tokenCfg)
			pairIDs = append(pairIDs, pairCfg.PairID)
		}
	}
	return configs, pairIDs
}

// GetTokenConfig get token config
func GetTokenConfig(pairID string, isSrc bool) *TokenConfig {
	pairCfg, exist := tokenPairsConfig[strings.ToLower(pairID)]
	if !exist {
		log.Trace("GetTokenConfig: pairID not exist", "pairID", pairID)
		return nil
	}
	if isSrc {
		return pairCfg.SrcToken
	}
	return pairCfg.DestToken
}

// GetTokenConfigsByDirection get token configs by direction
func GetTokenConfigsByDirection(pairID string, isSwapin bool) (fromTokenConfig, toTokenConfig *TokenConfig) {
	pairCfg, exist := tokenPairsConfig[strings.ToLower(pairID)]
	if !exist {
		log.Trace("GetTokenConfigs: pairID not exist", "pairID", pairID)
		return nil, nil
	}
	if isSwapin {
		return pairCfg.SrcToken, pairCfg.DestToken
	}
	return pairCfg.DestToken, pairCfg.SrcToken
}

func checkTokenPairsConfig(pairsConfig map[string]*TokenPairConfig) (err error) {
	pairsMap := make(map[string]struct{})
	srcContractsMap := make(map[string]struct{})
	dstContractsMap := make(map[string]struct{})
	nonContractSrcCount := 0
	for _, tokenPair := range pairsConfig {
		pairID := strings.ToLower(tokenPair.PairID)
		pairsMap[pairID] = struct{}{}
		// check source contract address
		srcContract := strings.ToLower(tokenPair.SrcToken.ContractAddress)
		if srcContract != "" {
			if _, exist := srcContractsMap[srcContract]; exist {
				return fmt.Errorf("duplicate source contract '%v'", tokenPair.SrcToken.ContractAddress)
			}
			srcContractsMap[srcContract] = struct{}{}
		} else {
			nonContractSrcCount++
		}
		// check destination contract address
		dstContract := strings.ToLower(tokenPair.DestToken.ContractAddress)
		if !tokenPair.SrcToken.IsDelegateContract {
			if _, exist := dstContractsMap[dstContract]; exist {
				return fmt.Errorf("duplicate destination contract '%v'", tokenPair.DestToken.ContractAddress)
			}
			dstContractsMap[dstContract] = struct{}{}
		} else if !tokenPair.DestToken.DisableSwap {
			return fmt.Errorf("must close withdraw if is delegate swapin")
		}
		// check config
		err = tokenPair.CheckConfig()
		if err != nil {
			return err
		}
		err = SrcBridge.VerifyTokenConfig(tokenPair.SrcToken)
		if err != nil {
			return err
		}
		err = DstBridge.VerifyTokenConfig(tokenPair.DestToken)
		if err != nil {
			return err
		}
		if *tokenPair.SrcToken.Decimals != *tokenPair.DestToken.Decimals {
			return fmt.Errorf("decimals of pair are not equal, src %v, dest %v", *tokenPair.SrcToken.Decimals, *tokenPair.DestToken.Decimals)
		}
	}
	if nonContractSrcCount > 1 {
		return fmt.Errorf("only support one non-contract token swapin")
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
	pairsConfig, err := LoadTokenPairsConfigInDir(tokenPairsConfigDirectory, check)
	if err != nil {
		log.Fatal("load token pair config error", "err", err)
	}
	SetTokenPairsConfig(pairsConfig, check)
	if TokenPriceCfg != nil {
		go watchAndReloadTokenPrices()
	}
}

// LoadTokenPairsConfigInDir load token pairs config
func LoadTokenPairsConfigInDir(dir string, check bool) (map[string]*TokenPairConfig, error) {
	fileInfoList, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Error("read directory failed", "dir", dir, "err", err)
		return nil, err
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
			return nil, err
		}
		// use all small case to identify
		pairID := strings.ToLower(pairConfig.PairID)
		// check duplicate pairID
		if _, exist := pairsConfig[pairID]; exist {
			return nil, fmt.Errorf("duplicate pairID '%v'", pairConfig.PairID)
		}
		pairsConfig[pairID] = pairConfig
	}
	if check {
		err = checkTokenPairsConfig(pairsConfig)
		if err != nil {
			return nil, err
		}
	}
	return pairsConfig, nil
}

func loadTokenPairConfig(configFile string) (config *TokenPairConfig, err error) {
	log.Println("start load token pair config file", configFile)
	if !common.FileExist(configFile) {
		return nil, fmt.Errorf("config file '%v' not exist", configFile)
	}
	config = &TokenPairConfig{}
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		return nil, fmt.Errorf("toml decode file error: %w", err)
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

// AddPairConfig add pair config dynamically
func AddPairConfig(configFile string) (pairConfig *TokenPairConfig, err error) {
	pairConfig, err = loadTokenPairConfig(configFile)
	if err != nil {
		return nil, err
	}
	err = checkAddTokenPairsConfig(pairConfig)
	if err != nil {
		return nil, err
	}
	// use all small case to identify
	tokenPairsConfig[strings.ToLower(pairConfig.PairID)] = pairConfig
	log.Info("add pair config success", "pairID", pairConfig.PairID, "configFile", configFile)
	return pairConfig, nil
}

func checkAddTokenPairsConfig(pairConfig *TokenPairConfig) (err error) {
	err = pairConfig.CheckConfig()
	if err != nil {
		return err
	}
	err = SrcBridge.VerifyTokenConfig(pairConfig.SrcToken)
	if err != nil {
		return err
	}
	err = DstBridge.VerifyTokenConfig(pairConfig.DestToken)
	if err != nil {
		return err
	}
	pairID := strings.ToLower(pairConfig.PairID)
	if _, exist := tokenPairsConfig[pairID]; exist {
		return fmt.Errorf("pairID '%v' already exist", pairID)
	}
	srcContract := strings.ToLower(pairConfig.SrcToken.ContractAddress)
	if srcContract == "" {
		return fmt.Errorf("source contract address is empty, need restart program")
	}
	isDelegateSwapin := pairConfig.SrcToken.IsDelegateContract
	if isDelegateSwapin && !pairConfig.DestToken.DisableSwap {
		return fmt.Errorf("must close withdraw if is delegate swapin")
	}
	dstContract := strings.ToLower(pairConfig.DestToken.ContractAddress)
	for _, tokenPair := range tokenPairsConfig {
		if strings.EqualFold(srcContract, tokenPair.SrcToken.ContractAddress) {
			return fmt.Errorf("source contract '%v' already exist", srcContract)
		}
		if !isDelegateSwapin && strings.EqualFold(dstContract, tokenPair.DestToken.ContractAddress) {
			return fmt.Errorf("destination contract '%v' already exist", dstContract)
		}
	}
	return nil
}
