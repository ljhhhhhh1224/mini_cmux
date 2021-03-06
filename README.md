## mini_cmux
***
mini_cmux支持在启动后只监听一个端口的情况下同时接受http访问和grpc访问

`项目结构`
```
├── client
│   └── client.go                   # 客户端访问入口
├── ginServer                       # http服务
│   ├── ginserver.go
│   └── ginserver_test.go
├── go.mod
├── go.sum
├── grpcServer                      # gprc服务
│   ├── gprcserver_test.go
│   └── grpcserver.go
├── logging                         # 日志组件
│   ├── file.go
│   └── log.go
├── mini_cmux                       # mini_cmux 核心组件
│   ├── buffer.go
│   ├── matchers.go
│   └── mini_cmux.go
├── pb                              # protocol
│   ├── build.sh
│   ├── hello_grpc_grpc.pb.go
│   ├── hello_grpc.pb.go
│   └── hello_grpc.proto
├── resource                        # 资源文件
│   ├── deployment.yaml
│   ├── docker-compose.yml
│   ├── Dockerfile
│   ├── Makefile
│   └── service.yaml
├── server.go                       # 服务端启动入口
├── syscallOperate                  # 监听系统关闭信号组件
│   ├── syscallOperate.go
│   └── syscallOperate_test.go
├── test                            # mini_cmux单元测试
│   └── mini_cmux_test.go
│── utils                           # 工具方法
│    ├── utils.go
│    └── utils_test.go
├── conf                            # toml配置文件
     └── config.toml
```


## 使用方式

```golang
	l, err := net.Listen("tcp", ":23456")
	if err != nil {
		log.Fatal(err)
	}
   
	//创建mini_cmux实例
	m := mini_cmux.New(l)

	//匹配HTTP与GRPC
	grpcL := m.Match(mini_cmux.HTTP2HeaderField("content-type", "application/grpc"))
	httpL := m.Match(mini_cmux.HTTP1HeaderField("content-type", "application/json"))

	//GRPC服务
	grpcS := grpc.NewServer()
	hello_grpc.RegisterHelloGRPCServer(grpcS, &grpcServer.Server{})

	//HTTP服务
	router := ginServer.SetupRouter()
        httpS := &http.Server{
	    Handler: &helloHTTP1Handler{},
        }
    
        go grpcS.Serve(grpcL)
	go httpS.Serve(httpL)

	m.Serve()
```

## 部署方式
首次部署需要对服务端与客户端的参数(ip、端口号、协议等信息)进行配置,配置文件为`conf/config.toml`,配置完成后即可开始部署项目
```toml
[client]
IP     = "127.0.0.1"
Port   = ":23456"

[server]
Port   = ":23456"
Network = "tcp"
```

***
使用docker部署项目  
docker安装步骤见官网 https://docs.docker.com/get-started/  
docker-compose 安装步骤见官网 https://docs.docker.com/compose/install/

```
$ cd $GOPATH/src
$ git clone https://github.com/ljhhhhhh1224/mini_cmux.git
$ cd mini_cmux/resource
$ docker-compose up
```

部署成功后即可使用客户端对服务进行访问

***

k8s部署(采用minikube进行部署)  
minikube安装步骤见官网 https://minikube.sigs.k8s.io/docs/start/  
kubectl安装步骤 https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/

安装成功后在终端输入`minikube start`启动minikube

在上一步`docker-compose up`之后,会生成对应的docker镜像和容器，我们使用生成的docker镜像进行k8s部署

`deployment.yaml`
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: minicmux
  namespace: default
  labels:
    app: minicmux
    version: 0.0.1
spec:
  replicas: 3
  selector:
    matchLabels:
      app: minicmux
  template:
    metadata:
      labels:
        app: minicmux
    spec:
      containers:
        - name: minicmux
          image: $(docker image name) #对应的docker镜像名称
          ports:
            - containerPort: 23456
```

`service.yaml`
```yaml
apiVersion: v1
kind: Service
metadata:
  name: minicmux
spec:
  selector:
    app: minicmux
  type: NodePort
  ports:
    - port: 23456
      targetPort: 23456
      nodePort: 31958
```

以上的`deployment.yaml`和`service.yaml`,我们通过`kubectl`执行  
```shell
$ kubectl create -f deployment.yaml
$ kubectl apply -f service.yaml
```

分别查看deployment、service、pod的详情
```shell
$ kubectl get pods
$ kubectl get deploy
$ kubectl get service
```

获得服务的`url`
```shell
$ minikube service --url minicmux
```

后续可以在本机上通过客户端发送请求到该`url`访问到服务
***
## 实现原理
`mini_cmux`通过 `matchers(匹配器)`对`HTTP header fields`中的键值对将请求按照规则分配到不同的服务当中  
  
`mini_cmux`的核心为 `mini_cmux(多路复用器)` 及 `matchers(匹配器)`的实现

框架中最核心的为继承了`CMux`接口的`cMux`结构体
```go
type cMux struct {
	root   net.Listener
	bufLen int                // 匹配器中缓存连接的队列长度
	sls    []matchersListener // 注册的匹配器列表
	donec  chan struct{}      // 通知多路复用器关闭的channel
	mu     sync.Mutex
}
```

此多路复用器的实现方式是通过接受一个连接，然后通过遍历多路复用器中的匹配器列表，找到对应的服务,然后将请求交给对应的服务进行处理

首先我们从`matchers`开始讲起,mini_cmux通过区分`HTTP header fields`中的键值对,mini_cmux提供了`HTTP1`、`GRPC`和`Any`三种匹配规则
```go
// HTTP1HeaderField 返回一个匹配 HTTP 1 连接的第一个请求的头字段的匹配器。
func HTTP1HeaderField(name, value string) MatchWriter {
	return func(w io.Writer, r io.Reader) bool {
		req, err := http.ReadRequest(bufio.NewReader(r))
		if err != nil {
			return false
		}
		return req.Header.Get(name) == value
	}
}
```

调用`matchers`中的匹配规则方法会返回一个`MatchWriter`(func(io.Writer, io.Reader) bool),对于此`matchWriter`,cMux提供了`Match`方法将MatchWriter注册到cMux的匹配器列表中

```go
// Match 对传入的 MatchWriter 进行包装成 muxListener并作为函数的返回值，muxListener实现了 net.Listener 接口
// muxListener用一个 conn channel 和 done channel 来让与匹配器匹配成功的服务端进行连接的获取、处理和关闭等操作
// 此方法返回的 listener 可被用于各服务的监听
func (m *cMux) Match(matchers MatchWriter) net.Listener {
	ml := muxListener{
		Listener: m.root,
		connc:    make(chan net.Conn, m.bufLen),
		donec:    make(chan struct{}),
	}
	//将该muxListener 与 matcherWriter 封装成 matcherListener 添加到CMux匹配器列表中
	m.sls = append(m.sls, matchersListener{ss: matchers, l: ml})
	return ml
}
```
`muxListener`结构
```go
type muxListener struct {
	net.Listener
	connc chan net.Conn
	donec chan struct{}
}

// muxListener还重写了Accept()方法让各服务接收conn
func (l muxListener) Accept() (net.Conn, error) {
	select {
	case c, ok := <-l.connc:
		if !ok {
			return nil, ConnError
		}
		return c, nil
	case <-l.donec:
		return nil, ServerCloseErr
	}
}
```

当我们完成上述将`matchersListener`匹配器注册到cMux匹配器列表中的操作后，我们的cMux就可以开始正式工作了



以下是具体实现
```go
func (m *cMux) Serve() error {
	var wg sync.WaitGroup

	defer func() {
		m.closeDoneChans()
		wg.Wait()

		for _, sl := range m.sls {
			close(sl.l.connc)
			// 关闭各匹配器对应的连接队列
			for c := range sl.l.connc {
				_ = c.Close()
			}
		}
	}()

	for {
		c, err := m.root.Accept()
		if err != nil {
			return err
		}

		wg.Add(1)
		go m.serve(c, m.donec, &wg)
	}
}

func (m *cMux) serve(c net.Conn, donec <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	// 将 net.Conn 包装为 MuxConn
	muc := newMuxConn(c)

	// 遍历已注册的匹配器列表
	for _, sl := range m.sls {
		matched := sl.ss(muc.Conn, muc.startSniffing())
		if matched {
			muc.doneSniffing()
			select {
			// 将匹配成功的连接放入匹配器的缓存队列中，结束
			case sl.l.connc <- muc: 
				// 如果多路复用器标识为终止，则关闭连接，结束
			case <-donec:
				_ = c.Close()
			}
			return
		}
	}
	c.Close()
}
```





