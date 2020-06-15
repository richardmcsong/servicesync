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

//EnsureService ensures that getting the service will not lead to a 404. An empty service is created if not found.
func EnsureService(ctx context.Context, namespace, targetName string, cs kubernetes.Interface) error {
	_, err := cs.CoreV1().Services(namespace).Get(ctx, targetName, metav1.GetOptions{})
	if err != nil {
		if err.(*k8serror.StatusError).Status().Code == http.StatusNotFound {
			_, err = cs.CoreV1().Services(namespace).Create(ctx, &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      targetName,
					Namespace: namespace,
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Port: 80,
						},
					},
				},
			}, metav1.CreateOptions{})
			if err != nil {
				logrus.Errorf("Unexpected error while creating service: %s", err)
				return err
			}
		} else {
			logrus.Errorf("Unexpected error while ensuring service: %s", err)
			return err
		}
	}
	return nil
}

//GetAndUpdateService does a one time sync between the source and target service resource
func GetAndUpdateService(ctx context.Context, sourceNamespace, sourceName, targetNamespace, targetName string, sourceCS, targetCS kubernetes.Interface) error {
	s, err := sourceCS.CoreV1().Services(sourceNamespace).Get(ctx, sourceName, metav1.GetOptions{})
	if err != nil {
		logrus.Errorf("error while getting service definition from source: %s", err)
		return err
	}
	return UpdateService(ctx, s, targetNamespace, targetName, targetCS)
}

func SyncService(ctx context.Context, sourceNamespace, sourceName, targetNamespace, targetName string, sourceCS, targetCS kubernetes.Interface) error {
	w, err := sourceCS.CoreV1().Services(sourceNamespace).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		logrus.Errorf("error while establishing a watch connection from source: %s", err)
		return err
	}
	wc := w.ResultChan()
	go func() {
		for {
			event := <-wc
			if event.Type == "MODIFIED" {
				if source := event.Object.(*corev1.Service); source.Name == sourceName {
					if err = UpdateService(ctx, source, targetNamespace, targetName, targetCS); err != nil {
						logrus.Errorf("error while updating service: %s", err)
					}
				}
			}
		}
	}()
	return nil
}

func UpdateService(ctx context.Context, source *corev1.Service, targetNamespace, targetName string, targetCS kubernetes.Interface) error {
	target, err := targetCS.CoreV1().Services(targetNamespace).Get(ctx, targetName, metav1.GetOptions{})
	if err != nil {
		logrus.Errorf("error while getting existing target service: %s", err)
		return err
	}
	service := transformService(source, target)
	_, err = targetCS.CoreV1().Services(targetNamespace).Update(ctx, service, metav1.UpdateOptions{})
	if err != nil {
		logrus.Errorf("error while updating target service: %s", err)
		return err
	}
	return nil
}

func transformService(source, target *corev1.Service) *corev1.Service {
	transformed := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            target.Name,
			Namespace:       target.Namespace,
			ResourceVersion: target.ResourceVersion,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP:   target.Spec.ClusterIP,
			Ports:       source.Spec.Ports,
			Type:        source.Spec.Type,
			ExternalIPs: source.Spec.ExternalIPs,
		},
	}
	return &transformed
}
