package params

import (
	"errors"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
)

// CheckConfig check config
func CheckConfig(isServer bool) (err error) {
	config := GetConfig()
	if config.Identifier == "" {
		return errors.New("server must config non empty 'Identifier'")
	}
	if isServer {
		if config.MongoDB == nil {
			return errors.New("server must config 'MongoDB'")
		}
		if config.APIServer == nil {
			return errors.New("server must config 'APIServer'")
		}
	} else {
		if config.Oracle == nil {
			return errors.New("oracle must config 'Oracle'")
		}
		err = config.Oracle.CheckConfig()
		if err != nil {
			return err
		}
	}
	err = checkTokenConfig()
	if err != nil {
		return err
	}
	if config.Dcrm == nil {
		return errors.New("server must config 'Dcrm'")
	}
	err = config.Dcrm.CheckConfig(isServer)
	if err != nil {
		return err
	}
	return nil
}

func checkTokenConfig() (err error) {
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
	err = config.SrcChain.CheckConfig()
	if err != nil {
		return err
	}
	err = config.DestChain.CheckConfig()
	if err != nil {
		return err
	}
	return nil
}

// CheckConfig check dcrm config
func (c *DcrmConfig) CheckConfig(isServer bool) (err error) {
	if c.Disable {
		return nil
	}
	if c.RPCAddress == nil {
		return errors.New("dcrm must config 'RPCAddress'")
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
	if c.Mode != 0 {
		return errors.New("dcrm must config 'Mode' to 0 (managed)")
	}
	if c.ServerAccount == "" {
		return errors.New("dcrm must config 'ServerAccount'")
	}
	if isServer {
		if c.Pubkey == nil {
			return errors.New("swap server dcrm must config 'Pubkey'")
		}
		if len(c.SignGroups) == 0 {
			return errors.New("swap server dcrm must config 'SignGroups'")
		}
	}
	if c.KeystoreFile == nil {
		return errors.New("dcrm must config 'KeystoreFile'")
	}
	if c.PasswordFile == nil {
		return errors.New("dcrm must config 'PasswordFile'")
	}
	return nil
}

// CheckConfig check oracle config
func (c *OracleConfig) CheckConfig() (err error) {
	ServerAPIAddress = c.ServerAPIAddress
	if ServerAPIAddress == "" {
		return errors.New("oracle must config 'ServerAPIAddress'")
	}
	var version string
	for {
		err = client.RPCPost(&version, ServerAPIAddress, "swap.GetVersionInfo")
		if err == nil {
			log.Info("oracle get server version info succeed", "version", version)
			break
		}
		log.Warn("oracle connect ServerAPIAddress failed", "ServerAPIAddress", ServerAPIAddress, "err", err)
		time.Sleep(3 * time.Second)
	}
	return err
}
