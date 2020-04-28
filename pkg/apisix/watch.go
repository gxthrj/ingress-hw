package apisix

import (
	"fmt"
	"go.etcd.io/etcd/client"
	"context"
	"github.com/gxthrj/ingress-hw/conf"
	"github.com/gxthrj/ingress-hw/log"
	"encoding/json"
	"strconv"
)

const ApisixUpstreams = "/apisix/upstreams"
var EUpstreams = &client.Response{}
var logger = log.GetLogger()

// EtcdWatcher
type EtcdWatcher struct {
	client     client.Client
	etcdKey 	string
	ctx        context.Context
	cancels    []context.CancelFunc
}

// NewEtcdWatcher create new `EtcdWatcher`
func NewEtcdWatcher() *EtcdWatcher {
	cfg := client.Config{
		Endpoints: conf.EtcdConfig.Addresses,
		Transport: client.DefaultTransport,
	}
	if c, err := client.New(cfg); err != nil {
		panic(fmt.Sprintf("failed to initialize etcd watcher. %s", err.Error()))
	} else {
		return &EtcdWatcher{
			client:     c,
			etcdKey: 	ApisixUpstreams,
			ctx:        context.Background(),
			cancels:    make([]context.CancelFunc, 0),
		}
	}
}

// List etcd upstreams
func (e *EtcdWatcher) ListUpstreams(){
	kapi := client.NewKeysAPI(e.client)
	// balancer pod level
	if resp, err := kapi.Get(context.Background(), ApisixUpstreams, nil); err != nil {
		logger.Error(err.Error())
	} else {
		EUpstreams = resp
		// trans to map
		trans(EUpstreams)
		// sync upstreams
		upstreamK8sMap := conf.GetUpstreamK8sMap()
		for _, v := range upstreamK8sMap {
			//k8s.Sync(v.Namespace, v.Name, int32(v.Port))
			se := &conf.SyncEvent{Namespace: v.Namespace, Name: v.ServiceName, Port: int32(v.Port)}
			conf.SyncQueue <- se
		}
	}
}

// StopWatch stops all etcd key watching
func (e *EtcdWatcher) StopWatch() {
	for _, cancel := range e.cancels {
		cancel()
	}
}

func trans(eus *client.Response){
	if len(eus.Node.Nodes) > 0 {
		for _, n := range eus.Node.Nodes {
			TransOne(n)
		}
	}
}

func TransOne(node *client.Node) conf.Upstream{
	var upstream conf.Upstream
	if err := json.Unmarshal([]byte(node.Value), &upstream); err != nil {
		logger.Error(err.Error())
	} else {
		ns := upstream.K8sDeployInfo.Namespace
		name := upstream.K8sDeployInfo.ServiceName
		port := upstream.K8sDeployInfo.Port
		desc := ns + conf.Separator + name + conf.Separator + strconv.Itoa(int(port))
		// map[upstreamName]= upstream key
		conf.GetUpstreamMap()[desc] = node.Key
		// map[upstreamName] = modifiedIndex
		conf.GetUpstreamIndexMap()[desc] = node.ModifiedIndex
		// map[upstreamName] = k8s deployment info
		conf.GetUpstreamK8sMap()[desc] = &upstream.K8sDeployInfo
	}
	return upstream
}

func Watch(){
	etcdWatcher := NewEtcdWatcher()
	etcdWatcher.ListUpstreams()
	// watch etcd
	watchCtx, cancel := context.WithCancel(etcdWatcher.ctx)
	etcdWatcher.cancels = append(etcdWatcher.cancels, cancel)
	kapi := client.NewKeysAPI(etcdWatcher.client)
	go watchEtcd(watchCtx, kapi, etcdWatcher.etcdKey)
}

func watchEtcd(ctx context.Context, kapi client.KeysAPI, key string) {
	watcher := kapi.Watcher(key, &client.WatcherOptions{Recursive: true})
	for {
		select {
		case <-ctx.Done():
			logger.Info("etcd watch stopped")
			return
		default:
			if v, err := watcher.Next(context.Background()); err != nil {
				continue
			} else {
				n := v.Node
				value := n.Value
				var upstream conf.Upstream
				if err := json.Unmarshal([]byte(value), &upstream); err != nil {
					logger.Error(err)
				} else {
					//index := conf.GetUpstreamIndexMap()[upstream.Desc]
					//logger.Infof("%d:%d", index, n.ModifiedIndex)
					//if index >= n.ModifiedIndex {
						// do nothing
						//logger.Infof("%s index out of date", upstream.Desc)
					//} else {
						ns := upstream.K8sDeployInfo.Namespace
						name := upstream.K8sDeployInfo.ServiceName
						port := upstream.K8sDeployInfo.Port
						desc := ns + conf.Separator + name + conf.Separator + strconv.Itoa(int(port))
						logger.Infof("%s 开始同步", desc)
						TransOne(n)
						// sync upstream
						logger.Info(conf.GetUpstreamK8sMap())
						k := conf.GetUpstreamK8sMap()[desc]
						if k != nil && k.Port != 0{
							//k8s.Sync(k.Namespace, k.Name, int32(k.Port))
							se := &conf.SyncEvent{Namespace: k.Namespace, Name: k.ServiceName, Port: int32(k.Port)}
							conf.SyncQueue <- se
						}
					//}
				}
				// 遍历 nodes，查看是否已经存在，不存在的需要添加
				//for _, n := range v.Node.Nodes{
				//	value := n.Value
				//	var upstream conf.Upstream
				//	if err := json.Unmarshal([]byte(value), &upstream); err != nil {
				//		log.Println(err)
				//	} else {
				//		index := conf.GetUpstreamIndexMap()[upstream.Desc]
				//		if index >= n.ModifiedIndex {
				//			// do nothing
				//		} else {
				//			TransOne(n)
				//			// sync upstream
				//			k := conf.GetUpstreamK8sMap()[upstream.Desc]
				//			if k != nil {
				//				//k8s.Sync(k.Namespace, k.Name, int32(k.Port))
				//				se := &conf.SyncEvent{Namespace: k.Namespace, Name: k.Name, Port: int32(k.Port)}
				//				conf.SyncQueue <- se
				//			}
				//		}
				//	}
				//}
			}
		}
	}
}
