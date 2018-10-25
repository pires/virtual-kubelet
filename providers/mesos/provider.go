package mesos

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/virtual-kubelet/virtual-kubelet/providers"
	"github.com/virtual-kubelet/virtual-kubelet/providers/mesos/scheduler"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/remotecommand"
)

// Provider implements the virtual-kubelet provider interface and stores podsMap in memory.
type Provider struct {
	config             *providerConfig
	nodeName           string
	operatingSystem    string
	internalIP         string
	daemonEndpointPort int32
	lastTransitionTime time.Time
	mesosScheduler     scheduler.Scheduler
}

// NewProvider creates a new Mesos provider
func NewProvider(
	configPath string,
	nodeName string,
	operatingSystem string,
	internalIP string,
	daemonEndpointPort int32) (*Provider, error) {

	log.Println("Initializing the Mesos provider.")

	p := &Provider{
		nodeName:           nodeName,
		operatingSystem:    operatingSystem,
		internalIP:         internalIP,
		daemonEndpointPort: daemonEndpointPort,
	}

	// Load configuration file.
	if configPath != "" {
		f, err := os.Open(configPath)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		if err := p.loadConfig(f); err != nil {
			return nil, err
		}
	}

	log.Println("Initializing the Mesos scheduler.")
	// Initialize the Mesos scheduler.
	p.mesosScheduler = scheduler.New(p.config.Scheduler)
	// Start the scheduler
	go p.mesosScheduler.Run()

	log.Println("Mesos scheduler initialized.")

	log.Printf("Mesos provider initialized: %+v.", p)

	return p, nil
}

// CreatePod accepts a Pod definition and stores it in memory.
func (p *Provider) CreatePod(ctx context.Context, pod *corev1.Pod) error {
	log.Printf("receivedd CreatePod %q\n", pod.Name)

	p.mesosScheduler.AddPod(pod)

	return nil
}

// UpdatePod accepts a Pod definition and updates its reference.
func (p *Provider) UpdatePod(ctx context.Context, pod *corev1.Pod) error {
	log.Printf("received UpdatePod %q\n", pod.Name)

	return p.mesosScheduler.UpdatePod(pod)
}

// DeletePod deletes the specified pod out of memory.
func (p *Provider) DeletePod(ctx context.Context, pod *corev1.Pod) (err error) {
	log.Printf("received DeletePod %q\n", pod.Name)

	return p.mesosScheduler.DeletePod(pod)
}

// GetPod returns a pod by name that is stored in memory.
func (p *Provider) GetPod(ctx context.Context, namespace, name string) (pod *corev1.Pod, err error) {
	log.Printf("received GetPod %q from namespace %q\n", name, namespace)

	return p.mesosScheduler.GetPod(namespace, name), nil
}

// GetContainerLogs retrieves the logs of a container by name from the provider.
func (p *Provider) GetContainerLogs(ctx context.Context, namespace, podName, containerName string, tail int) (string, error) {
	log.Printf("received GetContainerLogs %q\n", podName)
	// TODO implement
	return "TODO", nil
}

// GetPodFullName retrieves full pod name as defined in the provider context
func (p *Provider) GetPodFullName(namespace string, pod string) string {
	// TODO implement
	return "TODO"
}

// ExecInContainer executes a command in a container in the pod, copying data
// between in/out/err and the container's stdin/stdout/stderr.
func (p *Provider) ExecInContainer(name string, uid types.UID, container string, cmd []string, in io.Reader, out, err io.WriteCloser, tty bool, resize <-chan remotecommand.TerminalSize, timeout time.Duration) error {
	log.Printf("received ExecInContainer %q\n", container)
	// TODO implement
	return nil
}

// GetPodStatus returns the status of a pod by name that is "running".
// returns nil if a pod by that name is not found.
func (p *Provider) GetPodStatus(ctx context.Context, namespace, name string) (*corev1.PodStatus, error) {
	log.Printf("received GetPodStatus %q\n", name)

	// TODO implement

	now := metav1.NewTime(time.Now())

	status := &corev1.PodStatus{
		Phase:     corev1.PodRunning,
		HostIP:    "1.2.3.4",
		PodIP:     "5.6.7.8",
		StartTime: &now,
		Conditions: []corev1.PodCondition{
			{
				Type:   corev1.PodInitialized,
				Status: corev1.ConditionTrue,
			},
			{
				Type:   corev1.PodReady,
				Status: corev1.ConditionTrue,
			},
			{
				Type:   corev1.PodScheduled,
				Status: corev1.ConditionTrue,
			},
		},
	}

	pod, err := p.GetPod(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	if pod == nil {
		errorMessage := fmt.Sprintf("failed to retrieve pod status. no pod %q found in namespace %q", name, namespace)
		return nil, errors.New(errorMessage)
	}

	for _, container := range pod.Spec.Containers {
		status.ContainerStatuses = append(status.ContainerStatuses, corev1.ContainerStatus{
			Name:         container.Name,
			Image:        container.Image,
			Ready:        true,
			RestartCount: 0,
			State: corev1.ContainerState{
				Running: &corev1.ContainerStateRunning{
					StartedAt: now,
				},
			},
		})
	}

	return status, nil
}

// GetPods returns a list of all pods known to be "running".
func (p *Provider) GetPods(ctx context.Context) ([]*corev1.Pod, error) {
	log.Printf("received GetPods\n")

	return p.mesosScheduler.ListPods(), nil
}

// Capacity returns a resource list containing the capacity limits.
func (p *Provider) Capacity(ctx context.Context) corev1.ResourceList {
	return corev1.ResourceList{
		corev1.ResourceCPU:     resource.MustParse(p.config.CPU),
		corev1.ResourceMemory:  resource.MustParse(p.config.Memory),
		corev1.ResourceStorage: resource.MustParse(p.config.Storage),
		corev1.ResourcePods:    resource.MustParse(p.config.Pods),
	}
}

// NodeConditions returns a list of conditions (Ready, OutOfDisk, etc),
// for updates to the node status within Kubernetes.
func (p *Provider) NodeConditions(ctx context.Context) []corev1.NodeCondition {
	lastHeartbeatTime := metav1.Now()
	lastTransitionTime := metav1.NewTime(p.lastTransitionTime)
	lastTransitionReason := "Mesos scheduler is ready"
	lastTransitionMessage := "ok"

	// Return static thumbs-up values for all conditions.
	return []corev1.NodeCondition{
		{
			Type:               corev1.NodeReady,
			Status:             corev1.ConditionTrue,
			LastHeartbeatTime:  lastHeartbeatTime,
			LastTransitionTime: lastTransitionTime,
			Reason:             lastTransitionReason,
			Message:            lastTransitionMessage,
		},
		{
			Type:               corev1.NodeOutOfDisk,
			Status:             corev1.ConditionFalse,
			LastHeartbeatTime:  lastHeartbeatTime,
			LastTransitionTime: lastTransitionTime,
			Reason:             lastTransitionReason,
			Message:            lastTransitionMessage,
		},
		{
			Type:               corev1.NodeMemoryPressure,
			Status:             corev1.ConditionFalse,
			LastHeartbeatTime:  lastHeartbeatTime,
			LastTransitionTime: lastTransitionTime,
			Reason:             lastTransitionReason,
			Message:            lastTransitionMessage,
		},
		{
			Type:               corev1.NodeDiskPressure,
			Status:             corev1.ConditionFalse,
			LastHeartbeatTime:  lastHeartbeatTime,
			LastTransitionTime: lastTransitionTime,
			Reason:             lastTransitionReason,
			Message:            lastTransitionMessage,
		},
		{
			Type:               corev1.NodeNetworkUnavailable,
			Status:             corev1.ConditionFalse,
			LastHeartbeatTime:  lastHeartbeatTime,
			LastTransitionTime: lastTransitionTime,
			Reason:             lastTransitionReason,
			Message:            lastTransitionMessage,
		},
		{
			Type:               "KubeletConfigOk",
			Status:             corev1.ConditionTrue,
			LastHeartbeatTime:  lastHeartbeatTime,
			LastTransitionTime: lastTransitionTime,
			Reason:             lastTransitionReason,
			Message:            lastTransitionMessage,
		},
	}
}

// NodeAddresses returns a list of addresses for the node status
// within Kubernetes.
func (p *Provider) NodeAddresses(ctx context.Context) []corev1.NodeAddress {
	return []corev1.NodeAddress{
		{
			Type:    corev1.NodeInternalIP,
			Address: p.internalIP,
		},
	}
}

// NodeDaemonEndpoints returns NodeDaemonEndpoints for the node status
// within Kubernetes.
func (p *Provider) NodeDaemonEndpoints(ctx context.Context) *corev1.NodeDaemonEndpoints {
	return &corev1.NodeDaemonEndpoints{
		KubeletEndpoint: corev1.DaemonEndpoint{
			Port: p.daemonEndpointPort,
		},
	}
}

// OperatingSystem returns the operating system for this provider.
// This is a noop to default to Linux for now.
func (p *Provider) OperatingSystem() string {
	return providers.OperatingSystemLinux
}
