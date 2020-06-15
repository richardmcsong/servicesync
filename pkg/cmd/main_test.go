package cmd

import (
	"reflect"
	"testing"

	"github.com/spf13/viper"
	"k8s.io/client-go/rest"
)

func TestDefaultDestKubeConfig(t *testing.T) {
	v := viper.New()
	handleDefaultDestKubeConfig(v)
	if destKubeConfig := v.Get("destination-kube-config"); destKubeConfig != nil {
		if _, ok := destKubeConfig.(*rest.Config); !ok {
			t.Errorf("destination-kube-config should be pointer to rest.Config, but found %s", reflect.TypeOf(v.Get("destination-kube-config")))
		}
	}
}

func TestSetDestKubeConfig(t *testing.T) {
	v := viper.New()
	v.Set("destination-kube-config", "testdata/test.config")
	if err := handleDefaultDestKubeConfig(v); err != nil {
		t.Errorf("error while loading custom path to destination config: %s", err)
	}
	config, ok := v.Get("destination-kube-config").(*rest.Config)
	if !ok {
		t.Fatalf("destination-kube-config should be pointer to rest.Config, but found %s", reflect.TypeOf(v.Get("destination-kube-config")))
	}
	if config.BearerToken != "token2" {
		t.Errorf("config from test file was not loaded correctly")
	}
}
