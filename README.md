# acslogspout

acslogspout是基于logspout（https://github.com/gliderlabs/logspout） 扩展的，适用于阿里云容器服务的日志收集功能。

## 使用方法
以compose模板方式使用:
需要先build出镜像
```
version: 2
services:
    logspout:
        image: {your image}
        labels:
            aliyun.global: true
        restart: always
        volumes:
            - /acs/log/:/acs/log/
            - /var/run/docker.sock:/tmp/docker.sock
        environment:
            - ROUTE_URIS=file:///acs/log
        net: host
```
