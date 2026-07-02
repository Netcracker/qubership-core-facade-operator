package restclient

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/netcracker/qubership-core-lib-go/v3/context-propagation/ctxhelper"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	"github.com/netcracker/qubership-core-lib-go/v3/security/rest"
)

type SimpleRestClient struct {
	logger logging.Logger
	client *rest.M2MRestClient
}

func NewSimpleRestClient() *SimpleRestClient {
	return &SimpleRestClient{
		logger: logging.GetLogger("SimpleRestClient"),
		client: rest.NewM2MRestClient(),
	}
}

func (c *SimpleRestClient) DoRequest(ctx context.Context, httpMethod string, url string, body string) (*Response, error) {
	c.logger.InfoC(ctx, "Perform request: %s %s", httpMethod, url)

	// dump context
	headers := map[string][]string{}
	err := ctxhelper.AddSerializableContextData(ctx, func(name, vals string) {
		headers[name] = []string{vals}
	})
	if err != nil {
		return nil, fmt.Errorf("error dump context data to request: %w", err)
	}

	res, err := c.client.DoRequest(ctx, httpMethod, url, headers, strings.NewReader(body))
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
