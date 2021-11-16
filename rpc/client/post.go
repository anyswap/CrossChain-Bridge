package client

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/anyswap/CrossChain-Bridge/log"
)

const (
	defaultSlowTimeout = 60 // seconds
	defaultTimeout     = 5  // seconds
	defaultRequestID   = 1
)

// Request json rpc request
type Request struct {
	Method  string
	Params  interface{}
	Timeout int
	ID      int
}

// NewRequest new request
func NewRequest(method string, params ...interface{}) *Request {
	return &Request{
		Method:  method,
		Params:  params,
		Timeout: defaultTimeout,
		ID:      defaultRequestID,
	}
}

// NewRequestWithTimeoutAndID new request with timeout and id
func NewRequestWithTimeoutAndID(timeout, id int, method string, params ...interface{}) *Request {
	return &Request{
		Method:  method,
		Params:  params,
		Timeout: timeout,
		ID:      id,
	}
}

// RPCPost rpc post
func RPCPost(result interface{}, url, method string, params ...interface{}) error {
	req := NewRequest(method, params...)
	return RPCPostRequest(url, req, result)
}

// RPCPostWithTimeout rpc post with timeout
func RPCPostWithTimeout(timeout int, result interface{}, url, method string, params ...interface{}) error {
	req := NewRequestWithTimeoutAndID(timeout, defaultRequestID, method, params...)
	return RPCPostRequest(url, req, result)
}

// RPCPostWithTimeoutAndID rpc post with timeout and id
func RPCPostWithTimeoutAndID(result interface{}, timeout, id int, url, method string, params ...interface{}) error {
	req := NewRequestWithTimeoutAndID(timeout, id, method, params...)
	return RPCPostRequest(url, req, result)
}

// RequestBody request body
type RequestBody struct {
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
}

type jsonError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (err *jsonError) Error() string {
	return fmt.Sprintf("json-rpc error %d, %s", err.Code, err.Message)
}

type jsonrpcResponse struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Error   *jsonError      `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

// RPCPostRequest rpc post request
func RPCPostRequest(url string, req *Request, result interface{}) error {
	reqBody := &RequestBody{
		Version: "2.0",
		Method:  req.Method,
		Params:  req.Params,
		ID:      req.ID,
	}
	resp, err := HTTPPost(url, reqBody, nil, nil, req.Timeout)
	if err != nil {
		log.Trace("post rpc error", "url", url, "request", req, "err", err)
		return err
	}
	err = getResultFromJSONResponse(result, resp)
	if err != nil {
		log.Trace("post rpc error", "url", url, "request", req, "err", err)
	}
	return err
}

func getResultFromJSONResponse(result interface{}, resp *http.Response) error {
	defer func() {
		_ = resp.Body.Close()
	}()
	const maxReadContentLength int64 = 1024 * 1024 * 10 // 10M
	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, maxReadContentLength))
	if err != nil {
		return fmt.Errorf("read body error: %w", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("wrong response status %v. message: %v", resp.StatusCode, string(body))
	}
	if len(body) == 0 {
		return fmt.Errorf("empty response body")
	}

	var jsonResp jsonrpcResponse
	err = json.Unmarshal(body, &jsonResp)
	if err != nil {
		return fmt.Errorf("unmarshal body error, body is \"%v\" err=\"%w\"", string(body), err)
	}
	if jsonResp.Error != nil {
		return fmt.Errorf("return error: %w", jsonResp.Error)
	}
	err = json.Unmarshal(jsonResp.Result, &result)
	if err != nil {
		return fmt.Errorf("unmarshal result error: %w", err)
	}
	return nil
}

// RPCRawPost rpc raw post
func RPCRawPost(url, body string) (string, error) {
	return RPCRawPostWithTimeout(url, body, defaultSlowTimeout)
}

// RPCRawPostWithTimeout rpc raw post with timeout
func RPCRawPostWithTimeout(url, reqBody string, timeout int) (string, error) {
	resp, err := HTTPRawPost(url, reqBody, nil, nil, timeout)
	if err != nil {
		return "", err
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
		return "", fmt.Errorf("wrong response status %v. message: %v", resp.StatusCode, string(body))
	}
	return string(body), nil
}
