package web

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"github.com/gxthrj/ingress-hw/log"
	"io"
	"github.com/gxthrj/ingress-hw/conf"
	"strconv"
)

var logger = log.GetLogger()

func Route() *httprouter.Router{
	router := httprouter.New()
	router.GET("/healthz", Healthz)
	router.GET("/sync/:namespace/:name/:port", sync)
	return router
}

func Healthz(w http.ResponseWriter, req *http.Request, _ httprouter.Params){
	io.WriteString(w, "ok")
}

func sync(w http.ResponseWriter, req *http.Request, p httprouter.Params){
	ns := p.ByName("namespace")
	svcName := p.ByName("name")
	port := p.ByName("port")
	portNum, _ := strconv.Atoi(port)
	se := &conf.SyncEvent{Namespace: ns, Name: svcName, Port: int32(portNum)}
	conf.SyncQueue <- se
}

func Start(){
	router := Route()
	err := http.ListenAndServe(":8080", router)
	if err != nil {
		logger.Fatal("ListenAndServe: ", err)
	}
}
