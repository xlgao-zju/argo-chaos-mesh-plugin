package chaos_mesh

import (
	"context"
	"fmt"
	"net/http"

	chaosmeshapi "github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = chaosmeshapi.AddToScheme(scheme)
}

// Client define APi sets fro chaos mesh
type Client interface {
	// CreateExperiment create chaos experiment
	CreateExperiment(ctx context.Context, chaos chaosmeshapi.InnerObject) (chaosmeshapi.InnerObject, error)
	// DeleteExperiment delete chaos experiment
	DeleteExperiment(ctx context.Context, namespace, name, kind string) error
	// GetExperiment get chaos experiment
	GetExperiment(ctx context.Context, namespace, name, kind string) (chaosmeshapi.InnerObject, error)
	// ListExperiments list chaos experiments
	ListExperiments(ctx context.Context, kind string) ([]chaosmeshapi.InnerObject, error)
}

type client struct {
	kubeCli pkgclient.Client
}

// CreateExperiment create chaos experiment
func (c *client) CreateExperiment(ctx context.Context, chaos chaosmeshapi.InnerObject) (chaosmeshapi.InnerObject, error) {
	kind := ""
	switch chaos.(type) {
	case *chaosmeshapi.PodChaos:
		kind = string(chaosmeshapi.TypePodChaos)
	case *chaosmeshapi.NetworkChaos:
		kind = string(chaosmeshapi.TypeNetworkChaos)
	case *chaosmeshapi.StressChaos:
		kind = string(chaosmeshapi.TypeStressChaos)
	case *chaosmeshapi.IOChaos:
		kind = string(chaosmeshapi.TypeIOChaos)
	case *chaosmeshapi.HTTPChaos:
		kind = string(chaosmeshapi.TypeHTTPChaos)
	case *chaosmeshapi.DNSChaos:
		kind = string(chaosmeshapi.TypeDNSChaos)
	case *chaosmeshapi.TimeChaos:
		kind = string(chaosmeshapi.TypeTimeChaos)
	}
	_, ok := chaosmeshapi.AllKinds()[kind]
	if !ok {
		return nil, fmt.Errorf("not support chaos kind '%s'", kind)
	}

	if err := c.kubeCli.Create(ctx, chaos); err != nil {
		return nil, fmt.Errorf("failed create chaos, %s", err.Error())
	}
	return c.GetExperiment(ctx, chaos.GetNamespace(), chaos.GetName(), kind)
}

// DeleteExperiment delete chaos experiment
func (c *client) DeleteExperiment(ctx context.Context, namespace, name, kind string) error {
	chaosKind, exists := chaosmeshapi.AllKinds()[kind]
	if !exists {
		return fmt.Errorf("unknwon chaos kind '%s'", kind)
	}

	chaos := chaosKind.SpawnObject()
	namespacedName := types.NamespacedName{Namespace: namespace, Name: name}
	if err := c.kubeCli.Get(ctx, namespacedName, chaos); err != nil {
		return err
	}

	if err := c.kubeCli.Delete(ctx, chaos); err != nil {
		return fmt.Errorf("failed delete chaos, %s", err.Error())
	}
	return nil
}

// GetExperiment get chaos experiment
func (c *client) GetExperiment(ctx context.Context, namespace, name, kind string) (chaosmeshapi.InnerObject, error) {
	chaosKind, exists := chaosmeshapi.AllKinds()[kind]
	if !exists {
		return nil, fmt.Errorf("unknwon chaos kind '%s'", kind)
	}

	chaos := chaosKind.SpawnObject()
	namespacedName := types.NamespacedName{Namespace: namespace, Name: name}
	if err := c.kubeCli.Get(ctx, namespacedName, chaos); err != nil {
		return nil, err
	}
	return chaos.(chaosmeshapi.InnerObject), nil
}

// ListExperiments list chaos experiments
func (c *client) ListExperiments(ctx context.Context, kind string) ([]chaosmeshapi.InnerObject, error) {
	chaosKind, exists := chaosmeshapi.AllKinds()[kind]
	if !exists {
		return nil, fmt.Errorf("unknwon chaos kind '%s'", kind)
	}

	list := chaosKind.SpawnList()
	listOptions := &pkgclient.ListOptions{Namespace: "chaos-testing"}
	if err := c.kubeCli.List(ctx, list, listOptions); err != nil {
		return nil, err
	}

	experimentList := make([]chaosmeshapi.InnerObject, 0)
	for _, item := range list.GetItems() {
		experimentList = append(experimentList, item.(chaosmeshapi.InnerObject))
	}
	return experimentList, nil
}

// NewClient create chaos mesh client
func NewClient() (Client, error) {
	cfg, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}
	cli, err := pkgclient.New(cfg, pkgclient.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, err
	}
	return &client{
		kubeCli: cli,
	}, nil
}

func isNotFound(err error) bool {
	statusErr, ok := err.(*errors.StatusError)
	if !ok {
		return false
	}
	return statusErr.ErrStatus.Code == http.StatusNotFound
}
