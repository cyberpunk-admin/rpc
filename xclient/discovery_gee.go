package xclient

import (
	"log"
	"net/http"
	"strings"
	"time"
)

type GeeRegisterDiscovery struct {
	*MultiServersDiscovery
	register   string
	timeout    time.Duration
	lastUpdate time.Time
}

const defaultUpdateTimeout = time.Second * 10

func (d *GeeRegisterDiscovery) Update(server []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.servers = server
	d.lastUpdate = time.Now()
	return nil
}

func (d *GeeRegisterDiscovery) Refresh() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.lastUpdate.Add(d.timeout).After(time.Now()) {
		return nil
	}
	log.Println("rpc registry: refresh server from registry", d.register)
	resp, err := http.Get(d.register)
	if err != nil {
		log.Println("rpc registry: refresh error ", err)
		return err
	}
	servers := strings.Split(resp.Header.Get("X-GeeRPC-Servers"), ",")
	log.Println("server list: ", servers)
	d.servers = make([]string, 0, len(servers))
	for _, server := range servers {
		if strings.TrimSpace(server) != "" {
			d.servers = append(d.servers, strings.TrimSpace(server))
		}
	}
	d.lastUpdate = time.Now()
	return nil
}

func (d *GeeRegisterDiscovery) Get(mode SelectMode) (string, error) {
	if err := d.Refresh(); err != nil {
		return "", err
	}
	return d.MultiServersDiscovery.Get(mode)
}

func (d *GeeRegisterDiscovery) GetAll() ([]string, error) {
	if err := d.Refresh(); err != nil {
		return nil, err
	}
	return d.MultiServersDiscovery.GetAll()
}

func NewGeeRegisterDiscovery(registerAddr string, timeout time.Duration) *GeeRegisterDiscovery {
	if timeout == 0 {
		timeout = defaultUpdateTimeout
	}
	d := &GeeRegisterDiscovery{
		MultiServersDiscovery: NewMultiServersDiscovery([]string{}),
		register:              registerAddr,
		timeout:               timeout,
	}
	return d
}
