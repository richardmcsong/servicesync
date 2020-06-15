package cmd

import (
	"os"
	"path"
	"strings"

	"github.com/richardmcsong/servicesync/pkg/config"
	"github.com/richardmcsong/servicesync/pkg/servicesync"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var checkFlags []string

var (
	cfgFile               string
	sourceName            string
	sourceNamespace       string
	destinationName       string
	destinationNamespace  string
	sourceKubeConfig      string
	destinationKubeConfig string
)

var c = &cobra.Command{
	Use:   "servicesync",
	Short: "servicesync is a tool to synchonize service definitions across cluster boundaries.",
	Run: func(cmd *cobra.Command, args []string) {
		servicesync.Run(viper.GetViper())
	},
}

func addStringVarP(c *cobra.Command, p *string, name, shorthand, value, usage string) error {
	c.Flags().StringVarP(p, name, shorthand, value, usage)
	checkFlags = append(checkFlags, name)
	return viper.BindPFlag(name, c.Flags().Lookup(name))
}

func addStringVar(c *cobra.Command, p *string, name, value, usage string) error {
	c.Flags().StringVar(p, name, value, usage)
	checkFlags = append(checkFlags, name)
	return viper.BindPFlag(name, c.Flags().Lookup(name))
}

// Execute the cli
func Execute() {
	logrus.Infof("Starting service sync version: %s", config.Version)
	logrus.SetLevel(logrus.DebugLevel)
	cobra.OnInitialize(initConfig)
	c.Flags().StringVarP(&cfgFile, "config", "c", "", "path to config file") // not strictly necessary -- all other configurations are necessary.
	viper.BindPFlag("config", c.Flags().Lookup("config"))
	addStringVarP(c, &sourceName, "service", "s", "", "name of the source service to be synchronized from the source cluster")
	addStringVar(c, &destinationName, "rename-service", "", "name of the destination service in the destination cluster")
	addStringVar(c, &sourceKubeConfig, "source-kube-config", "", "path to kubeconfig file for source cluster")
	addStringVar(c, &destinationKubeConfig, "destination-kube-config", "", "path to kubeconfig file for destination cluster. Defaults to the current kube context.")
	addStringVar(c, &sourceNamespace, "source-namespace", "", "namespace of source service.")
	addStringVar(c, &destinationNamespace, "destination-namespace", "", "namespace of target service.")
	if err := c.Execute(); err != nil {
		os.Exit(1)
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func initConfig() {
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.SetEnvPrefix("SS")
	viper.AutomaticEnv()
	if viper.IsSet("config") {
		logrus.Debugf("using configuration file: %s", viper.GetString("config"))
		viper.SetConfigFile(viper.GetString("config"))
	}
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		logrus.Debug("Using config file:", viper.ConfigFileUsed())
	} else {
		logrus.Debugf("Could not use config file: %s: %s", viper.ConfigFileUsed(), err)
	}
	// handle defaults
	viper.SetDefault("rename-service", viper.Get("service"))
	handleDefaultDestKubeConfig(viper.GetViper())
	// load source-kube-config
	sourcek, err := clientcmd.BuildConfigFromFlags("", viper.GetString("source-kube-config"))
	if err != nil {
		logrus.Errorf("could not load source-kube-config: %s", err)
		c.Usage()
		os.Exit(1)
	}
	viper.Set("source-kube-config", sourcek)

	for _, v := range checkFlags {
		if !viper.IsSet(v) {
			logrus.Errorf("required configuration \"%s\" not set", v)
			c.Usage()
			os.Exit(1)
		}
	}
}

func handleDefaultDestKubeConfig(v *viper.Viper) error {
	// handle destination-kube-config. priority: 1. custom set on config, 2. inclusterconfig, 3. default home kubeconfig, 4. ???
	if !v.IsSet("destination-kube-config") {
		config, err := rest.InClusterConfig()
		if err == rest.ErrNotInCluster {
			if h := homeDir(); h != "" {
				config, err = clientcmd.BuildConfigFromFlags("", path.Join(h, ".kube", "config"))
				if err != nil {
					return err
				}
				v.Set("destination-kube-config", config)
				return nil
			}
			logrus.Fatal("could not find acceptable kubeconfig for the destination cluster")
		} else if err == nil {
			v.Set("destination-kube-config", config)
			return nil
		}
		logrus.Fatalf("Unexpected error while getting in cluster config: %s", err)
	}
	config, err := clientcmd.BuildConfigFromFlags("", v.GetString("destination-kube-config"))
	if err != nil {
		return err
	}
	v.Set("destination-kube-config", config)
	return nil
}
