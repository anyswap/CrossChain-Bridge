package client

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
)

func RpcGet(result interface{}, url string) error {
	return RpcGetRequest(result, url, nil, nil, defaultTimeout)
}

func RpcGetWithTimeout(result interface{}, url string, timeout int) error {
	return RpcGetRequest(result, url, nil, nil, timeout)
}

func RpcGetRequest(result interface{}, url string, params, headers map[string]string, timeout int) error {
	resp, err := HttpGet(url, params, headers, timeout)
	if err != nil {
		return fmt.Errorf("GET request error: %v (url: %v, params: %v)", err, url, params)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("error response status: %v (url: %v)", resp.StatusCode, url)
	}

	defer resp.Body.Close()
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

func RpcRawGet(url string) (string, error) {
	return RpcRawGetRequest(url, nil, nil, defaultTimeout)
}

func RpcRawGetWithTimeout(url string, timeout int) (string, error) {
	return RpcRawGetRequest(url, nil, nil, timeout)
}

func RpcRawGetRequest(url string, params, headers map[string]string, timeout int) (string, error) {
	resp, err := HttpGet(url, params, headers, timeout)
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
