package servicesync

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

const sleepLength = 1 * time.Second

func TestEnsureEndpointsNotFound(t *testing.T) {
	namespace := "foo"
	targetName := "fooService"
	cs := fake.NewSimpleClientset()
	ctx := context.Background()
	err := EnsureEndpoints(ctx, namespace, targetName, cs)
	if err != nil {
		t.Error(err)
	}
	_, err = cs.CoreV1().Endpoints(namespace).Get(ctx, targetName, metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}
}

func TestGetAndUpdateEndpoints(t *testing.T) {
	ctx := context.Background()
	targetNamespace := "bar"
	targetName := "barService"
	sourceNamespace := "foo"
	sourceName := "fooService"
	sourceNodeName := "source-node"
	sourceCS := fake.NewSimpleClientset(&corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sourceName,
			Namespace: sourceNamespace,
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP:       "1.2.3.4",
						NodeName: &sourceNodeName,
						TargetRef: &corev1.ObjectReference{
							Kind:            "Pod",
							Name:            "source-pod-name-1",
							Namespace:       sourceNamespace,
							ResourceVersion: "3499251",
							UID:             "5aeb747c-2f17-421f-bf73-ee3c47410822",
						},
					},
					{
						IP:       "1.2.3.5",
						NodeName: &sourceNodeName,
						TargetRef: &corev1.ObjectReference{
							Kind:            "Pod",
							Name:            "source-pod-name-2",
							Namespace:       sourceNamespace,
							ResourceVersion: "3499252",
							UID:             "5aeb747c-2f17-421f-bf73-ee3c47410823",
						},
					},
				},
				NotReadyAddresses: []corev1.EndpointAddress{
					{
						IP:       "1.2.3.6",
						NodeName: &sourceNodeName,
						TargetRef: &corev1.ObjectReference{
							Kind:            "Pod",
							Name:            "source-pod-name-3",
							Namespace:       sourceNamespace,
							ResourceVersion: "3499253",
							UID:             "5aeb747c-2f17-421f-bf73-ee3c47410824",
						},
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Name:     "http",
						Port:     80,
						Protocol: "TCP",
					},
				},
			},
		},
	})
	targetCS := fake.NewSimpleClientset(&corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      targetName,
			Namespace: targetNamespace,
		},
	})
	err := GetAndUpdateEndpoints(ctx, sourceNamespace, sourceName, targetNamespace, targetName, sourceCS, targetCS)
	if err != nil {
		t.Error(err)
	}
	out, err := targetCS.CoreV1().Endpoints(targetNamespace).Get(ctx, targetName, metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}
	for _, ss := range out.Subsets {
		if ss.Addresses[0].NodeName != nil {
			t.Errorf("node name was not stripped: %s", *ss.Addresses[0].NodeName)
		}
		if ss.NotReadyAddresses[0].TargetRef != nil {
			t.Errorf("targetRef was not stripped.")
		}
		if ss.Ports == nil {
			t.Error("ports were not synced")
		}
	}
}

func TestSyncEndpoints(t *testing.T) {
	ctx := context.Background()
	targetNamespace := "bar"
	targetName := "barService"
	sourceNamespace := "foo"
	sourceName := "fooService"
	sourceNodeName := "source-node"
	sourceCS := fake.NewSimpleClientset(&corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sourceName,
			Namespace: sourceNamespace,
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP:       "1.2.3.4",
						NodeName: &sourceNodeName,
						TargetRef: &corev1.ObjectReference{
							Kind:            "Pod",
							Name:            "source-pod-name-1",
							Namespace:       sourceNamespace,
							ResourceVersion: "3499251",
							UID:             "5aeb747c-2f17-421f-bf73-ee3c47410822",
						},
					},
					{
						IP:       "1.2.3.5",
						NodeName: &sourceNodeName,
						TargetRef: &corev1.ObjectReference{
							Kind:            "Pod",
							Name:            "source-pod-name-2",
							Namespace:       sourceNamespace,
							ResourceVersion: "3499252",
							UID:             "5aeb747c-2f17-421f-bf73-ee3c47410823",
						},
					},
				},
				NotReadyAddresses: []corev1.EndpointAddress{
					{
						IP:       "1.2.3.6",
						NodeName: &sourceNodeName,
						TargetRef: &corev1.ObjectReference{
							Kind:            "Pod",
							Name:            "source-pod-name-3",
							Namespace:       sourceNamespace,
							ResourceVersion: "3499253",
							UID:             "5aeb747c-2f17-421f-bf73-ee3c47410824",
						},
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Name:     "http",
						Port:     80,
						Protocol: "TCP",
					},
				},
			},
		},
	})
	targetCS := fake.NewSimpleClientset(&corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      targetName,
			Namespace: targetNamespace,
		},
	})
	if err := SyncEndpoints(ctx, sourceNamespace, sourceName, targetNamespace, targetName, sourceCS, targetCS); err != nil {
		t.Error(err)
	}
	patch := []byte(`{
		"subsets": [
			{
				"addresses": [
					{
						"ip": "1.2.3.7",
						"nodeName": "source-node"
					}
				]
			}
		]
	}`)
	_, err := sourceCS.CoreV1().Endpoints(sourceNamespace).Patch(ctx, sourceName, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		t.Error(err)
	}
	time.Sleep(sleepLength)
	s, err := targetCS.CoreV1().Endpoints(targetNamespace).Get(ctx, targetName, metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}
	for _, v := range s.Subsets {
		found := false
		for _, a := range v.Addresses {
			if a.IP == "1.2.3.7" {
				found = true
				break
			}
		}
		if !found {
			t.Error("target wasn't synced.")
		}
	}
}
