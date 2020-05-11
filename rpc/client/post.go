package client

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

const (
	defaultTimeout   = 60 // seconds
	defaultRequestID = 1
)

type Request struct {
	Method  string
	Params  interface{}
	Timeout int
	ID      int
}

func NewRequest(method string, params ...interface{}) *Request {
	return &Request{
		Method:  method,
		Params:  params,
		Timeout: defaultTimeout,
		ID:      defaultRequestID,
	}
}

func NewRequestWithTimeoutAndID(timeout, id int, method string, params ...interface{}) *Request {
	return &Request{
		Method:  method,
		Params:  params,
		Timeout: timeout,
		ID:      id,
	}
}

func RpcPost(result interface{}, url string, method string, params ...interface{}) error {
	req := NewRequest(method, params...)
	return RpcPostRequest(url, req, result)
}

func RpcPostWithTimeoutAndID(result interface{}, timeout, id int, url string, method string, params ...interface{}) error {
	req := NewRequestWithTimeoutAndID(timeout, id, method, params...)
	return RpcPostRequest(url, req, result)
}

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

func RpcPostRequest(url string, req *Request, result interface{}) error {
	reqBody := &RequestBody{
		Version: "2.0",
		Method:  req.Method,
		Params:  req.Params,
		ID:      req.ID,
	}
	resp, err := HttpPost(url, reqBody, nil, nil, req.Timeout)
	if err != nil {
		return err
	}
	return getResultFromJsonResponse(result, resp)
}

func getResultFromJsonResponse(result interface{}, resp *http.Response) error {
	defer resp.Body.Close()
	const maxReadContentLength int64 = 1024 * 1024 * 10 // 10M
	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, maxReadContentLength))
	if err != nil {
		return fmt.Errorf("read body error: %v", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("wrong response status %v. message: %v", resp.StatusCode, string(body))
	}

	var jsonResp jsonrpcResponse
	err = json.Unmarshal(body, &jsonResp)
	if err != nil {
		return fmt.Errorf("unmarshal body error: %v", err)
	}
	if jsonResp.Error != nil {
		return fmt.Errorf("return error:  %v", jsonResp.Error.Error())
	}
	err = json.Unmarshal(jsonResp.Result, &result)
	if err != nil {
		return fmt.Errorf("unmarshal result error: %v", err)
	}
	return nil
}

func RpcRawPost(url string, body string) (string, error) {
	return RpcRawPostWithTimeout(url, body, defaultTimeout)
}

func RpcRawPostWithTimeout(url string, reqBody string, timeout int) (string, error) {
	resp, err := HttpRawPost(url, reqBody, nil, nil, timeout)
	if err != nil {
		return "", err
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
