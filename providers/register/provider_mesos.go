// +build !no_mesos_provider

package register

import (
	"github.com/virtual-kubelet/virtual-kubelet/providers"
	"github.com/virtual-kubelet/virtual-kubelet/providers/mesos"
)

func init() {
	register("mesos", initMesos)
}

func initMesos(cfg InitConfig) (providers.Provider, error) {
	return mesos.NewProvider(
		cfg.ConfigPath,
		cfg.NodeName,
		cfg.OperatingSystem,
		cfg.InternalIP,
		cfg.DaemonPort,
	)
}
