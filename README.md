## mini_cmux
***
mini_cmux支持在启动后只监听一个端口的情况下同时接受http访问和grpc访问

`项目结构`
```
├── buffer.go
├── client
│   └── client.go
├── deployment.yaml
├── docker-compose.yml
├── Dockerfile
├── example
│   └── example.go
├── ginServer
│   ├── ginserver.go
│   └── ginserver_test.go
├── go.mod
├── go.sum
├── grpcServer
│   ├── gprcserver_test.go
│   └── grpcserver.go
├── logging
│   ├── file.go
│   └── log.go
├── Makefile
├── matchers.go
├── mini_cmux.go
├── pb
│   ├── build.sh
│   ├── hello_grpc_grpc.pb.go
│   ├── hello_grpc.pb.go
│   └── hello_grpc.proto
├── README.md
├── service.yaml
├── syscallOperate
│   ├── syscallOperate.go
│   └── syscallOperate_test.go
├── test
│   └── mini_cmux_test.go
└── utils
    ├── utils.go
    └── utils_test.go
```


## 使用方式

```golang
	l, err := net.Listen("tcp", ":23456")
	if err != nil {
		log.Fatal(err)
	}
    
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
***
使用docker部署项目  
docker安装步骤见官网 https://docs.docker.com/get-started/  
docker-compose 安装步骤见官网 https://docs.docker.com/compose/install/

```
cd $GOPATH/src
git clone https://github.com/ljhhhhhh1224/mini_cmux.git
cd mini_cmux
docker-compose up
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

通过以上的`deployment.yaml`和`service.yaml`,我们通过`kubectl`执行  
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

## 实现原理