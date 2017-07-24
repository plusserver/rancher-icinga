// Wraps the rancher GO API client for caching and easier testing.

package main

import (
	"errors"

	"github.com/rancher/go-rancher/v2"
)

type RancherGenClient interface {
	AddEnvironment(client.Project)
	Environments() (*client.ProjectCollection, error)
	GetEnvironment(string) client.Project
	DeleteEnvironment(string) error
	AddHost(client.Host)
	Hosts() (*client.HostCollection, error)
	GetHost(string) client.Host
	DeleteHost(string) error
	AddStack(client.Stack)
	Stacks() (*client.StackCollection, error)
	GetStack(string) client.Stack
	DeleteStack(string) error
	AddService(client.Service)
	Services() (*client.ServiceCollection, error)
	GetService(string) client.Service
	DeleteService(string) error
}

type RancherWebClient struct {
	rancher      *client.RancherClient
	environments map[string]client.Project
	hosts        map[string]client.Host
	stacks       map[string]client.Stack
	services     map[string]client.Service
}

type RancherMockClient struct {
	environments map[string]client.Project
	hosts        map[string]client.Host
	stacks       map[string]client.Stack
	services     map[string]client.Service
}

func NewRancherWebClient(rancher *client.RancherClient) *RancherWebClient {
	r := new(RancherWebClient)
	r.rancher = rancher
	r.environments = make(map[string]client.Project)
	r.stacks = make(map[string]client.Stack)
	r.services = make(map[string]client.Service)
	r.hosts = make(map[string]client.Host)
	return r
}

func NewRancherMockClient() *RancherMockClient {
	r := new(RancherMockClient)
	r.environments = make(map[string]client.Project)
	r.stacks = make(map[string]client.Stack)
	r.services = make(map[string]client.Service)
	r.hosts = make(map[string]client.Host)
	return r
}

// ---------

func (r *RancherWebClient) AddEnvironment(env client.Project) {
	r.environments[env.Id] = env
}

func (r *RancherWebClient) AddStack(stack client.Stack) {
	r.stacks[stack.Id] = stack
}

func (r *RancherWebClient) AddService(service client.Service) {
	r.services[service.Id] = service
}

func (r *RancherWebClient) AddHost(host client.Host) {
	r.hosts[host.Id] = host
}

func (r *RancherWebClient) Environments() (environments *client.ProjectCollection, err error) {
	envList, err := r.rancher.Project.List(nil)
	envArr := envList.Data

	for envList.Pagination != nil && envList.Pagination.Partial {
		envList, err = envList.Next()
		if err != nil {
			return
		}

		envArr = append(envArr, envList.Data...)
	}

	if err != nil {
		return
	}
	for _, env := range envArr {
		r.AddEnvironment(env)
	}
	return &client.ProjectCollection{Data: envArr}, nil
}

func (r *RancherWebClient) Hosts() (hosts *client.HostCollection, err error) {
	hostList, err := r.rancher.Host.List(nil)
	hostArr := hostList.Data

	for hostList.Pagination != nil && hostList.Pagination.Partial {
		hostList, err = hostList.Next()
		if err != nil {
			return
		}

		hostArr = append(hostArr, hostList.Data...)
	}

	if err != nil {
		return
	}
	for _, env := range hostArr {
		r.AddHost(env)
	}
	return &client.HostCollection{Data: hostArr}, nil
}

func (r *RancherWebClient) GetEnvironment(id string) client.Project {
	if _, ok := r.environments[id]; !ok {
		x, _ := r.rancher.Project.ById(id)
		r.environments[id] = *x
	}
	return r.environments[id]
}

func (r *RancherWebClient) GetHost(id string) client.Host {
	if _, ok := r.hosts[id]; !ok {
		x, _ := r.rancher.Host.ById(id)
		r.hosts[id] = *x
	}

	return r.hosts[id]
}

func (r *RancherWebClient) Stacks() (stacks *client.StackCollection, err error) {
	stackList, err := r.rancher.Stack.List(nil)
	stackArr := stackList.Data

	for stackList.Pagination != nil && stackList.Pagination.Partial {
		stackList, err = stackList.Next()
		if err != nil {
			return
		}

		stackArr = append(stackArr, stackList.Data...)
	}

	if err != nil {
		return
	}
	for _, env := range stackArr {
		r.AddStack(env)
	}
	return &client.StackCollection{Data: stackArr}, nil
}

func (r *RancherWebClient) GetStack(id string) client.Stack {
	if _, ok := r.stacks[id]; !ok {
		x, _ := r.rancher.Stack.ById(id)
		r.stacks[id] = *x
	}
	return r.stacks[id]
}

func (r *RancherWebClient) Services() (services *client.ServiceCollection, err error) {
	serviceList, err := r.rancher.Service.List(nil)
	serviceArr := serviceList.Data

	for serviceList.Pagination != nil && serviceList.Pagination.Partial {
		serviceList, err = serviceList.Next()
		if err != nil {
			return
		}

		serviceArr = append(serviceArr, serviceList.Data...)
	}

	if err != nil {
		return
	}
	for _, env := range serviceArr {
		r.AddService(env)
	}
	return &client.ServiceCollection{Data: serviceArr}, nil
}

func (r *RancherWebClient) GetService(id string) client.Service {
	if _, ok := r.services[id]; !ok {
		x, _ := r.rancher.Service.ById(id)
		r.services[id] = *x
	}
	return r.services[id]
}

func (r *RancherWebClient) DeleteService(id string) error {
	return errors.New("deleting objects not supported in Rancher web client")
}

func (r *RancherWebClient) DeleteStack(id string) error {
	return errors.New("deleting objects not supported in Rancher web client")
}

func (r *RancherWebClient) DeleteHost(id string) error {
	return errors.New("deleting objects not supported in Rancher web client")
}

func (r *RancherWebClient) DeleteEnvironment(id string) error {
	return errors.New("deleting objects not supported in Rancher web client")
}

// ---------

func (r *RancherMockClient) AddEnvironment(env client.Project) {
	r.environments[env.Id] = env
}

func (r *RancherMockClient) AddHost(host client.Host) {
	r.hosts[host.Id] = host
}

func (r *RancherMockClient) AddStack(stack client.Stack) {
	r.stacks[stack.Id] = stack
}

func (r *RancherMockClient) AddService(service client.Service) {
	r.services[service.Id] = service
}

func (r *RancherMockClient) GetEnvironment(id string) client.Project {
	return r.environments[id]
}

func (r *RancherMockClient) GetHost(id string) client.Host {
	return r.hosts[id]
}

func (r *RancherMockClient) GetStack(id string) client.Stack {
	return r.stacks[id]
}

func (r *RancherMockClient) GetService(id string) client.Service {
	return r.services[id]
}

func (r *RancherMockClient) Environments() (*client.ProjectCollection, error) {
	coll := make([]client.Project, 0, len(r.environments))

	for _, e := range r.environments {
		coll = append(coll, e)
	}

	return &client.ProjectCollection{Data: coll}, nil
}

func (r *RancherMockClient) Hosts() (*client.HostCollection, error) {
	coll := make([]client.Host, 0, len(r.hosts))

	for _, e := range r.hosts {
		coll = append(coll, e)
	}

	return &client.HostCollection{Data: coll}, nil
}

func (r *RancherMockClient) Stacks() (*client.StackCollection, error) {
	coll := make([]client.Stack, 0, len(r.stacks))

	for _, e := range r.stacks {
		coll = append(coll, e)
	}

	return &client.StackCollection{Data: coll}, nil
}

func (r *RancherMockClient) Services() (*client.ServiceCollection, error) {
	coll := make([]client.Service, 0, len(r.services))

	for _, e := range r.services {
		coll = append(coll, e)
	}

	return &client.ServiceCollection{Data: coll}, nil
}

func (r *RancherMockClient) DeleteService(id string) error {
	delete(r.services, id)
	return nil
}

func (r *RancherMockClient) DeleteStack(id string) error {
	delete(r.stacks, id)
	return nil
}

func (r *RancherMockClient) DeleteHost(id string) error {
	delete(r.hosts, id)
	return nil
}

func (r *RancherMockClient) DeleteEnvironment(id string) error {
	delete(r.environments, id)
	return nil
}
