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
	"fmt"
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
	ep, _ := qo.Obj.(*v1.Endpoints)
	if ep.Namespace != "kube-system"{
		for _, s := range ep.Subsets{
			// ips
			ips := make([]string, 0)
			for _, address := range s.Addresses{
				ips = append(ips, address.IP)
			}
			// ports
			for _, port := range s.Ports {
				infos := make([]podInfo, 0)
				for _, address := range s.Addresses {
					ips = append(ips, address.IP)
					infos = append(infos, podInfo{PodIp: address.IP, Port: port.Port})
				}
				se := &conf.SyncEvent{Namespace: ep.Namespace, Name: ep.Name, Port: port.Port, Nodes: ToMap(infos)}
				conf.SyncQueue <- se
			}
		}
	}
}

func ToMap(infos []podInfo) map[string]int64 {
	result := make(map[string]int64)
	for _, p := range infos {
		s := fmt.Sprintf("%s:%d", p.PodIp, p.Port)
		result[s] = 100 //权重默认100
	}
	return result
}

type podInfo struct {
	PodIp string `json:"podIp"`
	Port int32 `json:"port"`
}

func RunSync() {
	for {
		select {
			case se := <- conf.SyncQueue:
				logger.Info(se)
				Sync(se)
		}
	}
}

func Sync(se *conf.SyncEvent){
	ns := se.Namespace
	name := se.Name
	port := se.Port
	if se.Nodes == nil {
		se.Nodes = make(map[string]int64)
	}
	if port != 0 {
		desc := ns + "_" + name + "_" + strconv.Itoa(int(port))
		key := conf.GetUpstreamMap()[desc]
		if key != "" {
			tmp := strings.Split(key, "/")
			upstreamId := tmp[len(tmp) - 1]
			k8sInfoMap := conf.GetUpstreamK8sMap()
			k8sInfo := k8sInfoMap[desc]
			if k8sInfo != nil {
				// from svc endpoint
				nodes := se.Nodes
				if len(nodes) == 0 {
					epInformer := conf.GetEpInformer()
					ep, _ := epInformer.Lister().Endpoints(ns).Get(name)
					if port != 0 && ep != nil {
						ips := make([]string, 0)
						infos := make([]podInfo, 0)
						for _, s := range ep.Subsets{
							for _, address := range s.Addresses{
								ips = append(ips, address.IP)
								infos = append(infos, podInfo{PodIp: address.IP, Port: int32(port)})
							}
						}
						nodes = ToMap(infos)
					}
				}
				if len(nodes) == 0 {
					// 容错
					nodes["127.0.0.1:8080"] = 0
				}
				// check from apisix if need to update
				needToUpdate := false
				response := apisix.FindUpstreamByName(desc)
				if response.Upstream.Key != "" {
					// check nodes
					oldNodes := response.Upstream.UpstreamNodes.Nodes
					if len(oldNodes) == len(nodes) {
						for k, _ := range oldNodes {
							if nodes[k] == 0 {
								needToUpdate = true
							}
						}
					} else {
						needToUpdate = true
					}
				}
				// update upstream
				if needToUpdate {
					if resp, err := apisix.UpdateUpstream(upstreamId, desc, nodes); err != nil {
						logger.Error(err.Error())
					}else {
						// 更新 modifyIndex
						upstream := resp.Node.Value
						logger.Info(resp)
						logger.Info(upstream)
						conf.GetUpstreamIndexMap()[upstream.Desc] = resp.Node.ModifiedIndex
						//var upstream conf.Upstream
						//if err := json.Unmarshal([]byte(value), &upstream); err != nil {
						//	logger.Error(err)
						//} else {
						//	logger.Info(upstream)
						//	conf.GetUpstreamIndexMap()[upstream.Desc] = resp.Node.ModifiedIndex
						//}
					}
				}
				// from label
				//if pods, err := ListPods(k8sInfo.Label); err == nil {
				//	nodes := make(map[string]int64)
				//	for _, pod := range pods {
				//		for _, condition := range pod.Status.Conditions {
				//			if condition.Type == v1.PodReady && condition.Status == v1.ConditionTrue && pod.Status.Phase == v1.PodRunning { // pod Ready
				//				ip := pod.Status.PodIP
				//				nodes[ip + ":" + strconv.Itoa(int(port))] = 100
				//			}
				//		}
				//	}
				//	if len(nodes) == 0 {
				//		// 容错
				//		nodes["127.0.0.1:8080"] = 0
				//	}
				//	// update upstream
				//	if resp, err := apisix.UpdateUpstream(upstreamId, desc, nodes); err != nil {
				//
				//	}else {
				//		// 更新 modifyIndex
				//		value := resp.Node.Value
				//		var upstream conf.Upstream
				//		if err := json.Unmarshal([]byte(value), &upstream); err != nil {
				//			logger.Error(err)
				//		} else {
				//			conf.GetUpstreamIndexMap()[upstream.Desc] = resp.Node.ModifiedIndex
				//		}
				//	}
				//}
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
	epInformer := conf.GetEpInformer()
	c := &controller{
		queue: make(chan interface{}, 100),
	}
	epInformer.Informer().AddEventHandler(&QueueEventHandler{c:c})
	// podInformer
	go epInformer.Informer().Run(stopCh)
	go c.run()
	go RunSync()
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
