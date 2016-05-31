# acslogspout

acslogspout是基于logspout（https://github.com/gliderlabs/logspout） 扩展的，适用于阿里云容器服务的日志收集功能。

## 使用方法
1. 执行build.sh,会在bin目录下编译出logspout
2. docker build -t registry.aliyuncs.com/yourname/acslogspout:test
3. 以compose模板方式在阿里云容器服务创建应用:

```
version: 2
services:
    logspout:
        image: registry.aliyuncs.com/yourname/acslogspout:test
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
