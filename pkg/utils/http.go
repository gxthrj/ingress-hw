package utils

import (
	"gopkg.in/resty.v1"
	"time"
	"github.com/gxthrj/ingress-hw/conf"
	"net/http"
	"fmt"
	"github.com/gxthrj/ingress-hw/log"
)

const timeout = 3000

var logger = log.GetLogger()

func Get(url string) ([]byte, error){
	r := resty.New().
		SetTimeout(time.Duration(timeout)*time.Millisecond).
		R().
		SetHeader("content-type", "application/json").
		SetHeader("X-API-KEY", conf.ApiKey)
	resp, err := r.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("status: %d, body: %s", resp.StatusCode(), resp.Body())
	}
	return resp.Body(), nil
}

func Post(url string, bytes []byte) ([]byte, error){
	r := resty.New().
		SetTimeout(time.Duration(timeout)*time.Millisecond).
		R().
		SetHeader("content-type", "application/json").
		SetHeader("X-API-KEY", conf.ApiKey)
	r.SetBody(bytes)
	resp, err := r.Post(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("status: %d, body: %s", resp.StatusCode(), resp.Body())
	}
	return resp.Body(), nil
}

func Patch(url string, bytes []byte) ([]byte, error){
	r := resty.New().
		SetTimeout(time.Duration(timeout)*time.Millisecond).
		R().
		SetHeader("content-type", "application/json").
		SetHeader("X-API-KEY", conf.ApiKey)
	r.SetBody(bytes)
	resp, err := r.Patch(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("status: %d, body: %s", resp.StatusCode(), resp.Body())
	}
	return resp.Body(), nil
}

func Delete(url string) ([]byte, error) {
	r := resty.New().
		SetTimeout(time.Duration(timeout) * time.Millisecond).
		R().
		SetHeader("content-type", "application/json").
		SetHeader("X-API-KEY", conf.ApiKey)
	resp, err := r.Delete(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("status: %d, body: %s", resp.StatusCode(), resp.Body())
	}
	return resp.Body(), nil
}