package scheduler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/backoff"
	xmetrics "github.com/mesos/mesos-go/api/v1/lib/extras/metrics"
	"github.com/mesos/mesos-go/api/v1/lib/extras/scheduler/callrules"
	"github.com/mesos/mesos-go/api/v1/lib/extras/scheduler/controller"
	"github.com/mesos/mesos-go/api/v1/lib/extras/scheduler/eventrules"
	"github.com/mesos/mesos-go/api/v1/lib/extras/store"
	"github.com/mesos/mesos-go/api/v1/lib/resources"
	"github.com/mesos/mesos-go/api/v1/lib/scheduler"
	"github.com/mesos/mesos-go/api/v1/lib/scheduler/calls"
	"github.com/mesos/mesos-go/api/v1/lib/scheduler/events"

	corev1 "k8s.io/api/core/v1"
)

var (
	RegistrationMinBackoff = 1 * time.Second
	RegistrationMaxBackoff = 15 * time.Second
)

type mesosScheduler struct {
	quit   chan struct{}
	config *Config
	store  *stateStore
}

type Scheduler interface {
	// TODO @pires providers are stateless :(
	// Run() (quit chan<- struct{}, err error)
	Run()
	AddPod(pod *corev1.Pod) error
	DeletePod(pod *corev1.Pod) error
	UpdatePod(pod *corev1.Pod) error
	GetPod(podNamespace, podName string) *corev1.Pod
	ListPods() []*corev1.Pod
}

func New(config *Config) *mesosScheduler {
	return &mesosScheduler{
		quit:   make(chan struct{}, 1),
		config: config,
		store:  newStateStore(config),
	}
}

func (sched *mesosScheduler) Run() {
	// TODO log errors when they happen
	// TODO recover panics and rerun
	//defer func() {
	//	if r := recover(); r != nil {
	sched.run()
	//	}
	//}()
}

func (sched *mesosScheduler) run() error {
	log.Printf("Mesos scheduler running with configuration: %+v", sched.config)

	ctx, _ := context.WithCancel(context.Background())

	// TODO(jdef) how to track/handle timeout errors that occur for SUBSCRIBE calls? we should
	// probably tolerate X number of subsequent subscribe failures before bailing. we'll need
	// to track the lastCallAttempted along with subsequentSubscribeTimeouts.

	fidStore := store.DecorateSingleton(
		store.NewInMemorySingleton(),
		store.DoSet().AndThen(func(_ store.Setter, v string, _ error) error {
			log.Println("FrameworkID", v)
			return nil
		}))

	sched.store.cli = callrules.New(
		callrules.WithFrameworkID(store.GetIgnoreErrors(fidStore)),
		logCalls(map[scheduler.Call_Type]string{scheduler.Call_SUBSCRIBE: "connecting..."}),
		callMetrics(sched.store.metricsAPI, time.Now, true),
	).Caller(sched.store.cli)

	err := controller.Run(
		ctx,
		buildFrameworkInfo(sched.store.config),
		sched.store.cli,
		controller.WithEventHandler(buildEventHandler(sched.store, fidStore)),
		controller.WithFrameworkID(store.GetIgnoreErrors(fidStore)),
		controller.WithRegistrationTokens(
			backoff.Notifier(RegistrationMinBackoff, RegistrationMaxBackoff, ctx.Done()),
		),
		controller.WithSubscriptionTerminated(func(err error) {
			if err != nil {
				if err != io.EOF {
					log.Println(err)
				}
				return
			}
			log.Println("disconnected")
		}),
	)
	if sched.store.err != nil {
		err = sched.store.err
	}

	// wait to handle explicit quit
	go func() {
		<-sched.quit
		// TODO quit what exactly?
	}()

	//return sched.quit, err
	return err
}

func (sched *mesosScheduler) AddPod(pod *corev1.Pod) error {
	podKey, err := buildPodNameFromPod(pod)
	if err != nil {
		return err
	}

	sched.store.newPodMap.Set(podKey, &mesosPod{pod, buildPodTasks(pod)})

	return nil
}
func (sched *mesosScheduler) DeletePod(pod *corev1.Pod) error {
	// TODO signal pod delete
	return errors.New("TODO")
}
func (sched *mesosScheduler) UpdatePod(pod *corev1.Pod) error {
	// TODO signal pod update
	return errors.New("TODO")
}
func (sched *mesosScheduler) GetPod(podNamespace, podName string) *corev1.Pod {
	podKey, err := buildPodName(podNamespace, podName)
	if err == nil {
		if pod, ok := sched.store.newPodMap.Get(podKey); ok {
			return pod.pod
		}
		if pod, ok := sched.store.runningPodMap.Get(podKey); ok {
			return pod.pod
		}
	}

	return nil
}

func (sched *mesosScheduler) ListPods() []*corev1.Pod {
	// TODO should we return non-running pods too?
	pods := make([]*corev1.Pod, sched.store.runningPodMap.Count(), sched.store.runningPodMap.Count())
	for _, mesosPod := range sched.store.runningPodMap.Iter() {
		pods = append(pods, mesosPod.pod)
	}

	return pods
}

// buildEventHandler generates and returns a handler to process events received from the subscription.
func buildEventHandler(store *stateStore, fidStore store.Singleton) events.Handler {
	// disable brief logs when verbose logs are enabled (there's no sense logging twice!)
	logger := controller.LogEvents(nil).Unless(store.config.Verbose)
	return eventrules.New(
		logAllEvents().If(store.config.Verbose),
		eventMetrics(store.metricsAPI, time.Now, true),
		controller.LiftErrors().DropOnError(),
	).Handle(events.Handlers{
		scheduler.Event_FAILURE: logger.HandleF(failure),
		scheduler.Event_OFFERS:  trackOffersReceived(store).HandleF(resourceOffers(store)),
		scheduler.Event_UPDATE:  controller.AckStatusUpdates(store.cli).AndThen().HandleF(statusUpdate(store)),
		scheduler.Event_SUBSCRIBED: eventrules.New(
			logger,
			controller.TrackSubscription(fidStore, store.config.FailoverTimeout),
		),
	}.Otherwise(logger.HandleEvent))
}

func trackOffersReceived(store *stateStore) eventrules.Rule {
	return func(ctx context.Context, e *scheduler.Event, err error, chain eventrules.Chain) (context.Context, *scheduler.Event, error) {
		if err == nil {
			store.metricsAPI.offersReceived.Int(len(e.GetOffers().GetOffers()))
		}
		return chain(ctx, e, err)
	}
}

func failure(_ context.Context, e *scheduler.Event) error {
	var (
		f              = e.GetFailure()
		eid, aid, stat = f.ExecutorID, f.AgentID, f.Status
	)
	if eid != nil {
		// executor failed..
		msg := "executor '" + eid.Value + "' terminated"
		if aid != nil {
			msg += " on agent '" + aid.Value + "'"
		}
		if stat != nil {
			msg += " with status=" + strconv.Itoa(int(*stat))
		}
		log.Println(msg)
	} else if aid != nil {
		// agent failed..
		log.Println("agent '" + aid.Value + "' terminated")
	}
	return nil
}

func resourceOffers(store *stateStore) events.HandlerFunc {
	return func(ctx context.Context, e *scheduler.Event) error {
		var (
			offers                 = e.GetOffers().GetOffers()
			callOption             = calls.RefuseSecondsWithJitter(rand.New(rand.NewSource(time.Now().Unix())), store.config.MaxRefuseSeconds)
			tasksLaunchedThisCycle = 0
			offersDeclined         = 0
			executorWantsResources = mesos.Resources{
				resources.NewCPUs(0.1).Resource,
				resources.NewMemory(32).Resource,
				resources.NewDisk(256).Resource,
			}
		)

		for i := range offers {
			var (
				remainingOfferedResources = mesos.Resources(offers[i].Resources)
			)

			if store.config.Verbose {
				log.Printf("received offer id %q with resources %q\n", offers[i].ID.Value, remainingOfferedResources.String())
			}

			// decline if there are no new pods
			if store.newPodMap.Count() == 0 {
				log.Printf("no new pods. rejecting offer with id %q\n", offers[i].ID.Value)
				// send Reject call to Mesos
				reject := calls.Decline(offers[i].ID).With(callOption)
				err := calls.CallNoData(ctx, store.cli, reject)
				if err != nil {
					log.Printf("failed to reject offer with id %q. err %+v\n", offers[i].ID.Value, err)
				}
				offersDeclined++
				continue
			}

			firstNewPodName := store.newPodMap.Keys()[0]
			pod, _ := store.newPodMap.Get(firstNewPodName)

			flattened := remainingOfferedResources.ToUnreserved()

			// TODO @pires this only works if requests are defined
			taskGroupWantsResources := sumPodResources(pod.pod)

			if store.config.Verbose {
				log.Printf("Pod %q wants the following resources %q", firstNewPodName, taskGroupWantsResources.String())
			}

			// decline if there offer doesn't fit pod (executor + tasks) resources request
			if !resources.ContainsAll(flattened, executorWantsResources.Plus(taskGroupWantsResources...)) {
				log.Printf("not enough resources in offer. rejecting offer with id %q\n", offers[i].ID.Value)
				// send Reject call to Mesos
				reject := calls.Decline(offers[i].ID).With(callOption)
				err := calls.CallNoData(ctx, store.cli, reject)
				if err != nil {
					log.Printf("failed to reject offer with id %q. err %+v\n", offers[i].ID.Value, err)
				}
				offersDeclined++
				continue
			}

			for name, restype := range resources.TypesOf(flattened...) {
				if restype == mesos.SCALAR {
					sum, _ := name.Sum(flattened...)
					store.metricsAPI.offeredResources(sum.GetScalar().GetValue(), name.String())
				}
			}

			processedTasks := pod.tasks

			if store.config.Verbose {
				log.Printf("Pod %q has %d containers\n", firstNewPodName, len(pod.pod.Spec.Containers))
			}

			// Prepare executor
			executorInfo := buildDefaultExecutorInfo(offers[i].FrameworkID)
			found := func() mesos.Resources {
				if store.config.Role == "*" {
					return resources.Find(executorWantsResources, flattened...)
				}
				reservation := mesos.Resource_ReservationInfo{
					Type: mesos.Resource_ReservationInfo_STATIC.Enum(),
					Role: &store.config.Role,
				}
				return resources.Find(executorWantsResources.PushReservation(reservation))
			}()
			executorInfo.ExecutorID = mesos.ExecutorID{Value: "exec-" + firstNewPodName}
			executorInfo.Resources = found
			remainingOfferedResources.Subtract(found...)
			flattened = remainingOfferedResources.ToUnreserved()

			for pos, containerSpec := range pod.pod.Spec.Containers {
				taskWantsResources := calculateTaskResources(containerSpec)

				if store.config.Verbose {
					log.Printf("Container %q wants resources %q", containerSpec.Name, taskWantsResources.String())
				}

				//if resources.ContainsAll(flattened, taskWantsResources) {
				found := func() mesos.Resources {
					if store.config.Role == "*" {
						return resources.Find(taskWantsResources, flattened...)
					}
					reservation := mesos.Resource_ReservationInfo{
						Type: mesos.Resource_ReservationInfo_STATIC.Enum(),
						Role: &store.config.Role,
					}
					return resources.Find(taskWantsResources.PushReservation(reservation))
				}()

				if len(found) == 0 {
					log.Println("failed to find the resources that were supposedly contained")
				}

				if store.config.Verbose {
					log.Printf("launching pod %q using offer %q\n", firstNewPodName, offers[i].ID.Value)
				}

				task := pod.tasks[pos]
				task.AgentID = offers[i].AgentID
				task.Resources = found
				processedTasks[pos] = task // TODO is this assignment needed?
				remainingOfferedResources.Subtract(found...)
				flattened = remainingOfferedResources.ToUnreserved()
				//}
			}

			taskGroupInfo := mesos.TaskGroupInfo{Tasks: processedTasks} // only needed for printing

			fmt.Printf("ExecutorInfo: %q\n", executorInfo.String())
			fmt.Printf("TaskGroupInfo: %q\n", taskGroupInfo.String())

			// build Accept call to launch all of the tasks we've assembled
			accept := calls.Accept(
				calls.OfferOperations{calls.OpLaunchGroup(executorInfo, processedTasks...)}.WithOffers(offers[i].ID),
			).With(callOption)

			// send Accept call to Mesos
			err := calls.CallNoData(ctx, store.cli, accept)
			if err != nil {
				log.Printf("failed to launch tasks: %+v", err)
			} else {
				if n := len(processedTasks); n > 0 {
					tasksLaunchedThisCycle += n
				}
			}

			// move pod to running pod
			pod, ok := store.newPodMap.GetAndRemove(firstNewPodName)
			if ok {
				pod.tasks = processedTasks
				store.runningPodMap.Set(firstNewPodName, pod)
			} else {
				log.Printf("failed to move pod %+v to runningpod map", pod)
			}
		}

		store.metricsAPI.offersDeclined.Int(offersDeclined)
		store.metricsAPI.tasksLaunched.Int(tasksLaunchedThisCycle)
		store.metricsAPI.launchesPerOfferCycle(float64(tasksLaunchedThisCycle))
		if tasksLaunchedThisCycle == 0 && store.config.Verbose {
			log.Println("zero tasks launched this cycle")
		}
		return nil
	}
}

func statusUpdate(store *stateStore) events.HandlerFunc {
	return func(ctx context.Context, e *scheduler.Event) error {
		s := e.GetUpdate().GetStatus()
		if store.config.Verbose {
			msg := "Task " + s.TaskID.Value + " is in store " + s.GetState().String()
			if m := s.GetMessage(); m != "" {
				msg += " with message '" + m + "'"
			}
			log.Println(msg)
		}

		switch st := s.GetState(); st {
		case mesos.TASK_FINISHED:
			store.metricsAPI.tasksFinished()
			tryReviveOffers(ctx, store)

		case mesos.TASK_LOST, mesos.TASK_KILLED, mesos.TASK_FAILED, mesos.TASK_ERROR:
			store.err = errors.New("Exiting because task " + s.GetTaskID().Value +
				" is in an unexpected store " + st.String() +
				" with reason " + s.GetReason().String() +
				" from source " + s.GetSource().String() +
				" with message '" + s.GetMessage() + "'")
		}
		return nil
	}
}

func tryReviveOffers(ctx context.Context, store *stateStore) {
	// limit the rate at which we request offer revival
	select {
	case <-store.reviveTokens:
		// not done yet, revive offers!
		err := calls.CallNoData(ctx, store.cli, calls.Revive())
		if err != nil {
			log.Printf("failed to revive offers: %+v", err)
			return
		}
	default:
		// noop
	}
}

// logAllEvents logs every observed event; this is somewhat expensive to do
func logAllEvents() eventrules.Rule {
	return func(ctx context.Context, e *scheduler.Event, err error, ch eventrules.Chain) (context.Context, *scheduler.Event, error) {
		log.Printf("%+v\n", *e)
		return ch(ctx, e, err)
	}
}

// eventMetrics logs metrics for every processed API event
func eventMetrics(metricsAPI *metricsAPI, clock func() time.Time, timingMetrics bool) eventrules.Rule {
	timed := metricsAPI.eventReceivedLatency
	if !timingMetrics {
		timed = nil
	}
	harness := xmetrics.NewHarness(metricsAPI.eventReceivedCount, metricsAPI.eventErrorCount, timed, clock)
	return eventrules.Metrics(harness, nil)
}

// callMetrics logs metrics for every outgoing Mesos call
func callMetrics(metricsAPI *metricsAPI, clock func() time.Time, timingMetrics bool) callrules.Rule {
	timed := metricsAPI.callLatency
	if !timingMetrics {
		timed = nil
	}
	harness := xmetrics.NewHarness(metricsAPI.callCount, metricsAPI.callErrorCount, timed, clock)
	return callrules.Metrics(harness, nil)
}

// logCalls logs a specific message string when a particular call-type is observed
func logCalls(messages map[scheduler.Call_Type]string) callrules.Rule {
	return func(ctx context.Context, c *scheduler.Call, r mesos.Response, err error, ch callrules.Chain) (context.Context, *scheduler.Call, mesos.Response, error) {
		if message, ok := messages[c.GetType()]; ok {
			log.Println(message)
		}
		return ch(ctx, c, r, err)
	}
}
