package scheduler

import (
	"log"

	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/backoff"
	"github.com/mesos/mesos-go/api/v1/lib/httpcli"
	"github.com/mesos/mesos-go/api/v1/lib/httpcli/httpsched"
	"github.com/mesos/mesos-go/api/v1/lib/scheduler/calls"
)

func buildMesosCaller(schedulerConfig *Config) calls.Caller {
	var authConfigOpt httpcli.ConfigOpt
	cli := httpcli.New(
		httpcli.Endpoint(schedulerConfig.MesosURL),
		httpcli.Codec(schedulerConfig.Codec.Codec),
		httpcli.Do(httpcli.With(
			authConfigOpt,
			httpcli.Timeout(schedulerConfig.Timeout),
		)),
	)
	return httpsched.NewCaller(cli, httpsched.Listener(func(n httpsched.Notification) {
		if schedulerConfig.Verbose {
			log.Printf("scheduler client notification: %+v", n)
		}
	}))
}

func buildFrameworkInfo(schedulerConfig *Config) *mesos.FrameworkInfo {
	failoverTimeout := schedulerConfig.FailoverTimeout.Seconds()
	frameworkInfo := &mesos.FrameworkInfo{
		User:       schedulerConfig.User,
		Name:       schedulerConfig.Name,
		Checkpoint: &schedulerConfig.Checkpoint,
		Capabilities: []mesos.FrameworkInfo_Capability{
			{Type: mesos.FrameworkInfo_Capability_RESERVATION_REFINEMENT},
		},
	}
	if schedulerConfig.FailoverTimeout > 0 {
		frameworkInfo.FailoverTimeout = &failoverTimeout
	}
	if schedulerConfig.Role != "" {
		frameworkInfo.Role = &schedulerConfig.Role
	}
	if schedulerConfig.Principal != "" {
		frameworkInfo.Principal = &schedulerConfig.Principal
	}
	// TODO hostname?
	//if schedulerConfig.hostname != "" {
	//	frameworkInfo.Hostname = &schedulerConfig.hostname
	//}
	// TODO labels?
	//if len(schedulerConfig.labels) > 0 {
	//	log.Println("using labels:", schedulerConfig.labels)
	//	frameworkInfo.Labels = &mesos.Labels{Labels: schedulerConfig.labels}
	//}
	// TODO gpu?
	//if schedulerConfig.gpuClusterCompat {
	//	frameworkInfo.Capabilities = append(frameworkInfo.Capabilities,
	//		mesos.FrameworkInfo_Capability{Type: mesos.FrameworkInfo_Capability_GPU_RESOURCES},
	//	)
	//}
	return frameworkInfo
}

func newStateStore(schedulerConfig *Config) *stateStore {
	return &stateStore{
		config:        schedulerConfig,
		cli:           buildMesosCaller(schedulerConfig),
		reviveTokens:  backoff.BurstNotifier(schedulerConfig.ReviveBurst, schedulerConfig.ReviveWait, schedulerConfig.ReviveWait, nil),
		metricsAPI:    initMetrics(*schedulerConfig),
		newPodMap:     NewMesosPodMap(),
		runningPodMap: NewMesosPodMap(),
		deletedPodMap: NewMesosPodMap(),
	}
}

type stateStore struct {
	role          string
	cli           calls.Caller
	config        *Config
	reviveTokens  <-chan struct{}
	metricsAPI    *metricsAPI
	err           error
	newPodMap     *MesosPodMap
	runningPodMap *MesosPodMap
	deletedPodMap *MesosPodMap
}
