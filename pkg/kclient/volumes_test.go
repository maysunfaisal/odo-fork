package kclient

import (
	"testing"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktesting "k8s.io/client-go/testing"
)

func TestCreateCreatePVC(t *testing.T) {

	tests := []struct {
		name      string
		pvcName   string
		size      string
		namespace string
		labels    map[string]string
		wantErr   bool
	}{
		{
			name:      "Case: Valid pvc name",
			pvcName:   "mypvc",
			size:      "1Gi",
			namespace: "default",
			labels: map[string]string{
				"testpvc": "testpvc",
			},
			wantErr: false,
		},
		{
			name:      "Case: Invalid pvc name",
			pvcName:   "",
			size:      "1Gi",
			namespace: "default",
			labels: map[string]string{
				"testpvc": "testpvc",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// initialising the fakeclient
			fkclient, fkclientset := FakeNew()
			fkclient.Namespace = tt.namespace

			quantity, err := resource.ParseQuantity(tt.size)
			if err != nil {
				t.Errorf("resource.ParseQuantity unexpected error %v", err)
			}
			pvcSpec := GeneratePVCSpec(quantity)
			objectMeta := metav1.ObjectMeta{
				Name:        tt.pvcName,
				Namespace:   tt.namespace,
				Labels:      tt.labels,
				Annotations: nil,
			}

			fkclientset.Kubernetes.PrependReactor("create", "persistentvolumeclaims", func(action ktesting.Action) (bool, runtime.Object, error) {
				if tt.pvcName == "" {
					return true, nil, errors.Errorf("pvc name is empty")
				}
				pvc := corev1.PersistentVolumeClaim{
					TypeMeta: metav1.TypeMeta{
						Kind:       "PersistentVolumeClaim",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: tt.pvcName,
					},
				}
				return true, &pvc, nil
			})

			createdPVC, err := fkclient.CreatePVC(objectMeta, *pvcSpec)

			// Checks for unexpected error cases
			if !tt.wantErr == (err != nil) {
				t.Errorf("fkclient.CreatePVC unexpected error %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil {
				if len(fkclientset.Kubernetes.Actions()) != 1 {
					t.Errorf("expected 1 action in StartPVC got: %v", fkclientset.Kubernetes.Actions())
				} else {
					if createdPVC.Name != tt.pvcName {
						t.Errorf("deployment name does not match the expected name, expected: %s, got %s", tt.pvcName, createdPVC.Name)
					}
				}
			}
		})
	}
}
