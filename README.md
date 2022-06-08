## mini_cmux
***
mini_cmux支持在启动后只监听一个端口的情况下同时接受http访问和grpc访问

## 使用方式

```
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




