package conf

import (
	coreinformers "k8s.io/client-go/informers/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/informers"
	"os"
	"path/filepath"
	"io/ioutil"
	"fmt"
	"github.com/tidwall/gjson"
	"runtime"
	"go.etcd.io/etcd/client"
)

var (
	ENV      string
	basePath string

	podInformer coreinformers.PodInformer
	svcInformer coreinformers.ServiceInformer
	nsInformer coreinformers.NamespaceInformer
	EndpointsInformer coreinformers.EndpointsInformer

	// etcd upstreams
	EUpstreams = &client.Response{}
	// map[upstreamName]= upstream key
	upstreamMap = make(map[string]string)
	// map[upstreamName] = modifiedIndex
	upstreamIndexMap = make(map[string]uint64)
	// map[upstreamName] = k8s deployment info
	upstreamK8sMap = make(map[string]*K8sDeployInfo)
	// 同步 事件
	SyncQueue = make(chan *SyncEvent, 1000)
	// apisix admin url
	BaseUrl = "http://172.16.20.90:30116/apisix/admin"


)
const PROD = "prod"
const HBPROD = "hb-prod"
const BETA = "beta"
const DEV = "dev"
const LOCAL = "local"
const confPath = "/root/ingress-hw/conf.json"
const ApisixUpstreams = "/apisix/upstreams"



func setEnvironment() {
	if env := os.Getenv("ENV"); env == "" {
		ENV = LOCAL
	} else {
		ENV = env
	}
	_, basePath, _, _ = runtime.Caller(1)
}

func ConfPath() string {
	if ENV == LOCAL {
		return filepath.Join(filepath.Dir(basePath), "conf.json")
	} else {
		return confPath
	}
}

var EtcdConfig etcdConfig
var K8sAuth k8sAuth
var Syslog syslog

func init() {
	// 获取当前环境
	setEnvironment()
	// 获取配置文件路径
	filePath := ConfPath()
	// 获取配置文件内容
	if configurationContent, err := ioutil.ReadFile(filePath); err != nil {
		panic(fmt.Sprintf("failed to read configuration file: %s", filePath))
	} else {
		configuration := gjson.ParseBytes(configurationContent)
		// etcd conf
		etcdConf := configuration.Get("conf.etcd")
		addresses := make([]string, 0, len(etcdConf.Get("address").Array()))
		for _, address := range etcdConf.Get("address").Array() {
			addresses = append(addresses, address.String())
		}
		EtcdConfig.Addresses = addresses
		// k8sAuth conf
		k8sAuthConf := configuration.Get("conf.k8sAuth")
		K8sAuth.file = k8sAuthConf.Get("file").String()
		// syslog conf
		syslogConf := configuration.Get("conf.syslog")
		Syslog.Host = syslogConf.Get("host").String()
	}
	// 获取etcd中 upstreams
	//etcdWatcher := NewEtcdWatcher()
	//etcdWatcher.ListUpstreams()
	// watch etcd
	//watchCtx, cancel := context.WithCancel(etcdWatcher.ctx)
	//etcdWatcher.cancels = append(etcdWatcher.cancels, cancel)
	//kapi := client.NewKeysAPI(etcdWatcher.client)
	//go watchEtcd(watchCtx, kapi, etcdWatcher.etcdKey)

	// init informer
	InitInformer()
}
type SyncEvent struct {
	Namespace string `json:"namespace"`
	Name string `json:"name"`
	Port int32	`json:"port"`
	Nodes map[string]int64 `json:"nodes"`
}

type etcdConfig struct {
	Addresses []string
}

type k8sAuth struct {
	file string
}

type syslog struct {
	Host string
}


func GetPodInformer() coreinformers.PodInformer{
	return podInformer
}

func GetSvcInformer() coreinformers.ServiceInformer{
	return svcInformer
}

func GetNsInformer() coreinformers.NamespaceInformer{
	return nsInformer
}

func GetEpInformer() coreinformers.EndpointsInformer{
	return EndpointsInformer
}

func GetUpstreamK8sMap() map[string]*K8sDeployInfo {
	return upstreamK8sMap
}

func GetUpstreamMap() map[string]string{
	return upstreamMap
}

func GetUpstreamIndexMap() map[string]uint64{
	return upstreamIndexMap
}

func InitInformer() (coreinformers.PodInformer, coreinformers.ServiceInformer, coreinformers.NamespaceInformer){
	// 生成一个k8s client
	var config *restclient.Config
	var err error
	if ENV == LOCAL {
		clientConfig, err := clientcmd.LoadFromFile(K8sAuth.file)
		ExceptNilErr(err)

		config, err = clientcmd.NewDefaultClientConfig(*clientConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
		ExceptNilErr(err)
	} else {
		config, err = restclient.InClusterConfig()
		ExceptNilErr(err)
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	ExceptNilErr(err)

	// 创建一个informerFactory
	sharedInformerFactory := informers.NewSharedInformerFactory(k8sClient, 0)

	// 创建 informers
	podInformer = sharedInformerFactory.Core().V1().Pods()
	svcInformer = sharedInformerFactory.Core().V1().Services()
	nsInformer = sharedInformerFactory.Core().V1().Namespaces()
	EndpointsInformer = sharedInformerFactory.Core().V1().Endpoints()
	return podInformer, svcInformer, nsInformer
}

func ExceptNilErr(err error)  {
	if err != nil {
		panic(err)
	}
}

// EtcdWatcher
//type EtcdWatcher struct {
//	client     client.Client
//	etcdKey 	string
//	ctx        context.Context
//	cancels    []context.CancelFunc
//}

// NewEtcdWatcher create new `EtcdWatcher`
//func NewEtcdWatcher() *EtcdWatcher {
//	cfg := client.Config{
//		Endpoints: EtcdConfig.Addresses,
//		Transport: client.DefaultTransport,
//	}
//	if c, err := client.New(cfg); err != nil {
//		panic(fmt.Sprintf("failed to initialize etcd watcher. %s", err.Error()))
//	} else {
//		return &EtcdWatcher{
//			client:     c,
//			etcdKey: 	ApisixUpstreams,
//			ctx:        context.Background(),
//			cancels:    make([]context.CancelFunc, 0),
//		}
//	}
//}
//
//// StopWatch stops all etcd key watching
//func (e *EtcdWatcher) StopWatch() {
//	for _, cancel := range e.cancels {
//		cancel()
//	}
//}

// etcd upstream struct

type Upstream struct {
	Nodes map[string]int64 `json:"nodes"`
	Desc string `json:"desc"`
	LBType string `json:"type"`
	K8sDeployInfo K8sDeployInfo `json:"k8s-deployment-info"`
}

type K8sDeployInfo struct {
	Namespace string `json:"namespace"`
	Name string `json:"name"`
	Port int64 `json:"port"`
	Label map[string]string `json:"label"`
	//BackendType string `json:"backendType"`
}


// etcd upstream struct end

// List etcd upstreams
//func (e *EtcdWatcher) ListUpstreams(){
//	kapi := client.NewKeysAPI(e.client)
//	// balancer pod level
//	if resp, err := kapi.Get(context.Background(), ApisixUpstreams, nil); err != nil {
//		log.Println(err.Error())
//	} else {
//		EUpstreams = resp
//		// trans to map
//		trans(EUpstreams)
//	}
//}

//func trans(eus *client.Response){
//	if len(eus.Node.Nodes) > 0 {
//		for _, n := range eus.Node.Nodes {
//			TransOne(n)
//		}
//	}
//}
//
//func TransOne(node *client.Node){
//	var upstream Upstream
//	if err := json.Unmarshal([]byte(node.Value), &upstream); err != nil {
//		log.Println(err.Error())
//	} else {
//		// map[upstreamName]= upstream key
//		upstreamMap[upstream.Desc] = node.Key
//		// map[upstreamName] = modifiedIndex
//		upstreamIndexMap[upstream.Desc] = node.ModifiedIndex
//		// map[upstreamName] = k8s deployment info
//		upstreamK8sMap[upstream.Desc] = &upstream.K8sDeployInfo
//	}
//}



//func watchEtcd(ctx context.Context, kapi client.KeysAPI, key string) {
//	watcher := kapi.Watcher(key, &client.WatcherOptions{Recursive: true})
//	for {
//		select {
//		case <-ctx.Done():
//			log.Println("etcd watch stopped")
//			return
//		default:
//			if v, err := watcher.Next(context.Background()); err != nil {
//				continue
//			} else {
//				// 遍历 nodes，查看是否已经存在，不存在的需要添加
//				for _, n := range v.Node.Nodes{
//					value := n.Value
//					var upstream conf.Upstream
//					if err := json.Unmarshal([]byte(value), &upstream); err != nil {
//						log.Println(err)
//					} else {
//						index := upstreamIndexMap[upstream.Desc]
//						if index >= n.ModifiedIndex {
//							// do nothing
//						} else {
//							TransOne(n)
//							// todo sync upstream
//
//						}
//					}
//				}
//			}
//		}
//	}
//}
