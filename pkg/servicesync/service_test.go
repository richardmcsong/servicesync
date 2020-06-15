package servicesync

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
)

func TestEnsureServiceNotFound(t *testing.T) {
	namespace := "foo"
	targetName := "fooService"
	cs := fake.NewSimpleClientset()
	ctx := context.Background()
	err := EnsureService(ctx, namespace, targetName, cs)
	if err != nil {
		t.Error(err)
	}
	_, err = cs.CoreV1().Services(namespace).Get(ctx, targetName, metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}
}

func TestGetAndUpdateService(t *testing.T) {
	ctx := context.Background()
	targetNamespace := "bar"
	targetName := "barService"
	sourceNamespace := "foo"
	sourceName := "fooService"
	sourceCS := fake.NewSimpleClientset(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sourceName,
			Namespace: sourceNamespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"fookey": "foovalue",
			},
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Protocol: "TCP",
					Port:     80,
					TargetPort: intstr.IntOrString{
						IntVal: 8000,
					},
				},
			},
		},
	})
	targetCS := fake.NewSimpleClientset(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      targetName,
			Namespace: targetNamespace,
		},
	})
	err := GetAndUpdateService(ctx, sourceNamespace, sourceName, targetNamespace, targetName, sourceCS, targetCS)
	if err != nil {
		t.Error(err)
	}
	out, err := targetCS.CoreV1().Services(targetNamespace).Get(ctx, targetName, metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}
	if out.Spec.Selector != nil {
		t.Error("label selector was not stripped.")
	}
}

func TestSyncService(t *testing.T) {
	ctx := context.Background()
	targetNamespace := "bar"
	targetName := "barService"
	sourceNamespace := "foo"
	sourceName := "fooService"
	sourceCS := fake.NewSimpleClientset(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sourceName,
			Namespace: sourceNamespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"fookey": "foovalue",
			},
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Protocol: "TCP",
					Port:     80,
					TargetPort: intstr.IntOrString{
						IntVal: 8000,
					},
				},
			},
		},
	})
	targetCS := fake.NewSimpleClientset(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      targetName,
			Namespace: targetNamespace,
		},
	})
	if err := SyncService(ctx, sourceNamespace, sourceName, targetNamespace, targetName, sourceCS, targetCS); err != nil {
		t.Error(err)
	}
	patch := []byte(`{
		"spec": {
			"ports": [
				{
					"name": "http",
					"protocol": "TCP",
					"port": 81,
					"targetPort": 8000
				}
			]
		}	
	}`)
	_, err := sourceCS.CoreV1().Services(sourceNamespace).Patch(ctx, sourceName, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)
	s, err := targetCS.CoreV1().Services(targetNamespace).Get(ctx, targetName, metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}
	for _, v := range s.Spec.Ports {
		if v.Name == "http" {
			if v.Port != 81 {
				t.Errorf("port did not update: found %d but should be 81", v.Port)
			}
		}
	}
}
