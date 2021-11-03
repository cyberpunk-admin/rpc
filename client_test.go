package geerpc

import (
	"context"
	"net"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

type Bar int

func (b *Bar) TimeOut(args Args, reply *int) error {
	time.Sleep(time.Second * 2)
	return nil
}

func startServer(addr chan string) {
	var b Bar
	_ = Register(&b)

	l, _ := net.Listen("tcp", ":0")
	addr <- l.Addr().String()
	Accept(l)
}

func TestClient_dialTimeOut(t *testing.T) {
	t.Parallel()
	l, _ := net.Listen("tcp", ":0")

	f := func(conn net.Conn, opt *Option) (Client *Client, err error) {
		_ = conn.Close()
		time.Sleep(time.Second * 2)
		return
	}
	t.Run("timeout", func(t *testing.T) {
		_, err := dialTimeOut(f, "tcp", l.Addr().String(), &Option{ConnectTimeOut: time.Second})
		_assert(err != nil && strings.Contains(err.Error(), "connect timeout"), "expect a timeout error")
	})
	t.Run("0", func(t *testing.T) {
		_, err := dialTimeOut(f, "tcp", l.Addr().String(), &Option{ConnectTimeOut: 0})
		_assert(err == nil, "0 mean no limit")
	})
}

func TestClient_Call(t *testing.T) {
	t.Parallel()
	addrCh := make(chan string)
	go startServer(addrCh)
	addr := <-addrCh
	time.Sleep(time.Second)
	t.Run("client timeout", func(t *testing.T) {
		client, _ := Dial("tcp", addr)
		ctx, _ := context.WithTimeout(context.Background(), time.Second)
		var reply int
		err := client.Call(ctx, "Bar.TimeOut", 1, &reply)
		_assert(err != nil && strings.Contains(err.Error(), ctx.Err().Error()), "expect a timeout error1")
	})
	t.Run("server handle timeout", func(t *testing.T) {
		client, _ := Dial("tcp", addr, &Option{
			HandleTimeOut: time.Second,
		})
		var reply int
		err := client.Call(context.Background(), "Bar.TimeOut", 1, &reply)
		_assert(err != nil && strings.Contains(err.Error(), "handle timeOut"), "expect a time out error2")
	})
}

func TestXDial(t *testing.T) {
	if runtime.GOOS == "linux" {
		ch := make(chan struct{})
		addr := "/tmp/geerpc.sock"
		go func() {
			_ = os.Remove(addr)
			l, err := net.Listen("unix", addr)
			if err != nil {
				t.Fatal("failed to listen unix socket")
			}
			ch <- struct{}{}
			Accept(l)
		}()
		<-ch
		_, err := XDial("unix@" + addr)
		_assert(err==nil, "failed to connect unix socket")
	}
}
