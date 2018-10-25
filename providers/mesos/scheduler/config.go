package scheduler

import (
	"time"

	"github.com/mesos/mesos-go/api/v1/lib/encoding/codecs"
)

// Config represents the configuration of the Mesos scheduler.
type Config struct {
	// Mesos URL
	MesosURL string `json:"mesosUrl,omitempty"`
	// Principal to authenticate againts Mesos.
	Principal string `json:"principal,omitempty"`
	// Framework name.
	Name string `json:"name,omitempty"`
	// Framework role.
	Role string `json:"role,omitempty"`
	// Codec to be used when talking to Mesos.
	Codec codec `json:"codec,omitempty"`
	// Framework connection to Mesos timeout.
	Timeout time.Duration `json:"timeout,omitempty"`
	// Framework failover timeout.
	FailoverTimeout time.Duration `json:"failoverTimeout,omitempty"`
	// Checkpoint framework tasks.
	Checkpoint bool `json:"checkpoint,omitempty"`
	// TODO hostname advertised to Mesos?
	//hostname            string
	// TODO Framework labels?
	//labels              Labels
	// Framework verbosity enabled?
	Verbose bool `json:"verbose,omitempty"`
	// Number of revive messages that may be sent in a burst within revive-wait period.
	ReviveBurst int `json:"reviveBurst,omitempty"`
	// Wait this long to fully recharge revive-burst quota.
	ReviveWait time.Duration `json:"reviveWait,omitempty"`
	// URI path to metrics endpoint.
	Metrics *metrics `json:"metrics,omitempty"`
	// Max length of time to refuse future offers.
	MaxRefuseSeconds time.Duration `json:"maxRefuseSeconds,omitempty"`
	// Duration between job (internal service) restarts between failures
	JobRestartDelay time.Duration `json:"jobRestartDelay,omitempty"`
	// Framework tasks user.
	User string `json:"user,omitempty"`
}

// DefaultConfig returns the default configuration for the Mesos framework
func DefaultConfig() *Config {
	timeout, _ := time.ParseDuration("20s")
	failoverTimeout, _ := time.ParseDuration("1000h")
	reviveWait, _ := time.ParseDuration("1s")
	maxRefuseSeconds, _ := time.ParseDuration("5s")
	jobRestartDelay, _ := time.ParseDuration("5s")

	return &Config{
		MesosURL:         "http://:5050/api/v1/scheduler",
		Name:             "vk_mesos",
		Role:             "*",
		Codec:            codec{Codec: codecs.ByMediaType[codecs.MediaTypeProtobuf]},
		Timeout:          timeout,
		FailoverTimeout:  failoverTimeout,
		Checkpoint:       true,
		ReviveBurst:      3,
		ReviveWait:       reviveWait,
		MaxRefuseSeconds: maxRefuseSeconds,
		JobRestartDelay:  jobRestartDelay,
		Metrics: &metrics{
			address: "localhost",
			port:    64009,
			path:    "/metrics",
		},
		User: "root",
		Verbose: true,
	}
}

type metrics struct {
	address string
	path    string
	port    int
}
