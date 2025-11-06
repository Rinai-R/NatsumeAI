# NatsumeAI

一个基于 go-zero 的微服务电商，集成了权限认证，CDC，agent，分布式事务等技术。

待完善...

## 文档

(apifox 接口文档)[https://u6pteyxjh0.apifox.cn/]

## 前置环境

一台电脑

Go 1.25.2

docker 以及 compose **插件**

## 快速开始

1. 构建项目：

```bash
go mod tidy
chmod +x ./build.sh
./build.sh
```

2. 启动依赖
```bash
make dependency-prep
make dependency
make devops
```
3. 引入 sql 文件

4. 启动程序
```bash
make app
```