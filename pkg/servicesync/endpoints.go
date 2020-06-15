package servicesync

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

//EnsureEndpoints ensures that getting the endpoints will not lead to a 404. An empty endpoints is created if not found.
func EnsureEndpoints(ctx context.Context, namespace, targetName string, cs kubernetes.Interface) error {
	_, err := cs.CoreV1().Endpoints(namespace).Get(ctx, targetName, metav1.GetOptions{})
	if err != nil {
		if err.(*k8serror.StatusError).Status().Code == http.StatusNotFound {
			_, err = cs.CoreV1().Endpoints(namespace).Create(ctx, &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      targetName,
					Namespace: namespace,
				},
			}, metav1.CreateOptions{})
			if err != nil {
				logrus.Errorf("Unexpected error while creating endpoints: %s", err)
				return err
			}
		} else {
			logrus.Errorf("Unexpected error while ensuring endpoints: %s", err)
			return err
		}
	}
	return nil
}

//SyncEndpoints does a one time sync between the source and target endpoints resource
func GetAndUpdateEndpoints(ctx context.Context, sourceNamespace, sourceName, targetNamespace, targetName string, sourceCS, targetCS kubernetes.Interface) error {
	s, err := sourceCS.CoreV1().Endpoints(sourceNamespace).Get(ctx, sourceName, metav1.GetOptions{})
	if err != nil {
		logrus.Errorf("error while getting endpoints definition from source: %s", err)
		return err
	}
	s = transformEndpoints(s, targetNamespace, targetName)
	_, err = targetCS.CoreV1().Endpoints(targetNamespace).Update(ctx, s, metav1.UpdateOptions{})
	if err != nil {
		logrus.Errorf("error while updating new target endpoints definition: %s", err)
		return err
	}
	return nil
}

func SyncEndpoints(ctx context.Context, sourceNamespace, sourceName, targetNamespace, targetName string, sourceCS, targetCS kubernetes.Interface) error {
	w, err := sourceCS.CoreV1().Endpoints(sourceNamespace).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		logrus.Errorf("error while establishing a watch connection from source: %s", err)
		return err
	}
	wc := w.ResultChan()
	go func() {
		for {
			event := <-wc
			if event.Type == "MODIFIED" {
				if endpoints := event.Object.(*corev1.Endpoints); endpoints.Name == sourceName {
					endpoints = transformEndpoints(endpoints, targetNamespace, targetName)
					_, err := targetCS.CoreV1().Endpoints(targetNamespace).Update(ctx, endpoints, metav1.UpdateOptions{})
					if err != nil {
						logrus.Errorf("error while updating target endpoints: %s", err)
					}
				}
			}
		}
	}()
	return nil
}

func transformEndpoints(s *corev1.Endpoints, namespace, name string) *corev1.Endpoints {
	var newSubsets []corev1.EndpointSubset
	for _, v := range s.Subsets {
		var newSubset corev1.EndpointSubset
		for _, a := range v.Addresses {
			newSubset.Addresses = append(newSubset.Addresses, corev1.EndpointAddress{IP: a.IP})
		}
		for _, a := range v.NotReadyAddresses {
			newSubset.NotReadyAddresses = append(newSubset.NotReadyAddresses, corev1.EndpointAddress{IP: a.IP})
		}
		newSubset.Ports = v.Ports
		newSubsets = append(newSubsets, newSubset)
	}
	transformed := corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Subsets: newSubsets,
	}
	return &transformed
}
