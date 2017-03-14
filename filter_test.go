package main

import (
	"testing"

	"github.com/rancher/go-rancher/v2"
	"github.com/stretchr/testify/assert"
)

func TestFilterEnvironment(t *testing.T) {

	assert := assert.New(t)
	rancher := NewRancherContext(nil)

	environment := client.Project{Name: "Default"}
	assert.True(filterEnvironment(rancher, environment, "*"))
	assert.True(filterEnvironment(rancher, environment, "Default"))
	assert.True(filterEnvironment(rancher, environment, "Default,prod,dev,test"))
	assert.True(filterEnvironment(rancher, environment, "-*-Default,Default,prod,dev,test"))

	environment = client.Project{Name: "myuser-Default"}
	assert.False(filterEnvironment(rancher, environment, "Default,prod,dev,test"))
	assert.False(filterEnvironment(rancher, environment, "-*-Default,Default,prod,dev,test"))
}

func TestFilterHost(t *testing.T) {

	assert := assert.New(t)
	rancher := NewRancherContext(nil)

	rancher.AddEnvironment(client.Project{Name: "Default", Resource: client.Resource{Id: "1a5"}})
	host := client.Host{Hostname: "agent01.mysite.com", AccountId: "1a5", Labels: map[string]interface{}{"monitor": "true", "stage": "develop"}}

	assert.True(filterHost(rancher, host, ""))
	assert.True(filterHost(rancher, host, "*"))
	assert.True(filterHost(rancher, host, "+agent01.mysite.com"))
	assert.True(filterHost(rancher, host, "agent01.mysite.com"))
	assert.True(filterHost(rancher, host, "agent01.mysite.com"))
	assert.True(filterHost(rancher, host, "agent01.mysite.com,stage=develop"))
	assert.False(filterHost(rancher, host, "agent02.mysite.com"))
	assert.False(filterHost(rancher, host, "*,-stage=develop"))
	assert.False(filterHost(rancher, host, "agent01.mysite.com,-stage=develop"))
	assert.False(filterHost(rancher, host, "-agent01.mysite.com!L,stage=develop"))
	assert.True(filterHost(rancher, host, "%ENV=Default"))
	assert.False(filterHost(rancher, host, "%ENV=something"))
	assert.False(filterHost(rancher, host, "*,-%ENV=Default"))
}

func TestFilterStack(t *testing.T) {

	assert := assert.New(t)
	rancher := NewRancherContext(nil)

	rancher.AddEnvironment(client.Project{Name: "Default", Resource: client.Resource{Id: "1a5"}})
	rancher.AddService(client.Service{Name: "service1", AccountId: "1a5", Resource: client.Resource{Id: "3a1"}, LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{"monitor": "true"}}})

	stack1 := client.Stack{Name: "mygreatapp", AccountId: "1a5", ServiceIds: []string{"3a1"}}
	stack2 := client.Stack{Name: "healthcheck", AccountId: "1a5", System: true}

	assert.True(filterStack(rancher, stack1, ""))
	assert.True(filterStack(rancher, stack1, "*"))
	assert.True(filterStack(rancher, stack1, "mygreatapp"))
	assert.True(filterStack(rancher, stack1, "%ENV=Default"))
	assert.False(filterStack(rancher, stack1, "%ENV=another"))
	assert.False(filterStack(rancher, stack1, "%SYSTEM"))
	assert.True(filterStack(rancher, stack1, "%HAS_SERVICE(service1)"))
	assert.True(filterStack(rancher, stack1, "%HAS_SERVICE(monitor=true)"))

	assert.True(filterStack(rancher, stack2, "%SYSTEM"))
	assert.False(filterStack(rancher, stack2, "-%SYSTEM"))
}

func TestFilterService(t *testing.T) {

	assert := assert.New(t)
	rancher := NewRancherContext(nil)

	rancher.AddEnvironment(client.Project{Name: "Default", Resource: client.Resource{Id: "1a5"}})
	rancher.AddStack(client.Stack{Name: "mystack", AccountId: "1a5", ServiceIds: []string{"3a1", "3a2"}})

	service1 := client.Service{Name: "service1", AccountId: "1a5", Resource: client.Resource{Id: "3a1"}, LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{"monitor": "true"}}}
	service2 := client.Service{Name: "service2", AccountId: "1a5", Resource: client.Resource{Id: "3a2"}, LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{"monitor": "false"}}, System: true}

	assert.True(filterService(rancher, service1, ""))
	assert.True(filterService(rancher, service1, "*"))
	assert.False(filterService(rancher, service1, "blub"))
	assert.True(filterService(rancher, service1, "%STACK=mystack"))
	assert.False(filterService(rancher, service1, "%STACK=anotherstack"))
	assert.True(filterService(rancher, service1, "monitor=true"))
	assert.False(filterService(rancher, service1, "monitor=whatever"))
	assert.True(filterService(rancher, service1, "%ENV=Default"))
	assert.False(filterService(rancher, service1, "%ENV=another"))
	assert.False(filterService(rancher, service1, "%SYSTEM"))

	assert.True(filterService(rancher, service2, ""))
	assert.False(filterService(rancher, service2, "monitor=true"))
	assert.True(filterService(rancher, service2, "%SYSTEM"))
	assert.False(filterService(rancher, service2, "-%SYSTEM"))
}

func TestExample1(t *testing.T) {

	assert := assert.New(t)
	rancher := NewRancherContext(nil)

	rancher.AddEnvironment(client.Project{Name: "prod", Resource: client.Resource{Id: "1a5"}})
	rancher.AddEnvironment(client.Project{Name: "dev", Resource: client.Resource{Id: "1a6"}})

	prod1 := client.Host{Hostname: "prod01.mysite.com", AccountId: "1a5"}
	prod2 := client.Host{Hostname: "prod02.mysite.com", AccountId: "1a5"}

	dev1 := client.Host{Hostname: "dev01.mysite.com", AccountId: "1a6"}
	dev2 := client.Host{Hostname: "dev02.mysite.com", AccountId: "1a6"}

	system_p := client.Stack{Name: "systemstack", AccountId: "1a5", Resource: client.Resource{Id: "2a1"}, ServiceIds: []string{"3a1"}, System: true}
	system_d := client.Stack{Name: "systemstack", AccountId: "1a6", Resource: client.Resource{Id: "2b1"}, ServiceIds: []string{"3b1"}, System: true}

	app_p1 := client.Stack{Name: "myapp", AccountId: "1a5", Resource: client.Resource{Id: "2a2"}, ServiceIds: []string{"3a2"}}
	app_d1 := client.Stack{Name: "myapp", AccountId: "1a6", Resource: client.Resource{Id: "2b2"}, ServiceIds: []string{"3b2"}}

	sys_serv_p := client.Service{Name: "system-service", AccountId: "1a5", Resource: client.Resource{Id: "3a1"}, StackId: "2a1", System: true}
	sys_serv_d := client.Service{Name: "system-service", AccountId: "1a6", Resource: client.Resource{Id: "3b1"}, StackId: "2b1", System: true}

	app_serv_p := client.Service{Name: "myapp-service", AccountId: "1a5", Resource: client.Resource{Id: "3a1"}, StackId: "2a2"}
	app_serv_d := client.Service{Name: "myapp-service", AccountId: "1a6", Resource: client.Resource{Id: "3b2"}, StackId: "2b2"}

	rancher.AddStack(system_p)
	rancher.AddStack(system_d)
	rancher.AddStack(app_p1)
	rancher.AddStack(app_d1)

	filterHosts := "*"
	filterStacks := "*,-%ENV=dev,%SYSTEM"
	filterStacksV2 := "*,%SYSTEM!L,-%ENV=dev"
	filterServices := "*,-%ENV=dev,%SYSTEM"

	assert.True(filterHost(rancher, prod1, filterHosts))
	assert.True(filterHost(rancher, prod2, filterHosts))
	assert.True(filterHost(rancher, dev1, filterHosts))
	assert.True(filterHost(rancher, dev2, filterHosts))

	assert.True(filterStack(rancher, system_p, filterStacks))
	assert.True(filterStack(rancher, system_d, filterStacks))
	assert.True(filterStack(rancher, app_p1, filterStacks))
	assert.False(filterStack(rancher, app_d1, filterStacks))

	assert.True(filterStack(rancher, system_p, filterStacksV2))
	assert.True(filterStack(rancher, system_d, filterStacksV2))
	assert.True(filterStack(rancher, app_p1, filterStacksV2))
	assert.False(filterStack(rancher, app_d1, filterStacksV2))

	assert.True(filterService(rancher, sys_serv_p, filterServices))
	assert.True(filterService(rancher, sys_serv_d, filterServices))
	assert.True(filterService(rancher, app_serv_p, filterServices))
	assert.False(filterService(rancher, app_serv_d, filterServices))
}

func TestExample2(t *testing.T) {

	assert := assert.New(t)
	rancher := NewRancherContext(nil)

	rancher.AddEnvironment(client.Project{Name: "prod", Resource: client.Resource{Id: "1a5"}})
	rancher.AddEnvironment(client.Project{Name: "dev", Resource: client.Resource{Id: "1a6"}})

	prod1 := client.Host{Hostname: "prod01.mysite.com", AccountId: "1a5"}
	prod2 := client.Host{Hostname: "prod02.mysite.com", AccountId: "1a5"}

	system := client.Stack{Name: "systemstack", AccountId: "1a5", Resource: client.Resource{Id: "2a1"}, ServiceIds: []string{"3a1"}, System: true}

	app1 := client.Stack{Name: "app1", AccountId: "1a5", Resource: client.Resource{Id: "2a2"}, ServiceIds: []string{"a11", "a12"}}
	app2 := client.Stack{Name: "app2", AccountId: "1a5", Resource: client.Resource{Id: "2a3"}, ServiceIds: []string{"a21", "a22"}}

	sys_serv := client.Service{Name: "system-service", AccountId: "1a5", Resource: client.Resource{Id: "3a1"}, StackId: "2a1", System: true, LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{}}}

	app_serv11 := client.Service{Name: "app1-service1", AccountId: "1a5", Resource: client.Resource{Id: "a11"}, StackId: "2a2", LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{}}}
	app_serv12 := client.Service{Name: "app1-service2", AccountId: "1a5", Resource: client.Resource{Id: "a12"}, StackId: "2a2", LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{"monitor": "true"}}}
	app_serv21 := client.Service{Name: "app2-service1", AccountId: "1a5", Resource: client.Resource{Id: "a21"}, StackId: "2a3", LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{}}}
	app_serv22 := client.Service{Name: "app2-service2", AccountId: "1a5", Resource: client.Resource{Id: "a22"}, StackId: "2a3", LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{}}}

	rancher.AddStack(system)
	rancher.AddStack(app1)
	rancher.AddStack(app2)

	rancher.AddService(sys_serv)
	rancher.AddService(app_serv11)
	rancher.AddService(app_serv12)
	rancher.AddService(app_serv21)
	rancher.AddService(app_serv22)

	filterHosts := "*"
	filterStacks := "-*,%SYSTEM,%HAS_SERVICE(monitor=true)"
	filterServices := "-*,%SYSTEM,monitor=true"

	assert.True(filterHost(rancher, prod1, filterHosts))
	assert.True(filterHost(rancher, prod2, filterHosts))

	assert.True(filterStack(rancher, system, filterStacks))
	assert.True(filterStack(rancher, app1, filterStacks))
	assert.False(filterStack(rancher, app2, filterStacks))

	assert.True(filterService(rancher, sys_serv, filterServices))
	assert.False(filterService(rancher, app_serv11, filterServices))
	assert.True(filterService(rancher, app_serv12, filterServices))
	assert.False(filterService(rancher, app_serv21, filterServices))
	assert.False(filterService(rancher, app_serv22, filterServices))
}
