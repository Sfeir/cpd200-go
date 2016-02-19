package main

import (
	"log"
	"github.com/GoogleCloudPlatform/go-endpoints/endpoints"
	"golang.org/x/net/context"
)

type HelloWorldApi struct {
}

type Hello struct {
	Greeting string	`json:"greeting"`
}

type NameReq struct {
	Name string	`json:"name"`
}

func (h *HelloWorldApi) SayHello(c context.Context) (*Hello, error) {
	return &Hello{Greeting:"Hello World"}, nil
}

func (h *HelloWorldApi) SayHelloByName(c context.Context, r *NameReq) (*Hello, error) {
	return &Hello{Greeting:"Hello World " + r.Name}, nil
}

func init() {
	hello := &HelloWorldApi{}
	api, err := endpoints.RegisterService(hello, "helloworldendpoints", "v1", "Hello World API", true)
	if err != nil {
		log.Fatalf("Register service: %v", err)
	}
	
	register := func(orig, name, method, path, desc string) {
		m := api.MethodByName(orig)
		if m == nil {
			log.Fatalf("Missing method %s", orig)
		}
		i := m.Info()
		i.Name, i.HTTPMethod, i.Path, i.Desc = name, method, path, desc
	}

	register("SayHello", "sayHello", "GET", "sayHello", "Say hello")
	register("SayHelloByName", "sayHelloByName", "GET", "sayHelloByName", "Say hello by name")
	endpoints.HandleHTTP()
}
