package conf

import "fmt"

type Message struct {
	Code string
	Msg  string
}

var (
	//AA 01 表示kong-grace本身的错误
	//BB 00 表示系统信息
	SystemError = Message{"010001", "system errno"}

	//BB 01表示负载均衡错误
	LBThresholdError    = Message{"010101", "服务%s副本%s触发阈值%d%%告警"}
	LBWeightError = Message{"010102", "%s设置%s权重%d失败"}
	LBWeightDegradedError = Message{"010103", "%s均衡策略降级为RR轮询"}
	LBHealthCheckError = Message{"010104", "%s副本%s健康检查失败,错误:%s"}
	LBHealthCheckTransError = Message{"010105", "%s副本%s健康检查结果转换失败,错误:%s"}
)

func (m Message) ToString(params ...interface{}) string{
	params = append(params, m.Code)
	return fmt.Sprintf(m.Msg + " error_code:%s", params...)
}