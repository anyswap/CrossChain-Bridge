package client

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/anyswap/CrossChain-Bridge/log"
)

// RPCGet rpc get
func RPCGet(result interface{}, url string) error {
	return RPCGetRequest(result, url, nil, nil, defaultSlowTimeout)
}

// RPCGetWithTimeout rpc get with timeout
func RPCGetWithTimeout(result interface{}, url string, timeout int) error {
	return RPCGetRequest(result, url, nil, nil, timeout)
}

// RPCGetRequest rpc get request
func RPCGetRequest(result interface{}, url string, params, headers map[string]string, timeout int) error {
	resp, err := HTTPGet(url, params, headers, timeout)
	if err != nil {
		return fmt.Errorf("GET request error: %w (url: %v, params: %v)", err, url, params)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != 200 {
		log.Trace("get rpc status error", "url", url, "status", resp.StatusCode)
		return fmt.Errorf("error response status: %v (url: %v)", resp.StatusCode, url)
	}

	const maxReadContentLength int64 = 1024 * 1024 * 10 // 10M
	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, maxReadContentLength))
	if err != nil {
		return fmt.Errorf("read body error: %w", err)
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return fmt.Errorf("unmarshal result error: %w", err)
	}
	return nil
}

// RPCRawGet rpc raw get
func RPCRawGet(url string) (string, error) {
	return RPCRawGetRequest(url, nil, nil, defaultSlowTimeout)
}

// RPCRawGetWithTimeout rpc raw get with timeout
func RPCRawGetWithTimeout(url string, timeout int) (string, error) {
	return RPCRawGetRequest(url, nil, nil, timeout)
}

// RPCRawGetRequest rpc raw get request
func RPCRawGetRequest(url string, params, headers map[string]string, timeout int) (string, error) {
	resp, err := HTTPGet(url, params, headers, timeout)
	if err != nil {
		return "", fmt.Errorf("GET request error: %w (url: %v, params: %v)", err, url, params)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	const maxReadContentLength int64 = 1024 * 1024 * 10 // 10M
	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, maxReadContentLength))
	if err != nil {
		return "", fmt.Errorf("read body error: %w", err)
	}

	if resp.StatusCode != 200 {
		log.Trace("get rpc status error", "url", url, "status", resp.StatusCode)
		return "", fmt.Errorf("wrong response status %v. message: %v", resp.StatusCode, string(body))
	}
	return string(body), nil
}
