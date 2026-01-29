package nodesetup

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const labelKey = "gar-credential-provider/configured"

type Labeler interface {
	LabelNode(ctx context.Context, nodeName, version string) error
}

var _ Labeler = (*labeler)(nil)

type labeler struct {
	clientset kubernetes.Interface
}

func NewLabeler() (Labeler, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("get in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes clientset: %w", err)
	}

	return &labeler{clientset: clientset}, nil
}

func NewLabelerWithClient(clientset kubernetes.Interface) Labeler {
	return &labeler{clientset: clientset}
}

func (l *labeler) LabelNode(ctx context.Context, nodeName, version string) error {
	patch := []byte(fmt.Sprintf(
		`{"metadata":{"labels":{%q:%q}}}`,
		labelKey, version,
	))

	_, err := l.clientset.CoreV1().Nodes().Patch(
		ctx,
		nodeName,
		types.StrategicMergePatchType,
		patch,
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("patch node %s: %w", nodeName, err)
	}

	return nil
}
