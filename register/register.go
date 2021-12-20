package register

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// GeeRegister is a register center, provide following functions:
// * add a server and receive heartbeat to keep alive
// * return all alive servers and delete invalid server
type GeeRegister struct {
	timeout time.Duration			// if timeout is 0 will not time out
	mu      sync.Mutex
	servers map[string]*ServerItem
}

type ServerItem struct {
	Addr	string
	start	time.Time
}

const (
	defaultPath = "/_geerpc_/registry"
	defaultTimeout = time.Minute * 5	// The service is considered unavailable when the time exceeds
)

// New create a GeeRegister instance
func New(timeout time.Duration) *GeeRegister {
	return &GeeRegister{
		timeout: timeout,
		servers: make(map[string]*ServerItem),
	}
}

var DefaultGeeRegister = New(defaultTimeout)

func (r *GeeRegister) putServer(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.servers[addr]
	if s == nil {
		r.servers[addr] = &ServerItem{Addr: addr, start: time.Now()}
	}else{
		s.start = time.Now()
	}
}

func (r *GeeRegister) aliveServer() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var alive []string
	for addr, s := range r.servers {
		if r.timeout == 0 || s.start.Add(r.timeout).After(time.Now()) {
			alive = append(alive, addr)
		}else {
			delete(r.servers, addr)
		}
	}
	sort.Strings(alive)
	return alive
}

// Run at /_geerpc_/register
func (r *GeeRegister) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		// keep it simple, servers is in request header
		w.Header().Set("X-GeeRPC-Servers", strings.Join(r.aliveServer(), ","))
	case "POST":
		addr := req.Header.Get("X-GeeRPC-Servers")
		if addr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.putServer(addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// HandleHTTP register a HTTP handler for GeeRegister message on registerPath
func (r *GeeRegister) HandleHTTP(registerPath string) {
	http.Handle(registerPath, r)
	log.Println("rpc register path: ", registerPath)
}

func HandleHTTP() {
	DefaultGeeRegister.HandleHTTP(defaultPath)
}

// Heartbeat send a heartbeat message every once in a while
// it's a helper function for server to register or send a heartbeat
func Heartbeat(register, addr string, duration time.Duration)  {
	if duration == 0 {
		// make sure there is enough time to send heartbeat message before
		// removed from register
		duration = defaultTimeout - time.Minute
	}
	var err error
	err = sendHeartbeat(register, addr)
	go func() {
		t := time.NewTicker(duration)
		for err == nil {
			<-t.C
			err = sendHeartbeat(register, addr)
		}
	}()
}

func sendHeartbeat(register, addr string) error{
	log.Printf("%s send heart beat to register %s", addr, register)
	httpClient := http.Client{}
	req, _ := http.NewRequest("POST", register, nil)
	req.Header.Set("X-GeeRPC-Servers", addr)
	if _, err := httpClient.Do(req); err != nil {
		log.Println("rpc server: heart beat error ", err)
		return err
	}
	return nil
}