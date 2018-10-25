# Mesos provider

TODO @pires add a description of the provider and how it works

## Requirements

* Docker and Docker Compose
* `cfssl` and `cfssljson`
* Generate TLS certificates - this is a one-time only step:
  ```shell
  $ ./providers/mesos/hack/setup-certs.sh
  ```

## Bring-up development infrastructure

```shell
$ docker-compose -f providers/mesos/docker-compose.yml up
```

## Run VK with Mesos provider

First, build and run the development container:

```shell
$ docker build -f providers/mesos/Dockerfile -t vkdev .
 
$ docker run --name vkdev --network mesos_default -it --rm -v $(PWD):/go/src/github.com/virtual-kubelet/virtual-kubelet vkdev bash
```

Next, from within the development container, setup the Kubernetes client configuration:

```shell
$ cat > kubeconfig << EOF
apiVersion: v1
clusters:
- cluster:
    server: http://kube-apiserver:8181
  name: default-cluster
contexts:
- context:
    cluster: default-cluster
    user: ""
  name: vkdev
current-context: vkdev
kind: Config
preferences: {}
users: []
EOF
```

Just in case you haven't done so, please run:

```shell
$ dep ensure -v
```

Now, build:

```shell
$ make VK_BUILD_TAGS="no_alicloud_provider no_aws_provider no_azure_provider no_azurebatch_provider no_cri_provider no_huawei_provider no_hyper_provider no_mock_provider no_sfmesh_provider no_vic_provider no_web_provider" build
```

You're now ready to run VK with the Mesos provider:

```shell
$ APISERVER_CERT_LOCATION=./providers/mesos/hack/certs/virtual-kubelet-crt.pem \
  APISERVER_KEY_LOCATION=./providers/mesos/hack/certs/virtual-kubelet-key.pem \
  KUBELET_PORT=10250 \
  bin/virtual-kubelet --kubeconfig=kubeconfig --provider=mesos --provider-config=${PWD}/providers/mesos/mesos.toml
```

You should see output like follows:

```
2018/11/08 13:20:15 Initializing the Mesos provider.
2018/11/08 13:20:15 Initializing the Mesos scheduler.
2018/11/08 13:20:15 Mesos scheduler initialized.
2018/11/08 13:20:15 Mesos provider initialized: &{config:0xc00028bb80 nodeName:virtual-kubelet operatingSystem:Linux internalIP: daemonEndpointPort:10250 lastTransitionTime:{wall:0 ext:0 loc:<nil>} mesosScheduler:0xc000782620}.
2018/11/08 13:20:15 Mesos scheduler running with configuration: &{MesosURL:http://mesos:5050/api/v1/scheduler Principal: Name:vk_mesos Role:* Codec:{Codec:{Name:protobuf Type:application/x-protobuf NewEncoder:0x145c960 NewDecoder:0x145ca20}} Timeout:20s FailoverTimeout:1000h0m0s Checkpoint:true Verbose:true ReviveBurst:3 ReviveWait:1s Metrics:0xc000913ad0 MaxRefuseSeconds:5s JobRestartDelay:5s User:root}
2018/11/08 13:20:15 connecting...
INFO[0000] Registered node                               namespace= node=virtual-kubelet operatingSystem=Linux provider=mesos
2018/11/08 13:20:16 received GetPods
INFO[0001] Start to run pod cache controller.            namespace= node=virtual-kubelet operatingSystem=Linux provider=mesos
2018/11/08 13:20:16 scheduler client notification: {Type:connected}
2018/11/08 13:20:16 {Type:SUBSCRIBED Subscribed:&Event_Subscribed{FrameworkID:&mesos.FrameworkID{Value:fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0001,},HeartbeatIntervalSeconds:*15,MasterInfo:&mesos.MasterInfo{ID:fdb3e2a0-a79e-402a-9589-9be9b18afaf6,IP:16777343,Port:*5050,PID:*master@127.0.0.1:5050,Hostname:*127.0.0.1,Version:*1.8.0,Address:&Address{Hostname:*127.0.0.1,IP:*127.0.0.1,Port:5050,},Domain:nil,Capabilities:[{AGENT_UPDATE}],},} Offers:nil InverseOffers:nil Rescind:nil RescindInverseOffer:nil Update:nil UpdateOperationStatus:nil Message:nil Failure:nil Error:nil}
2018/11/08 13:20:16 FrameworkID fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0001
2018/11/08 13:20:16 {Type:HEARTBEAT Subscribed:nil Offers:nil InverseOffers:nil Rescind:nil RescindInverseOffer:nil Update:nil UpdateOperationStatus:nil Message:nil Failure:nil Error:nil}
2018/11/08 13:20:16 {Type:OFFERS Subscribed:nil Offers:&Event_Offers{Offers:[{{fdb3e2a0-a79e-402a-9589-9be9b18afaf6-O3} {fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0001} {fdb3e2a0-a79e-402a-9589-9be9b18afaf6-S0} 127.0.0.1 URL{Scheme:http,Address:Address{Hostname:*127.0.0.1,IP:*127.0.0.1,Port:5051,},Path:*/slave(1),Query:[],Fragment:nil,} nil [{nil cpus SCALAR &Value_Scalar{Value:2,} nil nil <nil> nil nil [] nil nil nil} {nil mem SCALAR &Value_Scalar{Value:999,} nil nil <nil> nil nil [] nil nil nil} {nil disk SCALAR &Value_Scalar{Value:54699,} nil nil <nil> nil nil [] nil nil nil} {nil ports RANGES nil &Value_Ranges{Range:[{31000 32000}],} nil <nil> nil nil [] nil nil nil}] [] [] nil nil}],} InverseOffers:nil Rescind:nil RescindInverseOffer:nil Update:nil UpdateOperationStatus:nil Message:nil Failure:nil Error:nil}
2018/11/08 13:20:16 received offer id "fdb3e2a0-a79e-402a-9589-9be9b18afaf6-O3" with resources "cpus:2;mem:999;disk:54699;ports:[31000-32000]"
2018/11/08 13:20:16 no new pods. rejecting offer with id "fdb3e2a0-a79e-402a-9589-9be9b18afaf6-O3"
2018/11/08 13:20:16 zero tasks launched this cycle
(...)
```

No Kubernetes pods are assigned to the VK, so the Mesos provider has no work to do.
In the future, the provider should enter `SUPRESS` mode until there's work to do.

## Create a test pod

First, be sure to be inside the development container:

```shell
$ docker exec -it vkdev bash
```

Second, be sure to have VK registered as a node:

```shell
$ ./kubectl --kubeconfig=kubeconfig get nodes
  NAME              STATUS   ROLES   AGE   VERSION
  virtual-kubelet   Ready    agent   22m   v1.11.2
```

Now, create the test pod:

```shell
$ cat <<EOF | ./kubectl --kubeconfig ./kubeconfig create -f -
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  containers:
  - name: nginx
    image: nginx:1.15.5-alpine
    resources:
      requests:
        memory: "128Mi"
        cpu: "250m"
      limits:
        memory: "128Mi"
        cpu: "250m"
    ports:
    - containerPort: 80
  automountServiceAccountToken: false
  tolerations:
  - key: "virtual-kubelet.io/provider"
    operator: "Equal"
    value: "mesos"
    effect: "NoSchedule"
EOF
```

It is imperative that any pod spec you come up with meets the following criteria:

- Has resource`requests` and `limits`.
 Mesos has no notion of running tasks without resource limits being clearly defined.
 In the future, the provider should enforce defaults when no resources are set.

- `tolerations` must be set **exactly** as above.

Looking at VK logs, you should see output like follows:

```
(...)
2018/11/08 14:22:11 receivedd CreatePod "nginx"
INFO[0000] Pod created                                   namespace=default node=virtual-kubelet operatingSystem=Linux pod=nginx provider=mesos
INFO[0000] Start to run pod cache controller.            namespace= node=virtual-kubelet operatingSystem=Linux provider=mesos
2018/11/08 14:22:11 zero tasks launched this cycle
2018/11/08 14:22:12 {Type:OFFERS Subscribed:nil Offers:&Event_Offers{Offers:[{{fdb3e2a0-a79e-402a-9589-9be9b18afaf6-O942} {fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004} {fdb3e2a0-a79e-402a-9589-9be9b18afaf6-S0} 127.0.0.1 URL{Scheme:http,Address:Address{Hostname:*127.0.0.1,IP:*127.0.0.1,Port:5051,},Path:*/slave(1),Query:[],Fragment:nil,} nil [{nil cpus SCALAR &Value_Scalar{Value:2,} nil nil <nil> nil nil [] nil nil nil} {nil mem SCALAR &Value_Scalar{Value:999,} nil nil <nil> nil nil [] nil nil nil} {nil disk SCALAR &Value_Scalar{Value:54699,} nil nil <nil> nil nil [] nil nil nil} {nil ports RANGES nil &Value_Ranges{Range:[{31000 32000}],} nil <nil> nil nil [] nil nil nil}] [] [] nil nil}],} InverseOffers:nil Rescind:nil RescindInverseOffer:nil Update:nil UpdateOperationStatus:nil Message:nil Failure:nil Error:nil}
2018/11/08 14:22:12 received offer id "fdb3e2a0-a79e-402a-9589-9be9b18afaf6-O942" with resources "cpus:2;mem:999;disk:54699;ports:[31000-32000]"
2018/11/08 14:22:12 Pod "default-nginx" wants the following resources "cpus:0.1;mem:128;disk:128"
2018/11/08 14:22:12 Pod "default-nginx" has 1 containers
2018/11/08 14:22:12 Container "nginx" wants resources "cpus:0.1;mem:128;disk:128"
2018/11/08 14:22:12 launching pod "default-nginx" using offer "fdb3e2a0-a79e-402a-9589-9be9b18afaf6-O942"
ExecutorInfo: "&ExecutorInfo{ExecutorID:ExecutorID{Value:exec-default-nginx,},Data:nil,Resources:[{nil cpus SCALAR Value_Scalar{Value:0.1,} nil nil <nil> nil nil [] nil nil nil} {nil mem SCALAR &Value_Scalar{Value:32,} nil nil <nil> nil nil [] nil nil nil} {nil disk SCALAR &Value_Scalar{Value:256,} nil nil <nil> nil nil [] nil nil nil}],Command:nil,FrameworkID:&FrameworkID{Value:fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004,},Name:nil,Source:nil,Container:&ContainerInfo{Type:*MESOS,Volumes:[],Docker:nil,Hostname:nil,Mesos:nil,NetworkInfos:[{[{<nil> <nil>}] <nil> [] nil []}],LinuxInfo:nil,RlimitInfo:nil,TTYInfo:nil,},Discovery:nil,ShutdownGracePeriod:nil,Labels:nil,Type:DEFAULT,}"
TaskGroupInfo: "&TaskGroupInfo{Tasks:[{ {default-nginx-nginx} {fdb3e2a0-a79e-402a-9589-9be9b18afaf6-S0} [{nil cpus SCALAR Value_Scalar{Value:0.1,} nil nil <nil> nil nil [] nil nil nil} {nil mem SCALAR &Value_Scalar{Value:128,} nil nil <nil> nil nil [] nil nil nil} {nil disk SCALAR &Value_Scalar{Value:128,} nil nil <nil> nil nil [] nil nil nil}] nil &CommandInfo{URIs:[],Environment:&Environment{Variables:[],},Value:nil,User:nil,Shell:*false,Arguments:[],} &ContainerInfo{Type:*MESOS,Volumes:[],Docker:nil,Hostname:nil,Mesos:&ContainerInfo_MesosInfo{Image:&Image{Type:*DOCKER,Appc:nil,Docker:&Image_Docker{Name:nginx:1.15.5-alpine,Credential:nil,Config:nil,},Cached:nil,},},NetworkInfos:[],LinuxInfo:nil,RlimitInfo:nil,TTYInfo:nil,} nil nil nil [] nil nil nil}],}"
2018/11/08 14:22:13 {Type:UPDATE Subscribed:nil Offers:nil InverseOffers:nil Rescind:nil RescindInverseOffer:nil Update:&Event_Update{Status:mesos.TaskStatus{TaskID:TaskID{Value:default-nginx-nginx,},State:*TASK_STARTING,Data:nil,Message:nil,AgentID:&AgentID{Value:fdb3e2a0-a79e-402a-9589-9be9b18afaf6-S0,},Timestamp:*1.54168693326496e+09,ExecutorID:&ExecutorID{Value:exec-default-nginx,},Healthy:nil,Source:*SOURCE_EXECUTOR,Reason:nil,UUID:*[229 135 243 56 47 80 78 15 129 13 165 60 96 160 141 240],Labels:nil,ContainerStatus:&ContainerStatus{NetworkInfos:[{[{<nil> 0xc000556850}] <nil> [] nil []}],CgroupInfo:nil,ExecutorPID:nil,ContainerID:&ContainerID{Value:76b0b90a-63db-4f43-b055-92746bfc5f84,Parent:&ContainerID{Value:e32698fe-7f3b-4f90-a40d-fb97b62c24a4,Parent:nil,},},},UnreachableTime:nil,CheckStatus:nil,Limitation:nil,},} UpdateOperationStatus:nil Message:nil Failure:nil Error:nil}
2018/11/08 14:22:13 Task default-nginx-nginx is in store TASK_STARTING
2018/11/08 14:22:14 {Type:UPDATE Subscribed:nil Offers:nil InverseOffers:nil Rescind:nil RescindInverseOffer:nil Update:&Event_Update{Status:mesos.TaskStatus{TaskID:TaskID{Value:default-nginx-nginx,},State:*TASK_RUNNING,Data:nil,Message:nil,AgentID:&AgentID{Value:fdb3e2a0-a79e-402a-9589-9be9b18afaf6-S0,},Timestamp:*1.541686934460822e+09,ExecutorID:&ExecutorID{Value:exec-default-nginx,},Healthy:nil,Source:*SOURCE_EXECUTOR,Reason:nil,UUID:*[139 76 6 242 90 70 74 111 146 211 85 130 202 107 205 217],Labels:nil,ContainerStatus:&ContainerStatus{NetworkInfos:[{[{<nil> 0xc0005bb010}] <nil> [] nil []}],CgroupInfo:nil,ExecutorPID:*923,ContainerID:&ContainerID{Value:76b0b90a-63db-4f43-b055-92746bfc5f84,Parent:&ContainerID{Value:e32698fe-7f3b-4f90-a40d-fb97b62c24a4,Parent:nil,},},},UnreachableTime:nil,CheckStatus:nil,Limitation:nil,},} UpdateOperationStatus:nil Message:nil Failure:nil Error:nil}
2018/11/08 14:22:14 Task default-nginx-nginx is in store TASK_RUNNING
```

Now, let's check Mesos.
Start a new terminal and enter the Mesos container:

```shell
$ docker exec -it mesos_mesos_1 bash
```

You are ready to check agents logs:

```shel
$ tail -f /var/log/mesos/mesos-agent.INFO
(...)
I1108 14:22:12.784819   156 slave.cpp:2014] Got assigned task group containing tasks [ default-nginx-nginx ] for framework fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004
I1108 14:22:12.785959   156 slave.cpp:2388] Authorizing task group containing tasks [ default-nginx-nginx ] for framework fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004
I1108 14:22:12.786568   153 slave.cpp:2831] Launching task group containing tasks [ default-nginx-nginx ] for framework fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004
I1108 14:22:12.787173   153 paths.cpp:752] Creating sandbox '/var/lib/mesos/agent/slaves/fdb3e2a0-a79e-402a-9589-9be9b18afaf6-S0/frameworks/fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004/executors/exec-default-nginx/runs/e32698fe-7f3b-4f90-a40d-fb97b62c24a4' for user 'root'
I1108 14:22:12.788005   153 paths.cpp:755] Creating sandbox '/var/lib/mesos/agent/meta/slaves/fdb3e2a0-a79e-402a-9589-9be9b18afaf6-S0/frameworks/fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004/executors/exec-default-nginx/runs/e32698fe-7f3b-4f90-a40d-fb97b62c24a4'
I1108 14:22:12.788245   153 slave.cpp:8994] Launching executor 'exec-default-nginx' of framework fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004 with resources [{"allocation_info":{"role":"*"},"name":"cpus","scalar":{"value":0.1},"type":"SCALAR"},{"allocation_info":{"role":"*"},"name":"mem","scalar":{"value":32.0},"type":"SCALAR"},{"allocation_info":{"role":"*"},"name":"disk","scalar":{"value":256.0},"type":"SCALAR"}] in work directory '/var/lib/mesos/agent/slaves/fdb3e2a0-a79e-402a-9589-9be9b18afaf6-S0/frameworks/fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004/executors/exec-default-nginx/runs/e32698fe-7f3b-4f90-a40d-fb97b62c24a4'
I1108 14:22:12.788493   153 slave.cpp:3509] Launching container e32698fe-7f3b-4f90-a40d-fb97b62c24a4 for executor 'exec-default-nginx' of framework fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004
I1108 14:22:12.789125   153 slave.cpp:3028] Queued task group containing tasks [ default-nginx-nginx ] for executor 'exec-default-nginx' of framework fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004
I1108 14:22:12.789624   157 docker.cpp:1175] Skipping non-docker container
I1108 14:22:12.789935   157 containerizer.cpp:1280] Starting container e32698fe-7f3b-4f90-a40d-fb97b62c24a4
I1108 14:22:12.793192   157 containerizer.cpp:3122] Transitioning the state of container e32698fe-7f3b-4f90-a40d-fb97b62c24a4 from PROVISIONING to PREPARING
I1108 14:22:12.796303   153 memory.cpp:478] Started listening for OOM events for container e32698fe-7f3b-4f90-a40d-fb97b62c24a4
I1108 14:22:12.796411   153 memory.cpp:590] Started listening on 'low' memory pressure events for container e32698fe-7f3b-4f90-a40d-fb97b62c24a4
I1108 14:22:12.796464   153 memory.cpp:590] Started listening on 'medium' memory pressure events for container e32698fe-7f3b-4f90-a40d-fb97b62c24a4
I1108 14:22:12.796515   153 memory.cpp:590] Started listening on 'critical' memory pressure events for container e32698fe-7f3b-4f90-a40d-fb97b62c24a4
I1108 14:22:12.796689   153 memory.cpp:198] Updated 'memory.soft_limit_in_bytes' to 32MB for container e32698fe-7f3b-4f90-a40d-fb97b62c24a4
I1108 14:22:12.796728   154 cpu.cpp:92] Updated 'cpu.shares' to 102 (cpus 0.1) for container e32698fe-7f3b-4f90-a40d-fb97b62c24a4
I1108 14:22:12.797696   153 memory.cpp:227] Updated 'memory.limit_in_bytes' to 32MB for container e32698fe-7f3b-4f90-a40d-fb97b62c24a4
I1108 14:22:12.799374   155 switchboard.cpp:316] Container logger module finished preparing container e32698fe-7f3b-4f90-a40d-fb97b62c24a4; IOSwitchboard server is not required
I1108 14:22:12.799979   158 linux_launcher.cpp:492] Launching container e32698fe-7f3b-4f90-a40d-fb97b62c24a4 and cloning with namespaces CLONE_NEWNS | CLONE_NEWPID
I1108 14:22:12.802342   155 containerizer.cpp:2044] Checkpointing container's forked pid 900 to '/var/lib/mesos/agent/meta/slaves/fdb3e2a0-a79e-402a-9589-9be9b18afaf6-S0/frameworks/fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004/executors/exec-default-nginx/runs/e32698fe-7f3b-4f90-a40d-fb97b62c24a4/pids/forked.pid'
I1108 14:22:12.803354   155 containerizer.cpp:3122] Transitioning the state of container e32698fe-7f3b-4f90-a40d-fb97b62c24a4 from PREPARING to ISOLATING
I1108 14:22:12.806640   153 containerizer.cpp:3122] Transitioning the state of container e32698fe-7f3b-4f90-a40d-fb97b62c24a4 from ISOLATING to FETCHING
I1108 14:22:12.807473   153 containerizer.cpp:3122] Transitioning the state of container e32698fe-7f3b-4f90-a40d-fb97b62c24a4 from FETCHING to RUNNING
I1108 14:22:13.258993   156 http.cpp:1177] HTTP POST for /slave(1)/api/v1/executor from 127.0.0.1:41614
I1108 14:22:13.259083   156 slave.cpp:4607] Received Subscribe request for HTTP executor 'exec-default-nginx' of framework fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004
I1108 14:22:13.259112   156 slave.cpp:4670] Creating a marker file for HTTP based executor 'exec-default-nginx' of framework fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004 (via HTTP) at path '/var/lib/mesos/agent/meta/slaves/fdb3e2a0-a79e-402a-9589-9be9b18afaf6-S0/frameworks/fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004/executors/exec-default-nginx/runs/e32698fe-7f3b-4f90-a40d-fb97b62c24a4/http.marker'
I1108 14:22:13.259613   154 disk.cpp:213] Updating the disk resources for container e32698fe-7f3b-4f90-a40d-fb97b62c24a4 to cpus(allocated: *):0.2; mem(allocated: *):160; disk(allocated: *):384
I1108 14:22:13.259678   156 memory.cpp:198] Updated 'memory.soft_limit_in_bytes' to 160MB for container e32698fe-7f3b-4f90-a40d-fb97b62c24a4
I1108 14:22:13.259698   154 cpu.cpp:92] Updated 'cpu.shares' to 204 (cpus 0.2) for container e32698fe-7f3b-4f90-a40d-fb97b62c24a4
I1108 14:22:13.259739   156 memory.cpp:227] Updated 'memory.limit_in_bytes' to 160MB for container e32698fe-7f3b-4f90-a40d-fb97b62c24a4
I1108 14:22:13.262405   160 slave.cpp:3282] Sending queued task group containing tasks [ default-nginx-nginx ] to executor 'exec-default-nginx' of framework fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004 (via HTTP)
I1108 14:22:13.265828   158 http.cpp:1177] HTTP POST for /slave(1)/api/v1 from 127.0.0.1:41618
I1108 14:22:13.266516   158 http.cpp:2408] Processing LAUNCH_NESTED_CONTAINER call for container 'e32698fe-7f3b-4f90-a40d-fb97b62c24a4.76b0b90a-63db-4f43-b055-92746bfc5f84'
I1108 14:22:13.266690   157 containerizer.cpp:1242] Creating sandbox '/var/lib/mesos/agent/slaves/fdb3e2a0-a79e-402a-9589-9be9b18afaf6-S0/frameworks/fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004/executors/exec-default-nginx/runs/e32698fe-7f3b-4f90-a40d-fb97b62c24a4/containers/76b0b90a-63db-4f43-b055-92746bfc5f84' for user 'root'
I1108 14:22:13.266957   157 containerizer.cpp:1280] Starting container e32698fe-7f3b-4f90-a40d-fb97b62c24a4.76b0b90a-63db-4f43-b055-92746bfc5f84
I1108 14:22:13.267918   160 http.cpp:1177] HTTP POST for /slave(1)/api/v1/executor from 127.0.0.1:41616
I1108 14:22:13.268025   160 slave.cpp:5269] Handling status update TASK_STARTING (Status UUID: e587f338-2f50-4e0f-810d-a53c60a08df0) for task default-nginx-nginx of framework fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004
I1108 14:22:13.268322   157 provisioner.cpp:545] Provisioning image rootfs '/var/lib/mesos/agent/provisioner/containers/e32698fe-7f3b-4f90-a40d-fb97b62c24a4/containers/76b0b90a-63db-4f43-b055-92746bfc5f84/backends/copy/rootfses/c9cdfb70-ff4a-42b6-9724-20b3c5d4067c' for container e32698fe-7f3b-4f90-a40d-fb97b62c24a4.76b0b90a-63db-4f43-b055-92746bfc5f84 using copy backend
W1108 14:22:13.271303   157 containerizer.cpp:2399] Skipping status for container e32698fe-7f3b-4f90-a40d-fb97b62c24a4.76b0b90a-63db-4f43-b055-92746bfc5f84 because: Container does not exist
I1108 14:22:13.271797   157 task_status_update_manager.cpp:328] Received task status update TASK_STARTING (Status UUID: e587f338-2f50-4e0f-810d-a53c60a08df0) for task default-nginx-nginx of framework fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004
I1108 14:22:13.272035   157 task_status_update_manager.cpp:842] Checkpointing UPDATE for task status update TASK_STARTING (Status UUID: e587f338-2f50-4e0f-810d-a53c60a08df0) for task default-nginx-nginx of framework fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004
I1108 14:22:13.272173   157 slave.cpp:5761] Forwarding the update TASK_STARTING (Status UUID: e587f338-2f50-4e0f-810d-a53c60a08df0) for task default-nginx-nginx of framework fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004 to master@127.0.0.1:5050
I1108 14:22:13.279791   158 task_status_update_manager.cpp:401] Received task status update acknowledgement (UUID: e587f338-2f50-4e0f-810d-a53c60a08df0) for task default-nginx-nginx of framework fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004
I1108 14:22:13.281401   158 task_status_update_manager.cpp:842] Checkpointing ACK for task status update TASK_STARTING (Status UUID: e587f338-2f50-4e0f-810d-a53c60a08df0) for task default-nginx-nginx of framework fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004
I1108 14:22:14.358237   154 containerizer.cpp:3122] Transitioning the state of container e32698fe-7f3b-4f90-a40d-fb97b62c24a4.76b0b90a-63db-4f43-b055-92746bfc5f84 from PROVISIONING to PREPARING
I1108 14:22:14.371640   155 switchboard.cpp:316] Container logger module finished preparing container e32698fe-7f3b-4f90-a40d-fb97b62c24a4.76b0b90a-63db-4f43-b055-92746bfc5f84; IOSwitchboard server is not required
I1108 14:22:14.372280   156 linux_launcher.cpp:492] Launching nested container e32698fe-7f3b-4f90-a40d-fb97b62c24a4.76b0b90a-63db-4f43-b055-92746bfc5f84 and cloning with namespaces CLONE_NEWNS | CLONE_NEWPID
I1108 14:22:14.375890   158 containerizer.cpp:3122] Transitioning the state of container e32698fe-7f3b-4f90-a40d-fb97b62c24a4.76b0b90a-63db-4f43-b055-92746bfc5f84 from PREPARING to ISOLATING
I1108 14:22:14.458549   153 containerizer.cpp:3122] Transitioning the state of container e32698fe-7f3b-4f90-a40d-fb97b62c24a4.76b0b90a-63db-4f43-b055-92746bfc5f84 from ISOLATING to FETCHING
I1108 14:22:14.459161   154 containerizer.cpp:3122] Transitioning the state of container e32698fe-7f3b-4f90-a40d-fb97b62c24a4.76b0b90a-63db-4f43-b055-92746bfc5f84 from FETCHING to RUNNING
I1108 14:22:14.464471   153 http.cpp:1177] HTTP POST for /slave(1)/api/v1 from 127.0.0.1:41620
I1108 14:22:14.464613   153 http.cpp:2643] Processing WAIT_NESTED_CONTAINER call for container 'e32698fe-7f3b-4f90-a40d-fb97b62c24a4.76b0b90a-63db-4f43-b055-92746bfc5f84'
I1108 14:22:14.520704   158 http.cpp:1177] HTTP POST for /slave(1)/api/v1/executor from 127.0.0.1:41616
I1108 14:22:14.520828   158 slave.cpp:5269] Handling status update TASK_RUNNING (Status UUID: 8b4c06f2-5a46-4a6f-92d3-5582ca6bcdd9) for task default-nginx-nginx of framework fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004
I1108 14:22:14.522142   154 task_status_update_manager.cpp:328] Received task status update TASK_RUNNING (Status UUID: 8b4c06f2-5a46-4a6f-92d3-5582ca6bcdd9) for task default-nginx-nginx of framework fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004
I1108 14:22:14.522181   154 task_status_update_manager.cpp:842] Checkpointing UPDATE for task status update TASK_RUNNING (Status UUID: 8b4c06f2-5a46-4a6f-92d3-5582ca6bcdd9) for task default-nginx-nginx of framework fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004
I1108 14:22:14.522266   154 slave.cpp:5761] Forwarding the update TASK_RUNNING (Status UUID: 8b4c06f2-5a46-4a6f-92d3-5582ca6bcdd9) for task default-nginx-nginx of framework fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004 to master@127.0.0.1:5050
I1108 14:22:14.526805   160 task_status_update_manager.cpp:401] Received task status update acknowledgement (UUID: 8b4c06f2-5a46-4a6f-92d3-5582ca6bcdd9) for task default-nginx-nginx of framework fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004
I1108 14:22:14.526857   160 task_status_update_manager.cpp:842] Checkpointing ACK for task status update TASK_RUNNING (Status UUID: 8b4c06f2-5a46-4a6f-92d3-5582ca6bcdd9) for task default-nginx-nginx of framework fdb3e2a0-a79e-402a-9589-9be9b18afaf6-0004
```

And here's more proof that the test pod is running:

```shell
$ ps -aef --forest
UID        PID  PPID  C STIME TTY          TIME CMD
root       923     0  0 14:22 ?        00:00:00 /usr/libexec/mesos/mesos-containerizer launch
root       925   923  0 14:22 ?        00:00:00  \_ nginx: master process nginx -g daemon off;
100        926   925  0 14:22 ?        00:00:00      \_ nginx: worker process
```

## Cleanup

### Remove test pods

First, be sure to be inside the development container:

```shell
$ docker exec -it vkdev bash
```

Next, **forcibly** remove the pod:

```shell
$ ./kubectl --kubeconfig=kubeconfig delete pod nginx --force
```

**Attention**: This **WILL NOT** terminate the Mesos task.
In the future, such behavior will be added.

### Tear-down development infrastructure

```shell
$ docker-compose -f providers/mesos/docker-compose.yml rm -sf
```

## Tips and Tricks

**Attention**: All tricks are meant to run within the development container, unless specified otherwise. 

- List Mesos agents:

```shell
$ curl -H "Content-Type: application/json" -d '{"type": "GET_AGENTS"}' mesos:5050/api/v1
```