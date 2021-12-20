package main

import (
	"context"
	"errors"
	"geerpc"
	"geerpc/register"
	"geerpc/xclient"
	"log"
	"net"
	"net/http"
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

func startRegister(wg *sync.WaitGroup) {
	l, _ := net.Listen("tcp", ":9999")
	register.HandleHTTP()
	wg.Done()
	_ = http.Serve(l, nil)
}

func startServer(registerAddr string, wg *sync.WaitGroup) {
	var foo Foo
	l, _ := net.Listen("tcp", ":0")
	log.Println("start server on address: ", l.Addr())
	server := geerpc.NewServer()
	_ = server.Register(&foo)
	register.Heartbeat(registerAddr, "tcp@" + l.Addr().String(), 0)
	wg.Done()
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

func call(registerAddr string) {
	d := xclient.NewGeeRegisterDiscovery(registerAddr, 0)
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

func broadcast(registerAddr string) {
	d := xclient.NewGeeRegisterDiscovery(registerAddr, 0)
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
			ctx, _ := context.WithTimeout(context.Background(), time.Second)
			_ = foo(xc, ctx, "broadcast", "Foo.Sleep", &Args{i, i*i})
		}(i)
	}
	wg.Wait()
}



func main() {
	log.SetFlags(0)
	registerAddr := "http://localhost:9999/_geerpc_/registry"

	var wg sync.WaitGroup
	wg.Add(1)
	go startRegister(&wg)
	wg.Wait()

	time.Sleep(time.Second)
	wg.Add(2)
	go startServer(registerAddr, &wg)
	go startServer(registerAddr, &wg)
	wg.Wait()

	time.Sleep(time.Second * 2)
	call(registerAddr)
	broadcast(registerAddr)
}
