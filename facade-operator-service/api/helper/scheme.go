package helper

import (
	"context"
	"sync"

	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	"github.com/netcracker/qubership-core-lib-go/v3/serviceloader"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	once                 sync.Once
	defaultSchemeManager SchemeManager
)

type SchemeManager interface {
	Register(ctx context.Context, kind string, object ...runtime.Object)
	AddAllToScheme(ctx context.Context, s *runtime.Scheme) error
}

func DefaultSchemeManager() SchemeManager {
	once.Do(func() {
		mgr := &schemeManager{
			logger:           logging.GetLogger("SchemeManager"),
			apiGroupProvider: serviceloader.MustLoad[ApiGroupVersionProvider](),
			builders:         make(map[string][]*scheme.Builder),
		}

		mgr.fillBuildersForKind(GatewayKind)
		mgr.fillBuildersForKind(FacadeServiceKind)

		defaultSchemeManager = mgr
	})

	return defaultSchemeManager
}

type schemeManager struct {
	logger           logging.Logger
	apiGroupProvider ApiGroupVersionProvider
	builders         map[string][]*scheme.Builder
}

func (m *schemeManager) fillBuildersForKind(kind string) {
	for _, apiGroup := range m.apiGroupProvider.GetApiGroups(kind) {
		// SchemeBuilder is used to add go types to the GroupVersionKind scheme
		schemeBuilder := &scheme.Builder{GroupVersion: apiGroup}

		if m.builders[kind] == nil {
			m.builders[kind] = make([]*scheme.Builder, 0, 2)
		}
		m.builders[kind] = append(m.builders[kind], schemeBuilder)
	}
}

func (m *schemeManager) Register(ctx context.Context, kind string, object ...runtime.Object) {
	for _, schemeBuilder := range m.builders[kind] {
		m.logger.InfoC(ctx, "Registering scheme for kind %s: %v", kind, schemeBuilder.GroupVersion)
		schemeBuilder.Register(object...)
	}
}

// AddAllToScheme adds the types in all these group-versions to the given scheme.
func (m *schemeManager) AddAllToScheme(ctx context.Context, s *runtime.Scheme) error {
	for _, buildersForKind := range m.builders {
		for _, schemeBuilder := range buildersForKind {
			m.logger.InfoC(ctx, "Adding groupVersion %s to scheme", schemeBuilder.GroupVersion)
			if err := schemeBuilder.AddToScheme(s); err != nil {
				m.logger.ErrorC(ctx, "Failed to add groupVersion %s to scheme:\n %v", schemeBuilder.GroupVersion, err)
				return err
			}
		}
	}
	return nil
}
