package utils

type addr struct {
	network string // name of the network (for example, "tcp", "udp")
	str     string // string form of address (for example, "192.0.2.1:25", "[2001:db8::1]:80")
}

func (a addr) Network() string {
	return a.network
}

func (a addr) String() string {
	return a.str
}

//func TestGetGrpcClientIP(t *testing.T) {
//	network := "tcp"
//	str := "127.0.0.1"
//	addr := addr{
//		network: network,
//		str:     str,
//	}
//	p := peer.Peer{
//		Addr:     addr,
//		AuthInfo: nil,
//	}
//	ctx := peer.NewContext(context.Background(), &p)
//
//	Convey("TestGetGrpcClientIP", t, func() {
//		pr, ok := peer.FromContext(ctx)
//		So(ok, ShouldEqual, true)
//		addSlice := strings.Split(pr.Addr.String(), ":")
//		So(addSlice[0], ShouldEqual, network)
//	})
//}
