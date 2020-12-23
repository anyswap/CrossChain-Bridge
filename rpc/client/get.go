package client

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
)

// RPCGet rpc get
func RPCGet(result interface{}, url string) error {
	return RPCGetRequest(result, url, nil, nil, defaultTimeout)
}

// RPCGetWithTimeout rpc get with timeout
func RPCGetWithTimeout(result interface{}, url string, timeout int) error {
	return RPCGetRequest(result, url, nil, nil, timeout)
}

// RPCGetRequest rpc get request
func RPCGetRequest(result interface{}, url string, params, headers map[string]string, timeout int) error {
	resp, err := HTTPGet(url, params, headers, timeout)
	if err != nil {
		return fmt.Errorf("GET request error: %v (url: %v, params: %v)", err, url, params)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("error response status: %v (url: %v)", resp.StatusCode, url)
	}

	const maxReadContentLength int64 = 1024 * 1024 * 10 // 10M
	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, maxReadContentLength))
	if err != nil {
		return fmt.Errorf("read body error: %v", err)
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return fmt.Errorf("unmarshal result error: %v", err)
	}
	return nil
}

// RPCRawGet rpc raw get
func RPCRawGet(url string) (string, error) {
	return RPCRawGetRequest(url, nil, nil, defaultTimeout)
}

// RPCRawGetWithTimeout rpc raw get with timeout
func RPCRawGetWithTimeout(url string, timeout int) (string, error) {
	return RPCRawGetRequest(url, nil, nil, timeout)
}

// RPCRawGetRequest rpc raw get request
func RPCRawGetRequest(url string, params, headers map[string]string, timeout int) (string, error) {
	resp, err := HTTPGet(url, params, headers, timeout)
	if err != nil {
		return "", fmt.Errorf("GET request error: %v (url: %v, params: %v)", err, url, params)
	}

	defer resp.Body.Close()
	const maxReadContentLength int64 = 1024 * 1024 * 10 // 10M
	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, maxReadContentLength))
	if err != nil {
		return "", fmt.Errorf("read body error: %v", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("wrong response status %v. message: %v", resp.StatusCode, string(body))
	}
	return string(body), nil
}
