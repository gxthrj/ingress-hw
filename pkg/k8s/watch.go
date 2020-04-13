package k8s

import (
	"github.com/gxthrj/ingress-hw/conf"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"github.com/gxthrj/ingress-hw/log"
	"k8s.io/api/core/v1"
	"strconv"
	"k8s.io/apimachinery/pkg/labels"
	"strings"
	"github.com/gxthrj/ingress-hw/pkg/apisix"
	"encoding/json"
)

const ADD = "ADD"
const UPDATE = "UPDATE"
const DELETE = "DELETE"

var logger = log.GetLogger()
var errorCallbacks = make([]func(error), 0)

type controller struct {
	queue chan interface{}
}

func (c *controller) pop() interface{}{
	e := <- c.queue
	return e
}

func (c *controller) run() {
	for {
		//c.pop()
		ele := c.pop()
		c.process(ele)
	}
}

func (c *controller) process(obj interface{}) {
	qo, _ := obj.(*queueObj)
	pod, _ := qo.Obj.(*v1.Pod)
	ns := pod.Namespace
	deployName := pod.Annotations["app"]
	for _, p := range pod.Spec.Containers[0].Ports {
		if p.ContainerPort != 0 {
			Sync(ns, deployName, p.ContainerPort)
		}

	}
}

func Sync(ns, name string, port int32){
	desc := ns + "_" + name + "_" + strconv.Itoa(int(port))
	key := conf.GetUpstreamMap()[desc]
	if key != ""{
		tmp := strings.Split(key, "/")
		upstreamId := tmp[len(tmp) - 1]
		k8sInfoMap := conf.GetUpstreamK8sMap()
		k8sInfo := k8sInfoMap[desc]
		if k8sInfo != nil {
			if pods, err := ListPods(k8sInfo.Label); err == nil {
				nodes := make(map[string]int64)
				for _, pod := range pods {
					for _, condition := range pod.Status.Conditions {
						if condition.Type == v1.PodReady && condition.Status == v1.ConditionTrue && pod.Status.Phase == v1.PodRunning { // pod Ready
							ip := pod.Status.PodIP
							nodes[ip + ":" + strconv.Itoa(int(port))] = 100
						}
					}
				}
				if len(nodes) == 0 {
					// 容错
					nodes["127.0.0.1:8080"] = 0
				}
				// update upstream
				if resp, err := apisix.UpdateUpstream(upstreamId, desc, nodes); err != nil {

				}else {
					// 更新 modifyIndex
					value := resp.Node.Value
					var upstream conf.Upstream
					if err := json.Unmarshal([]byte(value), &upstream); err != nil {
						logger.Error(err)
					} else {
						conf.GetUpstreamIndexMap()[upstream.Desc] = resp.Node.ModifiedIndex
					}
				}
			}
		}
	}
}

func ListPods(m map[string]string) ([]*v1.Pod, error){
	podInformer := conf.GetPodInformer()
	selector := labels.Set(m).AsSelector()
	ret, err := podInformer.Lister().List(selector)
	for _, pod := range ret {
		logger.Info(pod.Status.PodIP)
	}
	return ret, err
}

type QueueEventHandler struct {
	c *controller
}

type queueObj struct {
	OpeType string `json:"ope_type"`
	Obj interface{} `json:"obj"`
}

func (h *QueueEventHandler) OnAdd(obj interface{}) {
	h.c.queue <- &queueObj{ADD, obj}
}

func (h *QueueEventHandler) OnDelete(obj interface{}) {
	h.c.queue <- &queueObj{DELETE, obj}
}

func (h *QueueEventHandler) OnUpdate(old, update interface{}) {
	h.c.queue <- &queueObj{ UPDATE, update}
}

func Watch(){
	// 增加错误监控
	errorCallbacks = append(errorCallbacks, ErrorCallback)
	setupErrorHandlers()
	// informer
	stopCh := make(chan struct{})
	podInformer := conf.GetPodInformer()
	c := &controller{
		queue: make(chan interface{}, 100),
	}
	podInformer.Informer().AddEventHandler(&QueueEventHandler{c:c})
	// podInformer
	go podInformer.Informer().Run(stopCh)
	go c.run()
	// svcInformer
	svcInformer := conf.GetSvcInformer()
	go svcInformer.Informer().Run(stopCh)
	// nsInformer
	nsInformer := conf.GetNsInformer()
	go nsInformer.Informer().Run(stopCh)

	// scheduler
	//go Scheduler()
}

func setupErrorHandlers() {
	nErrFunc := len(utilruntime.ErrorHandlers)
	customErrorHandler := make([]func(error), nErrFunc+1)
	copy(customErrorHandler, utilruntime.ErrorHandlers)
	customErrorHandler[nErrFunc] = func(err error) {
		for _, callback := range errorCallbacks {
			callback(err)
		}
	}
	utilruntime.ErrorHandlers = customErrorHandler
}

func ErrorCallback(err error){
	logger.Error("ALARM FROM K8S", err.Error())
}
