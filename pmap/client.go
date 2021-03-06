package pmap

import (
	"errors"
	"log"
	"net"
	"os"
)

func NewClient() (*client, error) {
	conn, err := net.Dial("tcp", pmapAddr)
	if err != nil {
		return nil, err
	}
	pmc := newPMapConn(conn)
	cl := client{pmc: pmc, respChan: make(chan Response)}
	go cl.clientLoop()
	return &cl, nil
}

func (cl *client) Close() {
	cl.pmc.shutdown()
}

type client struct {
	pmc      *pmapConn
	respChan chan Response
}

func (c *client) call(req Request) (Response, error) {
	c.pmc.out.Encode(&req)
	resp, ok := <-c.respChan
	if !ok {
		return resp, errors.New("Socket closed")
	}
	return resp, nil
}

func (c *client) clientLoop() {
	for {
		var resp Response
		err := c.pmc.in.Decode(&resp)
		if err != nil {
			close(c.respChan)
			log.Fatal(err)
		}
		c.respChan <- resp
	}
}

func (c *client) ListRegex(pattern string) ([]Registration, error) {
	resp, err := c.call(Request{List: &pattern})
	if err != nil {
		return nil, err
	}
	return *resp.List, nil
}

func (c *client) List() ([]Registration, error) {
	return c.ListRegex(".*")
}

func (c *client) Register(name string, addr string) (bool, error) {
	resp, err := c.call(Request{Register: &Registration{Name: name, Addr: addr, Pid: os.Getpid()}})
	if err != nil {
		return false, err
	}
	if resp.Error != nil {
		return false, errors.New(*resp.Error)
	} else if resp.Register == nil {
		return false, errors.New("Malformed response, neither register nor error was set")
	}
	return *resp.Register, nil
}
