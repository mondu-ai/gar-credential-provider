package nodesetup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestLabeler_LabelNode(t *testing.T) {
	tests := []struct {
		name     string
		nodeName string
		version  string
		wantErr  bool
	}{
		{
			name:     "labels node successfully",
			nodeName: "test-node",
			version:  "v1.2.0",
		},
		{
			name:     "node not found",
			nodeName: "nonexistent-node",
			version:  "v1.0.0",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset(&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
				},
			})

			labeler := NewLabelerWithClient(clientset)

			err := labeler.LabelNode(context.Background(), tt.nodeName, tt.version)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			node, err := clientset.CoreV1().Nodes().Get(context.Background(), tt.nodeName, metav1.GetOptions{})
			require.NoError(t, err)

			assert.Equal(t, tt.version, node.Labels[labelKey])
		})
	}
}
