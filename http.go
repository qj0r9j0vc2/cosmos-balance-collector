package main

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/xlab/suplog"
	"io"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	ABCI_QUERY_PATH = "/abci_query"
	BLOCK_PATH      = "/block"
	STATUS_PATH     = "/status"
)

type HTTPClient struct {
	*http.Client
	url     string
	timeout time.Duration
}

func NewHTTPClient(url string, timeout int) (*HTTPClient, error) {
	return &HTTPClient{
		Client:  &http.Client{},
		url:     url,
		timeout: time.Duration(timeout) * time.Second,
	}, nil
}

func (c *HTTPClient) Query(path string, parameters map[string]string) ([]byte, error) {

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	var params string
	if strings.Contains(c.url, "?") {
		params = "&"
	} else {
		params = "?"
	}

	for k, v := range parameters {
		params += "&"
		params += fmt.Sprintf("%s=%s", k, v)
	}

	//log.Debugf(c.url + path + params)
	req, err := requestGet(ctx, c.url+path+params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to request")
	}

	var body []byte
	body, err = request(c.Client, req, 5)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func requestGet(ctx context.Context, url string) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
}

func request(c *http.Client, request *http.Request, retries int) ([]byte, error) {
	var errMsg string
	for i := 0; i < retries; i++ {
		res, err := c.Do(request)
		if err != nil {
			errMsg = errors.New("err: " + err.Error() + ", " + runtime.FuncForPC(reflect.ValueOf(request).Pointer()).Name() + ".Retries " + strconv.Itoa(i) + "...").Error()
			log.Warning(errMsg)
			time.Sleep(1 * time.Second)
			continue
		}

		body, err := io.ReadAll(res.Body)
		if err != nil {
			errMsg = errors.New("err: " + err.Error() + ", " + runtime.FuncForPC(reflect.ValueOf(request).Pointer()).Name() + ".Retries " + strconv.Itoa(i) + "...").Error()
			log.Warning(errMsg)
			time.Sleep(1 * time.Second)
			continue
		}
		defer res.Body.Close()

		return body, nil
	}

	return nil, errors.New(errMsg)
}
