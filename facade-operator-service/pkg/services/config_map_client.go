package services

import (
	"context"
	"fmt"
	customerrors "github.com/netcracker/qubership-core-facade-operator/pkg/errors"
	"github.com/netcracker/qubership-core-facade-operator/pkg/utils"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const CoreGatewayImageConfigMap = "core-gateway-image"

type ConfigMapClient interface {
	Apply(ctx context.Context, req ctrl.Request, configMap *corev1.ConfigMap) error
	Delete(ctx context.Context, req ctrl.Request, name string) error
	GetGatewayImage(ctx context.Context, req ctrl.Request) (string, error)
}

type ConfigMapClientImpl struct {
	client client.Client
	logger logging.Logger
}

func NewConfigMapClient(client client.Client) *ConfigMapClientImpl {
	return &ConfigMapClientImpl{
		client: client,
		logger: logging.GetLogger("ConfigMapClient"),
	}
}

func (r *ConfigMapClientImpl) GetGatewayImage(ctx context.Context, req ctrl.Request) (string, error) {
	foundConfigMap := &corev1.ConfigMap{}

	nameSpacedRequest := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      CoreGatewayImageConfigMap,
	}
	err := r.client.Get(ctx, nameSpacedRequest, foundConfigMap, &client.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return "", nil
		} else {
			return "", errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to get ConfigMap %v", CoreGatewayImageConfigMap), err)
		}
	}
	return foundConfigMap.Data["image"], nil
}

func (r *ConfigMapClientImpl) Apply(ctx context.Context, req ctrl.Request, configMap *corev1.ConfigMap) error {
	foundConfigMap := &corev1.ConfigMap{}

	nameSpacedRequest := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      configMap.Name,
	}
	found := true
	err := r.client.Get(ctx, nameSpacedRequest, foundConfigMap, &client.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			found = false
		} else {
			return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to get ConfigMap %v", configMap.Name), err)
		}
	}

	if found {
		r.logger.InfoC(ctx, "[%v] Update ConfigMap %v", req.NamespacedName, configMap.Name)
		configMap.ObjectMeta.ResourceVersion = foundConfigMap.ObjectMeta.ResourceVersion
		options := &client.UpdateOptions{
			FieldManager: "facadeOperator",
		}
		configMap.ObjectMeta.OwnerReferences = utils.MergeOwnerReferences(configMap.ObjectMeta.OwnerReferences, foundConfigMap.ObjectMeta.OwnerReferences)
		err = r.client.Update(ctx, configMap, options)
		if err != nil {
			if k8sErrors.IsConflict(err) {
				r.logger.Debugf("[%v] ConfigMap %v already updated. Error: %v", req.NamespacedName, configMap.Name, err)
				return &customerrors.ExpectedError{Message: err.Error()}
			}
			return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to update ConfigMap %v", configMap.Name), err)
		}
	} else {
		r.logger.InfoC(ctx, "[%v] Create ConfigMap %v", req.NamespacedName, configMap.Name)
		options := &client.CreateOptions{
			FieldManager: "facadeOperator",
		}
		err = r.client.Create(ctx, configMap, options)
		if err != nil {
			if k8sErrors.IsAlreadyExists(err) {
				r.logger.Debugf("[%v] ConfigMap %v already created. Error: %v", req.NamespacedName, configMap.Name, err)
				return &customerrors.ExpectedError{Message: err.Error()}
			}
			return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to create ConfigMap %v", configMap.Name), err)
		}
	}

	return nil
}

func (r *ConfigMapClientImpl) Delete(ctx context.Context, req ctrl.Request, configMapName string) error {
	r.logger.InfoC(ctx, "[%v] Start delete ConfigMap %v", req.NamespacedName, configMapName)

	foundConfigMap := &corev1.ConfigMap{}
	nameSpacedRequest := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      configMapName,
	}
	err := r.client.Get(ctx, nameSpacedRequest, foundConfigMap)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.logger.InfoC(ctx, "[%v] ConfigMap %v not found", req.NamespacedName, configMapName)
			return nil
		}
		return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to get ConfigMap %v", configMapName), err)
	}

	err = r.client.Delete(ctx, foundConfigMap)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.logger.Debugf("[%v] ConfigMap %v already deleted. Error: %v", req.NamespacedName, configMapName, err)
			return nil
		}
		return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to delete ConfigMap %v", configMapName), err)
	}
	r.logger.InfoC(ctx, "[%v] ConfigMap %v deleted", req.NamespacedName, configMapName)

	return nil
}
