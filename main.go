package main

import (
	"github.com/gxthrj/ingress-hw/pkg/web"
	"github.com/gxthrj/ingress-hw/pkg/apisix"
	"github.com/gxthrj/ingress-hw/pkg/k8s"
)

func main(){
	// sync with apisix
	apisix.Watch()
	// watch k8s
	k8s.Watch()
	// some tools
	web.Start()
}
