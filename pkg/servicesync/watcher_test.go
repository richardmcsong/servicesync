package servicesync

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func NewFake() *fake.Clientset {
	return fake.NewSimpleClientset(
		&corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fooService",
				Namespace: "foo",
			},
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{
						{
							IP: "1.2.3.4",
						},
					},
					Ports: []corev1.EndpointPort{
						{
							Port: 80,
						},
					},
				},
			},
		},
	)
}
