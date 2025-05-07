package restclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/netcracker/qubership-core-facade-operator/api/facade"
	customerrors "github.com/netcracker/qubership-core-facade-operator/pkg/errors"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	"github.com/netcracker/qubership-core-lib-go/v3/serviceloader"
	"net/http"
)

const gatewaysSpecsApi = "/api/v3/gateways/specs"

type ControlPlaneClient interface {
	RegisterGateway(ctx context.Context, gatewayServiceName string, cr facade.MeshGateway) error
	DropGateway(ctx context.Context, gatewayServiceName string) error
}

type GatewayDeclaration struct {
	Name              string             `json:"name"`
	GatewayType       facade.GatewayType `json:"gatewayType"`
	AllowVirtualHosts *bool              `json:"allowVirtualHosts"`
	Exists            *bool              `json:"exists,omitempty"`
}

type cpClient struct {
	logger          logging.Logger
	controlPlaneUrl string
	m2mClient       RestClient
}

func NewControlPlaneClient() ControlPlaneClient {
	controlPlaneUrl := "http://control-plane:8080"
	return &cpClient{
		logger:          logging.GetLogger("ControlPlaneClient"),
		controlPlaneUrl: controlPlaneUrl,
		m2mClient:       serviceloader.MustLoad[RestClient]()}
}

func (c *cpClient) sendRequest(ctx context.Context, method, path string, requestBody any) (*Response, error) {
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, errs.NewError(customerrors.UnknownErrorCode, "could not serialize gateway registration request body", err)
	}
	resp, err := c.m2mClient.DoRequest(ctx, method, c.controlPlaneUrl+path, bytes.NewReader(body))
	if err != nil {
		return nil, errs.NewError(customerrors.ControlPlaneError, fmt.Sprintf("%s request to control-plane %s failed with error", method, path), err)
	}
	return resp, nil
}

func (c *cpClient) RegisterGateway(ctx context.Context, gatewayServiceName string, cr facade.MeshGateway) error {
	requestDto := GatewayDeclaration{
		Name:              gatewayServiceName,
		GatewayType:       cr.GetGatewayType(),
		AllowVirtualHosts: cr.GetSpec().AllowVirtualHosts,
	}
	c.logger.InfoC(ctx, "Sending gateway registration request to control-plane %+v", requestDto)
	resp, err := c.sendRequest(ctx, http.MethodPost, gatewaysSpecsApi, requestDto)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errs.NewError(customerrors.ControlPlaneError,
			fmt.Sprintf("gateway registration request got %d error from control-plane with message '%s'", resp.StatusCode, stringifyBody(resp.Body)), nil)
	}
	c.logger.InfoC(ctx, "Gateway %+v successfully registered", requestDto)
	return nil
}

func (c *cpClient) DropGateway(ctx context.Context, gatewayServiceName string) error {
	requestDto := GatewayDeclaration{
		Name: gatewayServiceName,
	}
	c.logger.InfoC(ctx, "Sending gateway %s drop request to control-plane", gatewayServiceName)
	resp, err := c.sendRequest(ctx, http.MethodDelete, gatewaysSpecsApi, requestDto)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusBadRequest {
			// it is OK if we could not delete gateway declaration - there can still be some routes associated with this node group
			c.logger.InfoC(ctx, "Gateway %s will not be dropped in control-plane because this node group still has some associated entities")
			return nil
		}
		return errs.NewError(customerrors.ControlPlaneError,
			fmt.Sprintf("gateway registration request got %d error from control-plane with message '%s'", resp.StatusCode, stringifyBody(resp.Body)), nil)
	}
	c.logger.InfoC(ctx, "Gateway %s dropped successfully", gatewayServiceName)
	return nil
}

func stringifyBody(body []byte) string {
	if body == nil {
		return ""
	}
	return string(body)
}
