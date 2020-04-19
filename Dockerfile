FROM golang:1.13.8
MAINTAINER gxthrj@163.com

WORKDIR /go/src/github.com/gxthrj/ingress-hw
COPY . .
RUN mkdir /root/ingress-controller \
    && go env -w GOPROXY=https://goproxy.io,direct \
    && export GOPROXY=https://goproxy.io \
    && go build -o /root/ingress-controller/ingress-controller \
    && mv /go/src/github.com/gxthrj/ingress-hw/build.sh /root/ingress-controller/ \
    && mv /go/src/github.com/gxthrj/ingress-hw/conf.json /root/ingress-controller/ \
    && rm -rf /go/src/github.com/gxthrj/ingress-hw \
    && rm -rf /etc/localtime \
    && ln -s  /usr/share/zoneinfo/Hongkong /etc/localtime \
    && dpkg-reconfigure -f noninteractive tzdata

WORKDIR /root/ingress-controller

EXPOSE 8080
RUN chmod +x ./build.sh
CMD ["/root/ingress-controller/build.sh"]
