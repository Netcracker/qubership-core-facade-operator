package restclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/netcracker/qubership-core-lib-go/v3/context-propagation/ctxhelper"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	"github.com/netcracker/qubership-core-lib-go/v3/security"
	"github.com/netcracker/qubership-core-lib-go/v3/serviceloader"
)

type SimpleRestClient struct {
	logger logging.Logger
}

func NewSimpleRestClient() *SimpleRestClient {
	return &SimpleRestClient{logger: logging.GetLogger("SimpleRestClient")}
}

func (c SimpleRestClient) DoRequest(ctx context.Context, httpMethod string, url string, body string) (*Response, error) {
	c.logger.InfoC(ctx, "Perform request: %s %s", httpMethod, url)
	req, err := http.NewRequestWithContext(ctx, httpMethod, url, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("can not create request: %w", err)
	}

	// dump context
	err = ctxhelper.AddSerializableContextData(ctx, req.Header.Add)
	if err != nil {
		return nil, fmt.Errorf("error dump context data to request: %w", err)
	}

	// add token if it needed
	m2mToken, err := serviceloader.MustLoad[security.TokenProvider]().GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting m2m token from tokenprovider: %w", err)
	}
	if m2mToken != "" {
		req.Header.Add("Authorization", "Bearer "+m2mToken)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("can not perform request: %w", err)
	}
	defer res.Body.Close()
	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("can not read response body: %w", err)
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
