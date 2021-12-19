package main

import (
	"context"
	"errors"
	"geerpc"
	"geerpc/xclient"
	"log"
	"net"
	"sync"
	"time"
)

type Foo int

type Args struct {
	Num1, Num2 int
}

func (foo Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func (foo Foo) Sleep(args Args, reply *int) error {
	time.Sleep(time.Second * time.Duration(args.Num1))
	*reply = args.Num1 + args.Num2
	return nil
}

func startServer(addrCH chan string) {
	var foo Foo
	l, _ := net.Listen("tcp", ":0")
	log.Println("start server on address: ", l.Addr())
	server := geerpc.NewServer()
	_ = server.Register(&foo)
	addrCH <- l.Addr().String()
	server.Accept(l)
}

func foo(xc *xclient.XClient, ctx context.Context, typ, serviceMethod string, args *Args) error{
	var reply int
	var err error
	switch typ {
	case "call":
		err = xc.Call(ctx, serviceMethod, args, &reply)
	case "broadcast":
		err = xc.Broadcast(ctx, serviceMethod, args, &reply)
	default:
		err = errors.New("foo method: unsupported typ")
	}
	if err != nil {
		log.Printf("%s %s error: %v", typ, serviceMethod, err)
	}else{
		log.Printf("%s %s success: %d + %d = %d", typ, serviceMethod, args.Num1, args.Num2, reply)
	}
	return err
}

func call(addr1, addr2 string) {
	d := xclient.NewMultiServersDiscovery([]string{"tcp@" + addr1, "tcp@" + addr2})
	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	defer func() {_ = xc.Close()}()
	// send request and receive response
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = foo(xc, context.Background(), "call", "Foo.Sum", &Args{i, i*i})
		}(i)
	}
	wg.Wait()
}

func broadcast(addr1, addr2 string) {
	d := xclient.NewMultiServersDiscovery([]string{"tcp@" + addr1, "tcp@" + addr2})
	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	defer func() {_ = xc.Close()}()
	// send request and receive response
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = foo(xc, context.Background(), "broadcast", "Foo.Sum", &Args{i, i*i})
			// expect 2-5 timeout
			ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
			_ = foo(xc, ctx, "broadcast", "Foo.Sleep", &Args{i, i*i})
		}(i)
	}
	wg.Wait()
}



func main() {
	log.SetFlags(0)
	ch1 := make(chan string)
	ch2 := make(chan string)

	// start 2 servers
	go startServer(ch1)
	go startServer(ch2)
	addr1 := <-ch1
	addr2 := <-ch2
	time.Sleep(time.Second * 2)
	call(addr1, addr2)
	broadcast(addr1, addr2)
}
