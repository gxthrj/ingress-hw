package web

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"github.com/gxthrj/ingress-hw/log"
)

var logger = log.GetLogger()

func Route() *httprouter.Router{
	router := httprouter.New()
	return router
}

func Start(){
	router := Route()
	err := http.ListenAndServe(":8080", router)
	if err != nil {
		logger.Fatal("ListenAndServe: ", err)
	}
}
