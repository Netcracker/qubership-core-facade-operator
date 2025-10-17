package services

import (
	"context"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	facadeV1 "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade/v1"
	facadeV1Alpha "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade/v1alpha"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	mock_services "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/test/mock/services"
	"go.uber.org/mock/gomock"
	"testing"

	"github.com/stretchr/testify/assert"
	appsV1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type testCRPriorityTestStruct struct {
	name           string
	errorDetail    string
	cr             facade.MeshGateway
	expectedResult bool
	mockFunc       func()
}

func TestUpdateAvailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deploymentClient := GetMockDeploymentClient(ctrl)
	commonCRClient := GetMockCommonCRClient(ctrl)
	crPriorityService := GetCRPriorityService(deploymentClient, commonCRClient)

	ctx := context.Background()
	req := getRequest()
	gatewayName := "gatewayName"

	//gateway
	meshCR := &facadeV1.Gateway{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Gateway",
			APIVersion: "core.netcracker.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "name",
		},
		Spec: facade.FacadeServiceSpec{},
	}
	meshLastAppliedCr := &utils.LastAppliedCr{
		ApiVersion: meshCR.GetAPIVersion(),
		Kind:       meshCR.GetKind(),
		Name:       meshCR.GetName(),
	}
	meshLastAppliedCrStr, _ := utils.JsonMarshal(meshLastAppliedCr)

	//gateway master
	meshMasterCR := &facadeV1.Gateway{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Gateway",
			APIVersion: "core.netcracker.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "name",
		},
		Spec: facade.FacadeServiceSpec{
			MasterConfiguration: true,
		},
	}
	meshMasterCRSecond := &facadeV1.Gateway{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Gateway",
			APIVersion: "core.netcracker.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "name2",
		},
		Spec: facade.FacadeServiceSpec{
			MasterConfiguration: true,
		},
	}
	meshMasterLastAppliedCr := &utils.LastAppliedCr{
		ApiVersion: meshMasterCR.GetAPIVersion(),
		Kind:       meshMasterCR.GetKind(),
		Name:       meshMasterCR.GetName(),
	}
	meshMasterLastAppliedCrStr, _ := utils.JsonMarshal(meshMasterLastAppliedCr)

	//facade
	facadeCR := &facadeV1Alpha.FacadeService{
		TypeMeta: metav1.TypeMeta{
			Kind:       "FacadeService",
			APIVersion: "netcracker.com/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "name",
		},
		Spec: facade.FacadeServiceSpec{},
	}
	facadeLastAppliedCr := &utils.LastAppliedCr{
		ApiVersion: facadeCR.GetAPIVersion(),
		Kind:       facadeCR.GetKind(),
		Name:       facadeCR.GetName(),
	}
	facadeLastAppliedCrStr, _ := utils.JsonMarshal(facadeLastAppliedCr)

	//facade master
	facadeMasterCR := &facadeV1Alpha.FacadeService{
		TypeMeta: metav1.TypeMeta{
			Kind:       "FacadeService",
			APIVersion: "netcracker.com/v1alpha",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "name",
		},
		Spec: facade.FacadeServiceSpec{
			MasterConfiguration: true,
		},
	}
	facadeMasterLastAppliedCr := &utils.LastAppliedCr{
		ApiVersion: facadeMasterCR.GetAPIVersion(),
		Kind:       facadeMasterCR.GetKind(),
		Name:       facadeMasterCR.GetName(),
	}
	facadeMasterLastAppliedCrStr, _ := utils.JsonMarshal(facadeMasterLastAppliedCr)

	tests := []testCRPriorityTestStruct{
		{
			name:           "FailWhenGetDeploymentFail",
			cr:             meshCR,
			expectedResult: false,
			mockFunc: func() {
				deploymentClient.EXPECT().Get(ctx, req, gatewayName).Return(nil, getUnknownError())
			},
			errorDetail: "UnknownError",
		},
		{
			name:           "NonMasterGatewayToNonMasterFacade",
			cr:             facadeCR,
			expectedResult: false,
			mockFunc: func() {
				nonMasterDeployment := &appsV1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Labels:      map[string]string{},
						Annotations: map[string]string{utils.LastAppliedCRAnnotation: meshLastAppliedCrStr},
					},
				}
				deploymentClient.EXPECT().Get(ctx, req, gatewayName).Return(nonMasterDeployment, nil)
				commonCRClient.EXPECT().GetByLastAppliedCr(ctx, req, meshLastAppliedCr).Return(meshCR, nil)
			},
		},
		{
			name:           "NonMasterFacadeToNonMasterFacade",
			cr:             facadeCR,
			expectedResult: true,
			mockFunc: func() {
				nonMasterDeployment := &appsV1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Labels:      map[string]string{},
						Annotations: map[string]string{utils.LastAppliedCRAnnotation: facadeLastAppliedCrStr},
					},
				}
				deploymentClient.EXPECT().Get(ctx, req, gatewayName).Return(nonMasterDeployment, nil)
				commonCRClient.EXPECT().GetByLastAppliedCr(ctx, req, facadeLastAppliedCr).Return(facadeCR, nil)
			},
		},
		{
			name:           "NonMasterFacadeToNonMasterGateway",
			cr:             meshCR,
			expectedResult: true,
			mockFunc: func() {
				nonMasterDeployment := &appsV1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Labels:      map[string]string{},
						Annotations: map[string]string{utils.LastAppliedCRAnnotation: facadeLastAppliedCrStr},
					},
				}
				deploymentClient.EXPECT().Get(ctx, req, gatewayName).Return(nonMasterDeployment, nil)
				commonCRClient.EXPECT().GetByLastAppliedCr(ctx, req, facadeLastAppliedCr).Return(facadeCR, nil)
			},
		},
		{
			name:           "MasterGatewayToMasterFacade",
			cr:             facadeMasterCR,
			expectedResult: false,
			mockFunc: func() {
				masterDeployment := &appsV1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Labels:      map[string]string{utils.MasterCR: "name"},
						Annotations: map[string]string{utils.LastAppliedCRAnnotation: meshMasterLastAppliedCrStr},
					},
				}
				deploymentClient.EXPECT().Get(ctx, req, gatewayName).Return(masterDeployment, nil)
				commonCRClient.EXPECT().GetByLastAppliedCr(ctx, req, meshMasterLastAppliedCr).Return(meshMasterCR, nil)
				commonCRClient.EXPECT().IsCRExistByName(ctx, req, "name").Return(true, nil)
			},
		},
		{
			name:           "NonMasterGatewayToMasterGateway",
			cr:             meshMasterCR,
			expectedResult: true,
			mockFunc: func() {
				masterDeployment := &appsV1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Labels:      map[string]string{},
						Annotations: map[string]string{utils.LastAppliedCRAnnotation: meshLastAppliedCrStr},
					},
				}
				deploymentClient.EXPECT().Get(ctx, req, gatewayName).Return(masterDeployment, nil)
				commonCRClient.EXPECT().GetByLastAppliedCr(ctx, req, meshLastAppliedCr).Return(meshMasterCR, nil)
			},
		},
		{
			name:           "MasterFacadeToNonMasterGateway",
			cr:             meshCR,
			expectedResult: false,
			mockFunc: func() {
				masterDeployment := &appsV1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Labels:      map[string]string{utils.MasterCR: "name"},
						Annotations: map[string]string{utils.LastAppliedCRAnnotation: facadeMasterLastAppliedCrStr},
					},
				}
				deploymentClient.EXPECT().Get(ctx, req, gatewayName).Return(masterDeployment, nil)
				commonCRClient.EXPECT().GetByLastAppliedCr(ctx, req, facadeMasterLastAppliedCr).Return(facadeMasterCR, nil)
				commonCRClient.EXPECT().IsCRExistByName(ctx, req, "name").Return(true, nil)
			},
		},
		{
			name:           "MasterFacadeToNonMasterGatewayWithDiffManes",
			cr:             meshCR,
			expectedResult: false,
			mockFunc: func() {
				masterDeployment := &appsV1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Labels:      map[string]string{utils.MasterCR: "name1"},
						Annotations: map[string]string{utils.LastAppliedCRAnnotation: facadeMasterLastAppliedCrStr},
					},
				}
				deploymentClient.EXPECT().Get(ctx, req, gatewayName).Return(masterDeployment, nil)
				commonCRClient.EXPECT().GetByLastAppliedCr(ctx, req, facadeMasterLastAppliedCr).Return(facadeMasterCR, nil)
				commonCRClient.EXPECT().IsCRExistByName(ctx, req, "name1").Return(true, nil)
			},
		},
		{
			name:           "MasterFacadeToMasterGateway",
			cr:             meshMasterCR,
			expectedResult: true,
			mockFunc: func() {
				masterDeployment := &appsV1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Labels:      map[string]string{utils.MasterCR: "name"},
						Annotations: map[string]string{utils.LastAppliedCRAnnotation: facadeMasterLastAppliedCrStr},
					},
				}
				deploymentClient.EXPECT().Get(ctx, req, gatewayName).Return(masterDeployment, nil)
				commonCRClient.EXPECT().GetByLastAppliedCr(ctx, req, facadeMasterLastAppliedCr).Return(facadeMasterCR, nil)
				commonCRClient.EXPECT().IsCRExistByName(ctx, req, "name").Return(true, nil)
			},
		},
		{
			name:           "MasterCRNotFound",
			cr:             meshMasterCR,
			expectedResult: true,
			mockFunc: func() {
				masterDeployment := &appsV1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{utils.MasterCR: "unknown"},
					},
				}
				deploymentClient.EXPECT().Get(ctx, req, gatewayName).Return(masterDeployment, nil)
				commonCRClient.EXPECT().GetByLastAppliedCr(ctx, req, gomock.Any()).Return(nil, nil)
				commonCRClient.EXPECT().IsCRExistByName(ctx, req, "unknown").Return(false, nil)
			},
		},
		{
			name:           "MasterCRFoundButCurrentCRNotMaster",
			cr:             facadeCR,
			expectedResult: false,
			mockFunc: func() {
				masterDeployment := &appsV1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Labels:      map[string]string{utils.MasterCR: "unknown"},
						Annotations: map[string]string{utils.LastAppliedCRAnnotation: meshMasterLastAppliedCrStr},
					},
				}
				deploymentClient.EXPECT().Get(ctx, req, gatewayName).Return(masterDeployment, nil)
				commonCRClient.EXPECT().GetByLastAppliedCr(ctx, req, meshMasterLastAppliedCr).Return(meshMasterCR, nil)
				commonCRClient.EXPECT().IsCRExistByName(ctx, req, "unknown").Return(true, nil)
			},
		},
		{
			name:           "EqualPriorityButDiffMasterNames",
			cr:             meshMasterCR,
			expectedResult: false,
			mockFunc: func() {
				masterDeployment := &appsV1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Labels:      map[string]string{utils.MasterCR: "unknown"},
						Annotations: map[string]string{utils.LastAppliedCRAnnotation: meshMasterLastAppliedCrStr},
					},
				}
				deploymentClient.EXPECT().Get(ctx, req, gatewayName).Return(masterDeployment, nil)
				commonCRClient.EXPECT().GetByLastAppliedCr(ctx, req, meshMasterLastAppliedCr).Return(meshMasterCRSecond, nil)
				commonCRClient.EXPECT().IsCRExistByName(ctx, req, "unknown").Return(true, nil)
			},
		},
		{
			name:           "NewCRPriorityMoreThenLastMasterCR",
			cr:             meshMasterCRSecond,
			expectedResult: true,
			mockFunc: func() {
				masterDeployment := &appsV1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Labels:      map[string]string{utils.MasterCR: "unknown"},
						Annotations: map[string]string{utils.LastAppliedCRAnnotation: facadeMasterLastAppliedCrStr},
					},
				}
				deploymentClient.EXPECT().Get(ctx, req, gatewayName).Return(masterDeployment, nil)
				commonCRClient.EXPECT().GetByLastAppliedCr(ctx, req, facadeMasterLastAppliedCr).Return(facadeMasterCR, nil)
				commonCRClient.EXPECT().IsCRExistByName(ctx, req, "unknown").Return(true, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name != "NonMasterGatewayToMasterGateway" {
				return
			}
			if tt.mockFunc != nil {
				tt.mockFunc()
			}

			result, err := crPriorityService.UpdateAvailable(ctx, req, gatewayName, tt.cr)
			if tt.errorDetail != "" {
				assert.NotNil(t, err)
				assert.Equal(t, tt.errorDetail, err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func GetMockDeploymentClient(ctrl *gomock.Controller) *mock_services.MockDeploymentClient {
	return mock_services.NewMockDeploymentClient(ctrl)
}

func GetCRPriorityService(deploymentsClient DeploymentClient, commonCRClient CommonCRClient) CRPriorityService {
	return NewCRPriorityService(deploymentsClient, commonCRClient)
}
