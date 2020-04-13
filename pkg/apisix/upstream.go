package apisix

import (
	"github.com/gxthrj/seven/conf"
	"fmt"
	"encoding/json"
	"github.com/gxthrj/ingress-hw/pkg/utils"
	"go.etcd.io/etcd/client"
)

type UpstreamRequest struct {
	LBType string           `json:"type"`
	HashOn *string          `json:"hash_on,omitempty"`
	Key    *string          `json:"key,omitempty"`
	Nodes  map[string]int64 `json:"nodes"`
	Desc   string           `json:"desc"`
}

func UpdateUpstream(upstreamId string, name string, nodes map[string]int64) (*client.Response, error){
	url := fmt.Sprintf("%s/upstreams/%s", conf.BaseUrl, upstreamId)
	ur := UpstreamRequest{
		LBType: "roundrobin",
		Desc: name,
		Nodes: nodes,
	}
	if b, err := json.Marshal(ur); err != nil {
		return nil, err
	} else {
		if resp, err := utils.Patch(url, b); err != nil {
			return nil, err
		} else {
			var response client.Response
			if err := json.Unmarshal(resp, &response); err != nil {
				return nil, err
			}else {
				return &response, nil
			}
		}
	}
}
