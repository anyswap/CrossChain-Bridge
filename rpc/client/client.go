// Package client provides methods to do http GET / POST request.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"
)

var (
	httpClient *http.Client
	httpCtx    = context.Background()
)

// InitHTTPClient init http client
func InitHTTPClient() {
	httpClient = createHTTPClient()
}

const (
	maxIdleConns        int = 100
	maxIdleConnsPerHost int = 10
	maxConnsPerHost     int = 50
	idleConnTimeout     int = 90
)

// createHTTPClient for connection re-use
func createHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxConnsPerHost:     maxConnsPerHost,
			MaxIdleConns:        maxIdleConns,
			MaxIdleConnsPerHost: maxIdleConnsPerHost,
			IdleConnTimeout:     time.Duration(idleConnTimeout) * time.Second,
		},
		Timeout: defaultTimeout * time.Second,
	}
}

// HTTPGet http get
func HTTPGet(url string, params, headers map[string]string, timeout int) (*http.Response, error) {
	req, err := http.NewRequestWithContext(httpCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	addParams(req, params)
	addHeaders(req, headers)

	return doRequest(req, timeout)
}

// HTTPPost http post
func HTTPPost(url string, body interface{}, params, headers map[string]string, timeout int) (*http.Response, error) {
	req, err := http.NewRequestWithContext(httpCtx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}

	addParams(req, params)
	addHeaders(req, headers)
	if err := addPostBody(req, body); err != nil {
		return nil, err
	}

	return doRequest(req, timeout)
}

// HTTPRawPost http raw post
func HTTPRawPost(url, body string, params, headers map[string]string, timeout int) (*http.Response, error) {
	req, err := http.NewRequestWithContext(httpCtx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}

	addParams(req, params)
	addHeaders(req, headers)
	if err := addRawPostBody(req, body); err != nil {
		return nil, err
	}

	return doRequest(req, timeout)
}

func addParams(req *http.Request, params map[string]string) {
	if params != nil {
		q := req.URL.Query()
		for key, val := range params {
			q.Add(key, val)
		}
		req.URL.RawQuery = q.Encode()
	}
}

func addHeaders(req *http.Request, headers map[string]string) {
	for key, val := range headers {
		req.Header.Add(key, val)
	}
}

func addPostBody(req *http.Request, body interface{}) error {
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return err
		}
		req.Header.Set("Content-type", "application/json")
		req.GetBody = func() (io.ReadCloser, error) {
			return ioutil.NopCloser(bytes.NewBuffer(jsonData)), nil
		}
		req.Body, _ = req.GetBody()
	}
	return nil
}

func addRawPostBody(req *http.Request, body string) (err error) {
	if body == "" {
		return nil
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.GetBody = func() (io.ReadCloser, error) {
		return ioutil.NopCloser(strings.NewReader(body)), nil
	}
	req.Body, err = req.GetBody()
	return err
}

func doRequest(req *http.Request, timeoutSeconds int) (*http.Response, error) {
	timeout := time.Duration(timeoutSeconds) * time.Second
	if httpClient == nil {
		client := http.Client{
			Timeout: timeout,
		}
		return client.Do(req)
	}
	httpClient.Timeout = timeout
	return httpClient.Do(req)
}
