// Wraps the rancher GO API client for caching and easier testing.

package main

import (
	"github.com/rancher/go-rancher/v2"
)

type RancherContext struct {
	rancher      *client.RancherClient
	environments map[string]client.Project
	stacks       map[string]client.Stack
	services     map[string]client.Service
}

func NewRancherContext(rancher *client.RancherClient) *RancherContext {
	r := new(RancherContext)
	r.rancher = rancher
	r.environments = make(map[string]client.Project)
	r.stacks = make(map[string]client.Stack)
	r.services = make(map[string]client.Service)

	return r
}

func (r *RancherContext) AddEnvironment(env client.Project) {
	r.environments[env.Id] = env
}

func (r *RancherContext) Environments() (environments *client.ProjectCollection, err error) {
	environments, err = r.rancher.Project.List(nil)
	if err != nil {
		return 
	}
	for _, env := range environments.Data {
		r.AddEnvironment(env)
	}
	
	return
}

func (r *RancherContext) GetEnvironment(id string) client.Project {
	return r.environments[id]
}

func (r *RancherContext) AddStack(stack client.Stack) {
	r.stacks[stack.Id] = stack
}

func (r *RancherContext) Stacks() (stacks *client.StackCollection, err error) {
	stacks, err = r.rancher.Stack.List(nil)
	if err != nil {
		return
	}
	for _, env := range stacks.Data {
		r.AddStack(env)
	}
	return
}

func (r *RancherContext) GetStack(id string) client.Stack {
	return r.stacks[id]
}

func (r *RancherContext) AddService(service client.Service) {
	r.services[service.Id] = service
}

func (r *RancherContext) Services() (services *client.ServiceCollection, err error) {
	services, err = r.rancher.Service.List(nil)
	if err != nil {
		return
	}
	for _, env := range services.Data {
		r.AddService(env)
	}
	return
}

func (r *RancherContext) GetService(id string) client.Service {
	return r.services[id]
}
