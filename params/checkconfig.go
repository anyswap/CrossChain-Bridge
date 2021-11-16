package params

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
)

var blankOrCommaSepRegexp = regexp.MustCompile(`[\s,]+`) // blank or comma separated

func splitStringByBlankOrComma(str string) []string {
	return blankOrCommaSepRegexp.Split(strings.TrimSpace(str), -1)
}

// CheckConfig check config
func CheckConfig(isServer bool) (err error) {
	config := GetConfig()
	if config.Identifier == "" {
		return errors.New("server must config non empty 'Identifier'")
	}
	err = checkChainAndGatewayConfig(isServer)
	if err != nil {
		return err
	}
	if isServer {
		if config.Server == nil {
			return errors.New("server must config 'Server'")
		}
		err = config.Server.CheckConfig()
		if err != nil {
			return err
		}
	} else {
		err = config.Oracle.CheckConfig()
		if err != nil {
			return err
		}
	}
	if config.Dcrm == nil {
		return errors.New("server must config 'Dcrm'")
	}
	err = config.Dcrm.CheckConfig(isServer)
	if err != nil {
		return err
	}
	if config.Extra != nil {
		err = config.Extra.CheckConfig()
		if err != nil {
			return err
		}
	}
	return nil
}

func checkChainAndGatewayConfig(isServer bool) (err error) {
	config := GetConfig()
	if config.SrcChain == nil {
		return errors.New("server must config 'SrcChain'")
	}
	if config.SrcGateway == nil {
		return errors.New("server must config 'SrcGateway'")
	}
	if config.DestChain == nil {
		return errors.New("server must config 'DestChain'")
	}
	if config.DestGateway == nil {
		return errors.New("server must config 'DestGateway'")
	}
	err = config.SrcChain.CheckConfig(isServer)
	if err != nil {
		return err
	}
	err = config.DestChain.CheckConfig(isServer)
	if err != nil {
		return err
	}
	return nil
}

// CheckConfig check swap server config
func (c *ServerConfig) CheckConfig() error {
	if c.MongoDB == nil {
		return errors.New("server must config 'Server.MongoDB'")
	}
	if err := c.MongoDB.CheckConfig(); err != nil {
		return err
	}
	if c.APIServer == nil {
		return errors.New("server must config 'Server.APIServer'")
	}
	return nil
}

// CheckConfig check mongodb config
func (c *MongoDBConfig) CheckConfig() error {
	if c.DBName == "" {
		return errors.New("mongodb must config 'DBName'")
	}
	if c.DBURL == "" && len(c.DBURLs) == 0 {
		return errors.New("mongodb must config 'DBURL' or 'DBURLs'")
	}
	if c.DBURL != "" {
		if len(c.DBURLs) != 0 {
			return errors.New("mongodb can not config both 'DBURL' and 'DBURLs'")
		}
		c.DBURLs = splitStringByBlankOrComma(c.DBURL)
	}
	return nil
}

// CheckConfig check dcrm config
func (c *DcrmConfig) CheckConfig(isServer bool) (err error) {
	if c.Disable {
		return nil
	}
	if c.GroupID == nil {
		return errors.New("dcrm must config 'GroupID'")
	}
	if c.NeededOracles == nil {
		return errors.New("dcrm must config 'NeededOracles'")
	}
	if c.TotalOracles == nil {
		return errors.New("dcrm must config 'TotalOracles'")
	}
	if !(c.Mode == 0 || c.Mode == 1) {
		return errors.New("dcrm must config 'Mode' to 0 (managed) or 1 (private)")
	}
	if len(c.Initiators) == 0 {
		return errors.New("dcrm must config 'Initiators'")
	}
	if c.DefaultNode == nil {
		return errors.New("dcrm must config 'DefaultNode'")
	}
	err = c.DefaultNode.CheckConfig(isServer)
	if err != nil {
		return err
	}
	for _, dcrmNode := range c.OtherNodes {
		err = dcrmNode.CheckConfig(isServer)
		if err != nil {
			return err
		}
	}
	return nil
}

// CheckConfig check dcrm node config
func (c *DcrmNodeConfig) CheckConfig(isServer bool) (err error) {
	if c.RPCAddress == nil || *c.RPCAddress == "" {
		return errors.New("dcrm node must config 'RPCAddress'")
	}
	if c.KeystoreFile == nil || *c.KeystoreFile == "" {
		return errors.New("dcrm node must config 'KeystoreFile'")
	}
	if c.PasswordFile == nil {
		return errors.New("dcrm node must config 'PasswordFile'")
	}
	if isServer && len(c.SignGroups) == 0 {
		return errors.New("swap server dcrm node must config 'SignGroups'")
	}
	return nil
}

// CheckConfig check oracle config
func (c *OracleConfig) CheckConfig() (err error) {
	if c == nil {
		return errors.New("oracle must config 'Oracle'")
	}
	ServerAPIAddress = c.ServerAPIAddress
	if ServerAPIAddress == "" {
		return errors.New("oracle must config 'ServerAPIAddress'")
	}
	var version string
	for i := 0; i < 3; i++ {
		err = client.RPCPostWithTimeout(60, &version, ServerAPIAddress, "swap.GetVersionInfo")
		if err == nil {
			log.Info("oracle get server version info succeed", "version", version)
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Warn("oracle connect ServerAPIAddress failed", "ServerAPIAddress", ServerAPIAddress, "err", err)
	}
	return err
}

// CheckConfig extra config
func (c *ExtraConfig) CheckConfig() (err error) {
	return nil
}
