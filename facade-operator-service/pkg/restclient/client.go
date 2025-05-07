package restclient

import (
	"bytes"
	"context"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	"io"
	"net/http"
)

type RestClient interface {
	DoRequest(context.Context, string, string, *bytes.Reader) (*Response, error)
}

type SimpleRestClient struct {
	logger logging.Logger
}

func NewSimpleRestClient() *SimpleRestClient {
	return &SimpleRestClient{logger: logging.GetLogger("SimpleRestClient")}
}

func (c SimpleRestClient) DoRequest(_ context.Context, method string, url string, body *bytes.Reader) (*Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		c.logger.Error("can not create request: %w", err)
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.logger.Error("can not perform request: %w", err)
		return nil, err
	}
	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		c.logger.Error("can not read response body: %w", err)
		return nil, err
	}
	return &Response{
		StatusCode: res.StatusCode,
		Body:       respBody,
	}, nil
}

type Response struct {
	StatusCode int
	Body       []byte
}
