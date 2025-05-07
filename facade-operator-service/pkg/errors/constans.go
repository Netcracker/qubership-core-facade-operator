package customerrors

import (
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
)

// Reserved for features errors (2000-2999)
var UnknownErrorCode = errs.ErrorCode{Code: "CORE-MESH-OP-2000", Title: "Unexpected exception"}
var UnexpectedKubernetesError = errs.ErrorCode{Code: "CORE-MESH-OP-2001", Title: "Kubernetes API call failed while facade gateway processing"}
var UpdateImageUnexpectedKubernetesError = errs.ErrorCode{Code: "CORE-MESH-OP-2002", Title: "Kubernetes API call failed while update image processing"}
var InitParamsValidationError = errs.ErrorCode{Code: "CORE-MESH-OP-2003", Title: "Validation of default parameters failed"}
var GatewayImageError = errs.ErrorCode{Code: "CORE-MESH-OP-2004", Title: "Can not get gateway image"}
var TlsOperationError = errs.ErrorCode{Code: "CORE-MESH-OP-2005", Title: "Unexpected error while tls processing"}
var ControlPlaneError = errs.ErrorCode{Code: "CORE-MESH-OP-2006", Title: "Communication with control-plane failed"}
var InvalidFacadeServiceCRError = errs.ErrorCode{Code: "CORE-MESH-OP-2007", Title: "Invalid FacadeService custom resource fields"}
