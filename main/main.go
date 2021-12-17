package main

import (
	"context"
	"geerpc"
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

func startServer(addr chan string) {
	var foo Foo
	l, err := net.Listen("tcp", ":9999")
	if err != nil {
		log.Fatal("network error: ", err)
	}
	if err = geerpc.Register(&foo); err != nil {
		log.Fatal("register error: ", err)
	}

	log.Println("start server on address: ", l.Addr())
	addr <- l.Addr().String()
	_ = http.Serve(l, nil)
}


func call(addrCH chan string) {
	client, _ := geerpc.DialHTTP("tcp", <-addrCH)
	defer func() { _ = client.Close() }()
	time.Sleep(time.Second)

	var wg sync.WaitGroup
	for i:=0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := &Args{
				Num1: i,
				Num2: i*i,
			}
			var reply int
			if err:= client.Call(context.Background(), "Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum err ", err)
			}
			log.Println("%d + %d = %d", args.Num1, args.Num2, reply)
		}(i)
	}
	wg.Wait()
}



func main() {
	log.SetFlags(0)
	addr := make(chan string)
	go call(addr)
	startServer(addr)
}
