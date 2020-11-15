# Docker镜像的使用

在`go-ethereum-release-1.9/`中，有名为`Dockerfile`的文件，这是用来创建Docker image的文件。

```dockerfile
# Build Geth in a stock Go builder container
FROM golang:1.13-alpine as builder
# apk使用阿里云源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add --no-cache make gcc musl-dev linux-headers git

ADD . /go-ethereum
# go使用中国代理
RUN go env -w GOPROXY=https://goproxy.cn
RUN cd /go-ethereum && make geth

# Pull Geth into a second stage deploy alpine container
FROM alpine:latest
# apk使用阿里云源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add --no-cache ca-certificates
COPY --from=builder /go-ethereum/build/bin/geth /usr/local/bin/

EXPOSE 8545 8546 8547 30303 30303/udp
ENTRYPOINT ["geth"]
```

## 前提

Docker已经正确安装且运行。

## 使用

在终端下将工作目录调整到`go-ethereum-release-1.9`下。

若要生成镜像，执行`docker build -t xdlianrong/ethereum_zkp:0.0.1 .`

镜像生成之后，首次启动需要初始化创世区块，所以启动容器需要使用如下命令：`docker run -it --rm --name ethereum_zkp -v /path/to/genesis.json:/root/privatechain/genesis.json -v /path/to/privatechain:/root/privatechain --entrypoint /bin/sh xdlianrong/ethereum_zkp:0.0.1`

首次进入容器后初始化创世区块：`geth -datadir /root/privatechain/Data0/ init /root/privatechain/genesis.json`

初始化完成之后，就可以按照正常步骤启动geth，因为上面示例中的数据目录为`/root/privatechain/Data0/`，故启动geth需要声明`--datadir /root/privatechain/Data0/`，下面给出的是我使用的示例：

> geth --identity "666" --rpc --rpcport "8545" --rpccorsdomain "http://localhost:8000" --rpcapi "eth,net,web3,personal,admin,txpool,debug,miner" --datadir /root/privatechain/Data0/ --port "30303" --nodiscover --allow-insecure-unlock --regulatorip 39.99.227.43 console

**注意**：如果要修改port，需要在镜像启动时声明端口映射。geth的启动参数要和官方文档要求一致。

