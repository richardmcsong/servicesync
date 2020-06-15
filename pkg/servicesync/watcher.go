package servicesync

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
)

func Run(v *viper.Viper) {
	// create service and endpoint
	ctx := context.Background()
	targetCS, err := kubernetes.NewForConfig(v.Get("destination-kube-config").(*rest.Config))
	if err != nil {
		logrus.Fatalf("unexpected error while creating destination client set: %s", err)
	}
	err = EnsureService(ctx, v.GetString("destination-namespace"), v.GetString("rename-service"), targetCS)
	if err != nil {
		logrus.Fatalf("unexpected error while ensuring service: %s", err)
	}
	err = EnsureEndpoints(ctx, v.GetString("destination-namespace"), v.GetString("rename-service"), targetCS)
	if err != nil {
		logrus.Fatalf("unexpected error while ensuring endpoints: %s", err)
	}
	sourceCS, err := kubernetes.NewForConfig(v.Get("source-kube-config").(*rest.Config))
	if err != nil {
		logrus.Fatalf("error while building source cluster client set: %s", err)
	}

	err = GetAndUpdateService(ctx, v.GetString("source-namespace"), v.GetString("service"), v.GetString("destination-namespace"), v.GetString("rename-service"), sourceCS, targetCS)
	if err != nil {
		logrus.Fatalf("error while initially updating service: %s", err)
	}
	err = GetAndUpdateEndpoints(ctx, v.GetString("source-namespace"), v.GetString("service"), v.GetString("destination-namespace"), v.GetString("rename-service"), sourceCS, targetCS)
	if err != nil {
		logrus.Fatalf("error while initially updating endpoints: %s", err)
	}

	// sync services and endpoints on startup
	SyncService(ctx, v.GetString("source-namespace"), v.GetString("service"), v.GetString("destination-namespace"), v.GetString("rename-service"), sourceCS, targetCS)
	SyncEndpoints(ctx, v.GetString("source-namespace"), v.GetString("service"), v.GetString("destination-namespace"), v.GetString("rename-service"), sourceCS, targetCS)

	// sleep forever
	<-(chan int)(nil)
}
