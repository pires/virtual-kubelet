package scheduler

import (
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/resources"

	corev1 "k8s.io/api/core/v1"
)

const (
	defaultContainerCPU  = float64(0.1) // 0.1 = 100 milliseconds of a second
	defaultContainerMem  = float64(32)  // 32 MB
	defaultContainerDisk = float64(128) // 128 MB
)

// calculateTaskResources calculates the resources needed for a container.
func calculateTaskResources(container corev1.Container) mesos.Resources {

	containerCPU := float64(container.Resources.Requests.Cpu().MilliValue() / 1000)
	if containerCPU < defaultContainerCPU {
		containerCPU = defaultContainerCPU
	}

	containerMem := float64(container.Resources.Requests.Memory().Value() / 1024 / 1024)
	if containerMem < defaultContainerMem {
		containerMem = defaultContainerMem
	}

	containerDisk := float64(container.Resources.Requests.StorageEphemeral().MilliValue() / 1000)
	if containerDisk < defaultContainerDisk {
		containerDisk = defaultContainerDisk
	}

	return mesos.Resources{
		resources.NewCPUs(containerCPU).Resource,
		resources.NewMemory(containerMem).Resource,
		resources.NewDisk(containerDisk).Resource,
	}
}

// sumPodResources sums all the resources required for a Pod.
// Enforce default value if resource == 0
func sumPodResources(pod *corev1.Pod) mesos.Resources {

	var podResources mesos.Resources
	for _, containerSpec := range pod.Spec.Containers {
		containerResources := calculateTaskResources(containerSpec)
		podResources = podResources.Plus(containerResources...)
	}

	return podResources
}

// buildPodTasks creates a new Mesos TaskGroup based on a Kubernetes pod definition.
func buildPodTasks(pod *corev1.Pod) []mesos.TaskInfo {

	// Build a TaskInfo for each container in the Kubernetes Pod.
	cap := len(pod.Spec.Containers)
	tasks := make([]mesos.TaskInfo, cap, cap)

	for pos, containerSpec := range pod.Spec.Containers {
		taskId := mesos.TaskID{Value: pod.Namespace + "-" + pod.Name + "-" + containerSpec.Name}

		// Build task environment variables.
		var taskEnvVars []mesos.Environment_Variable
		for _, envVar := range containerSpec.Env {
			taskEnvVar := mesos.Environment_Variable{Name: envVar.Name, Value: proto.String(envVar.Value)}
			taskEnvVars = append(taskEnvVars, taskEnvVar)
		}

		// Build TaskInfo.
		task := mesos.TaskInfo{
			TaskID: taskId,
			Container: &mesos.ContainerInfo{
				Type: mesos.ContainerInfo_MESOS.Enum(),
				Mesos: &mesos.ContainerInfo_MesosInfo{
					Image: &mesos.Image{
						Type: mesos.Image_DOCKER.Enum(),
						Docker: &mesos.Image_Docker{
							Name: containerSpec.Image,
						},
					},
				},
			},
			// TODO @pires build command info properly based on podSpec.Command, podSpec.Args, etc.
			Command: &mesos.CommandInfo{
				Shell: proto.Bool(false),
				//Value:     proto.String(strings.Join(containerSpec.Command, " ")),
				//Arguments: containerSpec.Args,
				Environment: &mesos.Environment{
					Variables: taskEnvVars,
				},
			},
		}
		tasks[pos] = task
	}

	return tasks
}

// buildDefaultExecutorInfo returns the protof of a default executor.
func buildDefaultExecutorInfo(fid mesos.FrameworkID) (mesos.ExecutorInfo) {
	return mesos.ExecutorInfo{
		Type:        mesos.ExecutorInfo_DEFAULT,
		FrameworkID: &fid,
		Container: &mesos.ContainerInfo{
			Type: mesos.ContainerInfo_MESOS.Enum(),
			NetworkInfos: []mesos.NetworkInfo{
				{
					IPAddresses: []mesos.NetworkInfo_IPAddress{{}},
					//Name:        proto.String("dcos"), // TODO @pires configurable CNI
				},
			},
		},
	}
}

// buildPodNameFromPod is a helper for building the "key" for the providers pod store.
func buildPodName(podNamespace, podName string) (string, error) {
	if podNamespace == "" {
		return "", fmt.Errorf("pod namespace not found")
	}

	if podName == "" {
		return "", fmt.Errorf("pod name not found")
	}

	return fmt.Sprintf("%s-%s", podNamespace, podName), nil
}

// buildPodNameFromPod is a helper for building the "key" for the providers pod store.
func buildPodNameFromPod(pod *corev1.Pod) (string, error) {
	return buildPodName(pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
}
