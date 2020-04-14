package apisix

import (
	"fmt"
	"encoding/json"
	"github.com/gxthrj/ingress-hw/pkg/utils"
	"github.com/gxthrj/ingress-hw/conf"
	"strings"
)

type UpstreamRequest struct {
	LBType string           `json:"type"`
	HashOn *string          `json:"hash_on,omitempty"`
	Key    *string          `json:"key,omitempty"`
	Nodes  map[string]int64 `json:"nodes"`
	Desc   string           `json:"desc"`
}

type UpstreamUpdateResponse struct {
	Action string `json:"action"`
	Node *Node `json:"node"`
}

type Node struct {
	Value Value `json:"value"`
	ModifiedIndex uint64 `json:"modifiedIndex"`
}

type Value struct {
	Desc string `json:"desc"`
}


func UpdateUpstream(upstreamId string, name string, nodes map[string]int64) (*UpstreamUpdateResponse, error){
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
			var response UpstreamUpdateResponse
			if err := json.Unmarshal(resp, &response); err != nil {
				logger.Error(err.Error())
				return nil, err
			}else {
				return &response, nil
			}
		}
	}
}

type UpstreamResponse struct {
	Action string `json:"action"`
	Upstream Upstream `json:"node"`
}

type Upstream struct {
	Key string `json:"key"` // upstream key
	UpstreamNodes UpstreamNodes `json:"value"`
}

type UpstreamNodes struct {
	Nodes map[string]int64 `json:"nodes"`
	Desc string `json:"desc"` // upstream name  = k8s svc
	LBType string `json:"type"` // 负载均衡类型
}

func FindUpstreamByName(name string) UpstreamResponse{
	// 1.根据upstreamId查询
	key := conf.GetUpstreamMap()[name]
	if key != "" {
		urlWithId := fmt.Sprintf("%s%s", conf.BaseUrl, ReplacePrefix(key))
		logger.Info(fmt.Sprintf("=================== %s", urlWithId))
		uRet, _ := utils.Get(urlWithId)
		var upstream UpstreamResponse
		if err := json.Unmarshal(uRet, &upstream); err != nil {
			logger.Error(err.Error())
		} else {
			return upstream
		}
	}
	return UpstreamResponse{}
}

func ReplacePrefix(key string) string {
	return strings.Replace(key, "/apisix", "", 1)
}