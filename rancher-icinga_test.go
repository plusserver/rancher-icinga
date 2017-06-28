package main

import (
	//"fmt"
	"testing"
	"text/template"

	"github.com/Nexinto/go-icinga2-client/icinga2"
	"github.com/rancher/go-rancher/v2"
	"github.com/stretchr/testify/assert"
)

func initForTests() *RancherIcingaConfig {

	rancher := NewRancherMockClient()

	icinga := icinga2.NewMockClient()
	config, _ := NewBaseConfig()

	config.rancher = rancher
	config.icinga = icinga

	return config
}

func TestHostgroup(t *testing.T) {

	assert := assert.New(t)
	config := initForTests()

	config.rancher.AddEnvironment(client.Project{Name: "Default", Resource: client.Resource{Id: "1a5"}})

	err := sync(config)
	assert.Nil(err)

	hostGroups, _ := config.icinga.ListHostGroups()

	assert.NotEmpty(hostGroups)
	assert.Equal(len(hostGroups), 1)

	defGroup := hostGroups[0]

	assert.Equal(defGroup.Name, "Default")
}

func TestHostgroupDefaultIcingaVars(t *testing.T) {

	assert := assert.New(t)
	config := initForTests()

	config.hostgroupDefaultIcingaVars = icinga2.Vars{"myvar": "yes"}
	config.rancher.AddEnvironment(client.Project{Name: "Default", Resource: client.Resource{Id: "1a5"}})

	err := sync(config)
	assert.Nil(err)

	hostGroups, _ := config.icinga.ListHostGroups()

	assert.NotEmpty(hostGroups)
	assert.Equal(len(hostGroups), 1)

	myGroup := hostGroups[0]

	assert.Equal("yes", myGroup.Vars["myvar"])
	assert.Equal("default", myGroup.Vars["rancher_installation"])
	assert.Equal("Default", myGroup.Vars["rancher_environment"])
	assert.Equal("environment", myGroup.Vars["rancher_object_type"])

}

func TestTwoHostgroups(t *testing.T) {

	assert := assert.New(t)
	config := initForTests()

	config.rancher.AddEnvironment(client.Project{Name: "First", Resource: client.Resource{Id: "1a5"}})
	config.rancher.AddEnvironment(client.Project{Name: "Second", Resource: client.Resource{Id: "1a6"}})

	err := sync(config)
	assert.Nil(err)

	hostGroups, err := config.icinga.ListHostGroups()

	assert.Nil(err)

	assert.NotEmpty(hostGroups)
	assert.Equal(len(hostGroups), 2)

	first, _ := config.icinga.GetHostGroup("First")
	second, _ := config.icinga.GetHostGroup("Second")

	assert.Equal("First", first.Name)
	assert.Equal("Second", second.Name)
}

func TestHost(t *testing.T) {

	assert := assert.New(t)
	config := initForTests()

	config.rancher.AddEnvironment(client.Project{Name: "Default", Resource: client.Resource{Id: "1a5"}})

	config.rancher.AddHost(client.Host{
		Hostname:  "agent1",
		AccountId: "1a5",
		Resource:  client.Resource{Id: "2a1"},
		Labels:    map[string]interface{}{"icinga.host_notes_url": "http://docs.mysite.com/panic/agent_down.html"}})

	err := sync(config)
	assert.Nil(err)

	hosts, err := config.icinga.ListHosts()

	assert.Nil(err)
	assert.Equal(1, len(hosts), "We should have exactly 1 host")

	host := hosts[0]

	assert.Equal("agent1", host.Name)
	assert.Equal("hostalive", host.CheckCommand)
	assert.Equal("http://docs.mysite.com/panic/agent_down.html", host.NotesURL)
	assert.Equal("host", host.Vars[RANCHER_OBJECT_TYPE])
	assert.Equal("Default", host.Vars[RANCHER_ENVIRONMENT])
	assert.Equal("default", host.Vars[RANCHER_INSTALLATION])

	services, err := config.icinga.ListServices()

	assert.Nil(err)
	assert.Equal(1, len(services), "We should have exactly 1 service")

	service := services[0]

	assert.Equal("rancher-agent", service.Name)
	assert.Equal("agent1", service.HostName)
	assert.Equal("check_rancher_host", service.CheckCommand)
	assert.Equal("http://docs.mysite.com/panic/agent_down.html", service.NotesURL)
	assert.Equal("rancher-agent", service.Vars[RANCHER_OBJECT_TYPE])
	assert.Equal("Default", service.Vars[RANCHER_ENVIRONMENT])
	assert.Equal("default", service.Vars[RANCHER_INSTALLATION])

	// Add a host

	config.rancher.AddHost(client.Host{Hostname: "agent2", AccountId: "1a5", Resource: client.Resource{Id: "2a2"}})

	err = sync(config)
	assert.Nil(err)

	hosts, err = config.icinga.ListHosts()

	assert.Nil(err)
	assert.Equal(2, len(hosts), "We should have exactly 2 hosts")

	foundAgent1 := false
	foundAgent2 := false

	for _, host := range hosts {
		if host.Name == "agent1" {
			foundAgent1 = true
			assert.Equal("agent1", host.Name)
			assert.Equal("hostalive", host.CheckCommand)
			assert.Equal("http://docs.mysite.com/panic/agent_down.html", host.NotesURL)
			assert.Equal("Default", host.Vars[RANCHER_ENVIRONMENT])
			assert.Equal("default", host.Vars[RANCHER_INSTALLATION])
			assert.Equal("host", host.Vars[RANCHER_OBJECT_TYPE])
		}

		if host.Name == "agent2" {
			foundAgent2 = true
			assert.Equal("agent2", host.Name)
			assert.Equal("hostalive", host.CheckCommand)
			assert.Empty(host.NotesURL)
			assert.Equal("Default", host.Vars[RANCHER_ENVIRONMENT])
			assert.Equal("default", host.Vars[RANCHER_INSTALLATION])
			assert.Equal("host", host.Vars[RANCHER_OBJECT_TYPE])
		}
	}

	assert.True(foundAgent1 && foundAgent2, "we should find both hosts")

	services, err = config.icinga.ListServices()

	assert.Nil(err)
	assert.Equal(2, len(services), "We should have exactly 2 services")

	foundService1 := false
	foundService2 := false

	for _, service := range services {
		if service.HostName == "agent1" {
			foundService1 = true
			assert.Equal("rancher-agent", service.Name)
			assert.Equal("agent1", service.HostName)
			assert.Equal("check_rancher_host", service.CheckCommand)
			assert.Equal("http://docs.mysite.com/panic/agent_down.html", service.NotesURL)
			assert.Equal("rancher-agent", service.Vars[RANCHER_OBJECT_TYPE])
			assert.Equal("Default", service.Vars[RANCHER_ENVIRONMENT])
			assert.Equal("default", service.Vars[RANCHER_INSTALLATION])

		}
		if service.HostName == "agent2" {
			foundService2 = true
			assert.Equal("rancher-agent", service.Name)
			assert.Equal("agent2", service.HostName)
			assert.Equal("check_rancher_host", service.CheckCommand)
			assert.Empty(service.NotesURL)
			assert.Equal("rancher-agent", service.Vars[RANCHER_OBJECT_TYPE])
			assert.Equal("Default", service.Vars[RANCHER_ENVIRONMENT])
			assert.Equal("default", service.Vars[RANCHER_INSTALLATION])

		}
	}

	assert.True(foundService1, "we should find service1")
	assert.True(foundService2, "we should find service2")

	// Remove an agent

	err = config.rancher.DeleteHost("2a1")
	assert.Nil(err)

	err = sync(config)
	assert.Nil(err)

	hosts, err = config.icinga.ListHosts()

	assert.Nil(err)
	assert.Equal(1, len(hosts), "We should have exactly 1 host")

	host = hosts[0]

	assert.Equal("agent2", host.Name)
	assert.Equal("hostalive", host.CheckCommand)
	assert.Empty(host.NotesURL)
	assert.Equal("host", host.Vars[RANCHER_OBJECT_TYPE])
	assert.Equal("Default", host.Vars[RANCHER_ENVIRONMENT])
	assert.Equal("default", host.Vars[RANCHER_INSTALLATION])

	services, err = config.icinga.ListServices()

	assert.Nil(err)
	assert.Equal(1, len(services), "We should have exactly 1 service")

	service = services[0]

	assert.Equal("rancher-agent", service.Name)
	assert.Equal("agent2", service.HostName)
	assert.Equal("check_rancher_host", service.CheckCommand)
	assert.Empty(service.NotesURL)
	assert.Equal("rancher-agent", service.Vars[RANCHER_OBJECT_TYPE])
	assert.Equal("Default", service.Vars[RANCHER_ENVIRONMENT])
	assert.Equal("default", service.Vars[RANCHER_INSTALLATION])
}

func TestStack(t *testing.T) {

	assert := assert.New(t)
	config := initForTests()

	config.rancher.AddEnvironment(client.Project{Name: "Default", Resource: client.Resource{Id: "1a5"}})
	config.rancher.AddStack(client.Stack{Name: "mystack", AccountId: "1a5", Resource: client.Resource{Id: "3a1"}})

	err := sync(config)
	assert.Nil(err)

	hosts, err := config.icinga.ListHosts()

	assert.Nil(err, "listing the hosts should not cause an error")
	assert.Equal(1, len(hosts), "we should have exactly one host")

	host := hosts[0]

	//fmt.Printf("%+v\n", host)

	assert.Equal("Default.mystack", host.Name)
	assert.Equal("check_rancher_stack", host.CheckCommand)
	assert.Equal("stack", host.Vars["rancher_object_type"])
	assert.Equal("Default", host.Vars["rancher_environment"])
	assert.Equal("mystack", host.Vars["rancher_stack"])
	assert.NotNil(host.Groups, "the stack should be in its environment host group")
	assert.Equal(1, len(host.Groups), "the stack should be in its environment host group only")
	assert.Equal("Default", host.Groups[0], "the stack should be in its environment host group")

	// Add a second stack

	config.rancher.AddStack(client.Stack{Name: "mystack2", AccountId: "1a5", Resource: client.Resource{Id: "3a2"}})

	err = sync(config)
	assert.Nil(err)

	hosts, err = config.icinga.ListHosts()

	assert.Nil(err, "listing the hosts should not cause an error")
	assert.Equal(2, len(hosts), "we should have two hosts now")

	foundHost1 := false
	foundHost2 := false

	for _, host := range hosts {
		if host.Vars["rancher_stack"] == "mystack" {
			foundHost1 = true
			assert.Equal("Default.mystack", host.Name)
			assert.Equal("check_rancher_stack", host.CheckCommand)
			assert.Equal("stack", host.Vars["rancher_object_type"])
			assert.Equal("Default", host.Vars["rancher_environment"])
			assert.Equal("mystack", host.Vars["rancher_stack"])
			assert.NotNil(host.Groups, "the stack should be in its environment host group")
			assert.Equal(1, len(host.Groups), "the stack should be in its environment host group only")
			assert.Equal("Default", host.Groups[0], "the stack should be in its environment host group")
		}
		if host.Vars["rancher_stack"] == "mystack2" {
			foundHost2 = true
			assert.Equal("Default.mystack2", host.Name)
			assert.Equal("check_rancher_stack", host.CheckCommand)
			assert.Equal("stack", host.Vars["rancher_object_type"])
			assert.Equal("Default", host.Vars["rancher_environment"])
			assert.Equal("mystack2", host.Vars["rancher_stack"])
			assert.NotNil(host.Groups, "the stack should be in its environment host group")
			assert.Equal(1, len(host.Groups), "the stack should be in its environment host group only")
			assert.Equal("Default", host.Groups[0], "the stack should be in its environment host group")
		}
	}

	assert.True(foundHost1, "we should find the host for mystack")
	assert.True(foundHost2, "we should find the host for mystack2")

	// Remove the first stack

	config.rancher.DeleteStack("3a1")

	err = sync(config)
	assert.Nil(err)

	hosts, err = config.icinga.ListHosts()

	assert.Nil(err, "listing the hosts should not cause an error")
	assert.Equal(1, len(hosts), "we should have exactly one host again")

	host = hosts[0]

	assert.Equal("Default.mystack2", host.Name)
	assert.Equal("check_rancher_stack", host.CheckCommand)
	assert.Equal("stack", host.Vars["rancher_object_type"])
	assert.Equal("Default", host.Vars["rancher_environment"])
	assert.Equal("mystack2", host.Vars["rancher_stack"])
	assert.NotNil(host.Groups, "the stack should be in its environment host group")
	assert.Equal(1, len(host.Groups), "the stack should be in its environment host group only")
	assert.Equal("Default", host.Groups[0], "the stack should be in its environment host group")
}

func TestStackDefaultIcingaVars(t *testing.T) {

	assert := assert.New(t)
	config := initForTests()

	config.stackDefaultIcingaVars = icinga2.Vars{"var1": "val1", "var2": "val2", "var3": "val3"}

	config.rancher.AddEnvironment(client.Project{Name: "Default", Resource: client.Resource{Id: "1a5"}})
	config.rancher.AddStack(client.Stack{Name: "mystack", AccountId: "1a5", Resource: client.Resource{Id: "3a1"}})

	err := sync(config)
	assert.Nil(err)

	hosts, _ := config.icinga.ListHosts()
	host := hosts[0]

	assert.Equal("val1", host.Vars["var1"])
	assert.Equal("val2", host.Vars["var2"])
	assert.Equal("val3", host.Vars["var3"])

	// Change the global config

	config.stackDefaultIcingaVars = icinga2.Vars{"var1": "val1", "var3": "newval3", "var4": "val4"}

	err = sync(config)
	assert.Nil(err)

	hosts, _ = config.icinga.ListHosts()
	host = hosts[0]

	assert.Equal("val1", host.Vars["var1"])
	assert.Nil(host.Vars["var2"])
	assert.Equal("newval3", host.Vars["var3"])
	assert.Equal("val4", host.Vars["var4"])

}

func TestStackLocalIcingaVars(t *testing.T) {

	assert := assert.New(t)
	config := initForTests()

	config.rancher.AddEnvironment(client.Project{Name: "Default", Resource: client.Resource{Id: "1a5"}})
	config.rancher.AddStack(client.Stack{
		Name:       "mystack",
		AccountId:  "1a5",
		Resource:   client.Resource{Id: "2a1"},
		ServiceIds: []string{"3a1", "3a2"}})
	config.rancher.AddService(client.Service{
		Name:         "service1",
		AccountId:    "1a5",
		StackId:      "2a1",
		Resource:     client.Resource{Id: "3a1"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{}}})
	config.rancher.AddService(client.Service{
		Name:      "service1",
		AccountId: "1a5",
		StackId:   "2a1",
		Resource:  client.Resource{Id: "3a2"},
		LaunchConfig: &client.LaunchConfig{
			Labels: map[string]interface{}{"icinga.stack_vars": "var1=val1,var2=val2,var3=val3"}}})

	err := sync(config)
	assert.Nil(err)

	hosts, err := config.icinga.ListHosts()

	assert.Nil(err, "listing the hosts should not cause an error")
	assert.Equal(1, len(hosts), "we should have exactly one host")

	host := hosts[0]

	assert.Equal("val1", host.Vars["var1"])
	assert.Equal("val2", host.Vars["var2"])
	assert.Equal("val3", host.Vars["var3"])

	// Change the labels from one service to the other and change its values

	config.rancher.AddService(client.Service{
		Name:      "service1",
		AccountId: "1a5",
		StackId:   "2a1",
		Resource:  client.Resource{Id: "3a1"},
		LaunchConfig: &client.LaunchConfig{
			Labels: map[string]interface{}{"icinga.stack_vars": "var1=val1,var3=newval3,var4=val4"}}})
	config.rancher.AddService(client.Service{
		Name:      "service2",
		AccountId: "1a5",
		StackId:   "2a1",
		Resource:  client.Resource{Id: "3a2"},
		LaunchConfig: &client.LaunchConfig{
			Labels: map[string]interface{}{}}})

	err = sync(config)
	assert.Nil(err)

	hosts, _ = config.icinga.ListHosts()
	host = hosts[0]

	assert.Equal("val1", host.Vars["var1"])
	assert.Nil(host.Vars["var2"])
	assert.Equal("newval3", host.Vars["var3"])
	assert.Equal("val4", host.Vars["var4"])
}

func TestStackDefaultAndLocalIcingaVars(t *testing.T) {

	assert := assert.New(t)
	config := initForTests()

	config.stackDefaultIcingaVars = icinga2.Vars{"var1": "def1", "var2": "def2", "var3": "def3"}

	config.rancher.AddEnvironment(client.Project{Name: "Default", Resource: client.Resource{Id: "1a5"}})
	config.rancher.AddStack(client.Stack{
		Name:       "mystack",
		AccountId:  "1a5",
		Resource:   client.Resource{Id: "2a1"},
		ServiceIds: []string{"3a1", "3a2"}})
	config.rancher.AddService(client.Service{
		Name:         "service1",
		AccountId:    "1a5",
		StackId:      "2a1",
		Resource:     client.Resource{Id: "3a1"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{}}})
	config.rancher.AddService(client.Service{
		Name:      "service2",
		AccountId: "1a5",
		StackId:   "2a1",
		Resource:  client.Resource{Id: "3a2"},
		LaunchConfig: &client.LaunchConfig{
			Labels: map[string]interface{}{"icinga.stack_vars": "var2=local2,var4=local4"}}})

	err := sync(config)
	assert.Nil(err)

	hosts, err := config.icinga.ListHosts()

	assert.Nil(err, "listing the hosts should not cause an error")
	assert.Equal(1, len(hosts), "we should have exactly one host")

	host := hosts[0]

	assert.Equal("def1", host.Vars["var1"])
	assert.Equal("local2", host.Vars["var2"])
	assert.Equal("def3", host.Vars["var3"])
	assert.Equal("local4", host.Vars["var4"])

	// Change both the global and the stack local configuration

	config.stackDefaultIcingaVars = icinga2.Vars{"var1": "newdef1", "var2": "def2", "var5": "def5"}

	config.rancher.AddService(client.Service{
		Name:      "service1",
		AccountId: "1a5",
		StackId:   "2a1",
		Resource:  client.Resource{Id: "3a1"},
		LaunchConfig: &client.LaunchConfig{
			Labels: map[string]interface{}{"icinga.stack_vars": "var4=newlocal4,var6=local6"}}})
	config.rancher.AddService(client.Service{
		Name:      "service1",
		AccountId: "1a5",
		StackId:   "2a1",
		Resource:  client.Resource{Id: "3a2"},
		LaunchConfig: &client.LaunchConfig{
			Labels: map[string]interface{}{}}})

	err = sync(config)
	assert.Nil(err)

	hosts, _ = config.icinga.ListHosts()
	host = hosts[0]

	assert.Equal("newdef1", host.Vars["var1"], "the first var should be new from the global configuration")
	assert.Equal("def2", host.Vars["var2"], "the second var should no longer be overridden from local")
	assert.Nil(host.Vars["var3"], "the third var should be removed")
	assert.Equal("newlocal4", host.Vars["var4"], "the fourth var should be updated from the local value")
	assert.Equal("def5", host.Vars["var5"], "the fifth var should be new from the global configuration")
	assert.Equal("local6", host.Vars["var6"], "the sixth var should be new from the local value")
}

// If I have a stack and a service and there is a icinga.stack_notes_url label somewhere, I want the
// corresponding Notes URL to appear at the Stack-Host level. I also want it to change when I change the label.
func TestStackNotesURL(t *testing.T) {

	assert := assert.New(t)
	config := initForTests()

	config.rancher.AddEnvironment(client.Project{Name: "Default", Resource: client.Resource{Id: "1a5"}})
	config.rancher.AddStack(client.Stack{
		Name:       "mystack",
		AccountId:  "1a5",
		Resource:   client.Resource{Id: "2a1"},
		ServiceIds: []string{"3a1", "3a2"}})
	config.rancher.AddService(client.Service{
		Name:         "service1",
		AccountId:    "1a5",
		StackId:      "2a1",
		Resource:     client.Resource{Id: "3a1"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{}}})
	config.rancher.AddService(client.Service{
		Name:      "service1",
		AccountId: "1a5",
		StackId:   "2a1",
		Resource:  client.Resource{Id: "3a2"},
		LaunchConfig: &client.LaunchConfig{
			Labels: map[string]interface{}{"icinga.stack_notes_url": "http://mysite.com/docs"}}})

	err := sync(config)
	assert.Nil(err)

	hosts, err := config.icinga.ListHosts()

	assert.Nil(err, "listing the hosts should not cause an error")
	assert.Equal(1, len(hosts), "we should have exactly one host")

	host := hosts[0]

	assert.Equal("http://mysite.com/docs", host.NotesURL)

	// Change the label for the Notes URL from one service to the other and change its value

	config.rancher.AddService(client.Service{
		Name:      "service1",
		AccountId: "1a5",
		StackId:   "2a1",
		Resource:  client.Resource{Id: "3a1"},
		LaunchConfig: &client.LaunchConfig{
			Labels: map[string]interface{}{"icinga.stack_notes_url": "http://docs.mysite.com/newstuff"}}})
	config.rancher.AddService(client.Service{
		Name:      "service2",
		AccountId: "1a5",
		StackId:   "2a1",
		Resource:  client.Resource{Id: "3a2"},
		LaunchConfig: &client.LaunchConfig{
			Labels: map[string]interface{}{}}})

	err = sync(config)
	assert.Nil(err)

	hosts, _ = config.icinga.ListHosts()
	host = hosts[0]

	assert.Equal("http://docs.mysite.com/newstuff", host.NotesURL)
}

func assertBasicServiceVars(assert *assert.Assertions, service icinga2.Service) {

	assert.Equal("service1", service.Name)

	assert.NotEmpty(service.Vars)
	assert.Equal("default", service.Vars["rancher_installation"], "the installation should be set")
	assert.Equal("Default", service.Vars["rancher_environment"], "the environment should be set")
	assert.Equal("service", service.Vars["rancher_object_type"], "the object type should be set")
	assert.Equal("mystack", service.Vars["rancher_stack"], "the stack should be set")
	assert.Equal("service1", service.Vars["rancher_service"], "the service should be set")
}

// A number of test with Rancher services
func TestService(t *testing.T) {

	assert := assert.New(t)
	config := initForTests()

	config.rancher.AddEnvironment(client.Project{Name: "Default", Resource: client.Resource{Id: "1a5"}})
	config.rancher.AddStack(client.Stack{
		Name:       "mystack",
		AccountId:  "1a5",
		Resource:   client.Resource{Id: "2a1"},
		ServiceIds: []string{"3a1"}})

	// Add a service

	config.rancher.AddService(client.Service{
		Name:         "service1",
		AccountId:    "1a5",
		StackId:      "2a1",
		Resource:     client.Resource{Id: "3a1"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{}}})

	err := sync(config)
	assert.Nil(err)

	services, err := config.icinga.ListServices()

	assert.Nil(err, "listing the services should not cause an error")

	assert.NotEmpty(services, "the list of services should not be empty")
	assert.Equal(1, len(services), "we should have exactly one service")

	service := services[0]

	assertBasicServiceVars(assert, service)

	// Add a second service

	config.rancher.AddStack(client.Stack{
		Name:       "mystack",
		AccountId:  "1a5",
		Resource:   client.Resource{Id: "2a1"},
		ServiceIds: []string{"3a1", "3a2"}}) // need to update the stack with the ServiceIDs
	config.rancher.AddService(client.Service{
		Name:         "service2",
		AccountId:    "1a5",
		StackId:      "2a1",
		Resource:     client.Resource{Id: "3a2"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{}}})

	err = sync(config)
	assert.Nil(err)

	services, err = config.icinga.ListServices()

	assert.Nil(err, "listing the services should not cause an error")

	assert.NotEmpty(services, "the list of services should not be empty")
	assert.Equal(len(services), 2, "we should have two services now")

	foundMyServices := false

	if services[0].Name == "service1" &&
		services[1].Name == "service2" &&
		services[0].Vars["rancher_service"] == "service1" &&
		services[1].Vars["rancher_service"] == "service2" ||
		services[1].Name == "service1" &&
			services[0].Name == "service2" &&
			services[1].Vars["rancher_service"] == "service1" &&
			services[0].Vars["rancher_service"] == "service2" {
		foundMyServices = true
	}

	assert.True(foundMyServices, "both services should be configured")

	// Remove the first service

	config.rancher.AddStack(client.Stack{
		Name:       "mystack",
		AccountId:  "1a5",
		Resource:   client.Resource{Id: "2a1"},
		ServiceIds: []string{"3a2"}}) // need to update the stack with the ServiceIDs
	config.rancher.DeleteService("3a1")

	err = sync(config)
	assert.Nil(err)

	services, err = config.icinga.ListServices()

	assert.Nil(err, "listing the services should not cause an error")

	assert.NotEmpty(services, "the list of services should not be empty")
	assert.Equal(1, len(services), "we should have one service again now")

	service = services[0]

	assert.Equal("service2", service.Name, "this should be the second service")
}

// Test: If I add a service label with a notes url, the icinga service must use this URL.
// If I change the service label, the icinga service must change as well.
func TestServiceNotesURL(t *testing.T) {

	assert := assert.New(t)
	config := initForTests()

	config.rancher.AddEnvironment(client.Project{Name: "Default", Resource: client.Resource{Id: "1a5"}})
	config.rancher.AddStack(client.Stack{Name: "mystack", AccountId: "1a5", ServiceIds: []string{"3a1"}})
	config.rancher.AddService(client.Service{
		Name:      "service1",
		AccountId: "1a5",
		Resource:  client.Resource{Id: "3a1"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{
			"icinga.service_notes_url": "http://docs.mysite.com/service1.html"}}})

	err := sync(config)
	assert.Nil(err)

	services, err := config.icinga.ListServices()

	assert.Nil(err, "listing the services should not cause an error")

	assert.NotEmpty(services, "the list of services should not be empty")
	assert.Equal(1, len(services), "we should have exactly one service")

	service := services[0]

	assert.Equal("service1", service.Name)
	assert.Equal("http://docs.mysite.com/service1.html", service.NotesURL, "the notes URL should be set")

	assertBasicServiceVars(assert, service)

	config.rancher.AddService(client.Service{
		Name:      "service1",
		AccountId: "1a5",
		Resource:  client.Resource{Id: "3a1"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{
			"icinga.service_notes_url": "http://newdocs.mysite.com/service1.html"}}})

	err = sync(config)
	assert.Nil(err)

	services, err = config.icinga.ListServices()

	assert.Nil(err, "listing the services should not cause an error")

	assert.NotEmpty(services, "the list of services should not be empty")
	assert.Equal(1, len(services), "we should have exactly one service")

	service = services[0]

	assertBasicServiceVars(assert, service)

	assert.Equal("http://newdocs.mysite.com/service1.html", service.NotesURL, "the notes URL should be updated")
}

// Test: I can set service default vars and they will appear at the service and can be changed later.
func TestServiceDefaultIcingaVars(t *testing.T) {

	assert := assert.New(t)
	config := initForTests()

	config.serviceDefaultIcingaVars = icinga2.Vars{"myvar": "hi_there", "myothervar": "hello", "mythirdvar": "yo"}

	config.rancher.AddEnvironment(client.Project{Name: "Default", Resource: client.Resource{Id: "1a5"}})
	config.rancher.AddStack(client.Stack{Name: "mystack", AccountId: "1a5", ServiceIds: []string{"3a1"}})
	config.rancher.AddService(client.Service{
		Name:         "service1",
		AccountId:    "1a5",
		Resource:     client.Resource{Id: "3a1"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{}}})

	err := sync(config)
	assert.Nil(err)

	services, err := config.icinga.ListServices()

	assert.Nil(err, "listing the services should not cause an error")

	assert.NotEmpty(services, "the list of services should not be empty")
	assert.Equal(1, len(services), "we should have exactly one service")

	service := services[0]

	assertBasicServiceVars(assert, service)

	assert.Equal("hi_there", service.Vars["myvar"], "the var should be set")
	assert.Equal("hello", service.Vars["myothervar"], "the var should be set")
	assert.Equal("yo", service.Vars["mythirdvar"], "the var should be set")

	config.serviceDefaultIcingaVars = icinga2.Vars{"myvar": "hi_there", "myothervar": "moin"}

	err = sync(config)
	assert.Nil(err)

	services, err = config.icinga.ListServices()

	assert.Nil(err, "listing the services should not cause an error")

	assert.NotEmpty(services, "the list of services should not be empty")
	assert.Equal(1, len(services), "we should have exactly one service")

	service = services[0]

	assertBasicServiceVars(assert, service)

	assert.Equal("hi_there", service.Vars["myvar"], "the var should be unchanged")
	assert.Equal("moin", service.Vars["myothervar"], "the var should be changed")
	assert.Nil(service.Vars["mythirdvar"], "the var should be removed")
}

// Test: I can set service vars as tags and they will appear at the service and can be changed later.
func TestServiceLocalIcingaVars(t *testing.T) {

	assert := assert.New(t)
	config := initForTests()

	config.rancher.AddEnvironment(client.Project{Name: "Default", Resource: client.Resource{Id: "1a5"}})
	config.rancher.AddStack(client.Stack{Name: "mystack", AccountId: "1a5", ServiceIds: []string{"3a1"}})
	config.rancher.AddService(client.Service{
		Name:      "service1",
		AccountId: "1a5",
		Resource:  client.Resource{Id: "3a1"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{
			"icinga.service_vars": "myvar=hello,somevar=whatever"}}})

	err := sync(config)
	assert.Nil(err)

	services, err := config.icinga.ListServices()

	assert.Nil(err, "listing the services should not cause an error")

	assert.NotEmpty(services, "the list of services should not be empty")
	assert.Equal(1, len(services), "we should have exactly one service")

	service := services[0]

	assertBasicServiceVars(assert, service)

	assert.Equal("hello", service.Vars["myvar"], "the var should be set")
	assert.Equal("whatever", service.Vars["somevar"], "the var should be set")

	config.rancher.AddService(client.Service{
		Name:      "service1",
		AccountId: "1a5",
		Resource:  client.Resource{Id: "3a1"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{
			"icinga.service_vars": "myvar=hello,anothervar=boing"}}})

	err = sync(config)
	assert.Nil(err)

	services, err = config.icinga.ListServices()

	assert.Nil(err, "listing the services should not cause an error")

	assert.NotEmpty(services, "the list of services should not be empty")
	assert.Equal(1, len(services), "we should have exactly one service")

	service = services[0]

	assertBasicServiceVars(assert, service)

	assert.Equal("hello", service.Vars["myvar"], "the var should be unchanged")
	assert.Equal("boing", service.Vars["anothervar"], "the var should be updated")
	assert.Nil(service.Vars["somevar"], "the var should be removed")
}

// Test: I can set default and local services vars and they will appear at the service level in icinga.
// The local value will overwrite the global value.
// And both can be changed at runtime.
func TestServiceDefaultAndLocalIcingaVars(t *testing.T) {

	assert := assert.New(t)
	config := initForTests()

	config.serviceDefaultIcingaVars = icinga2.Vars{
		"var1": "globalvalue1",
		"var2": "globalvalue2",
		"var3": "globalvalue3"}

	config.rancher.AddEnvironment(client.Project{Name: "Default", Resource: client.Resource{Id: "1a5"}})
	config.rancher.AddStack(client.Stack{Name: "mystack", AccountId: "1a5", ServiceIds: []string{"3a1"}})
	config.rancher.AddService(client.Service{
		Name:      "service1",
		AccountId: "1a5",
		Resource:  client.Resource{Id: "3a1"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{
			"icinga.service_vars": "var2=localvalue2,var4=localvalue4"}}})

	err := sync(config)
	assert.Nil(err)

	services, err := config.icinga.ListServices()

	assert.Nil(err, "listing the services should not cause an error")

	assert.NotEmpty(services, "the list of services should not be empty")
	assert.Equal(1, len(services), "we should have exactly one service")

	service := services[0]

	assertBasicServiceVars(assert, service)

	assert.Equal("globalvalue1", service.Vars["var1"], "the first var should be set from the global configuration")
	assert.Equal("localvalue2", service.Vars["var2"], "the second var should be overridden from local")
	assert.Equal("globalvalue3", service.Vars["var3"], "the third var should be set from the global configuration")
	assert.Equal("localvalue4", service.Vars["var4"], "the fourth var should be set from the global configuration")

	config.serviceDefaultIcingaVars = icinga2.Vars{
		"var1": "newglobalvalue1",
		"var2": "globalvalue2",
		"var5": "globalvalue5"}

	config.rancher.AddService(client.Service{
		Name:      "service1",
		AccountId: "1a5",
		Resource:  client.Resource{Id: "3a1"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{
			"icinga.service_vars": "var4=newlocalvalue4,var6=localvalue6"}}})

	err = sync(config)
	assert.Nil(err)

	services, err = config.icinga.ListServices()

	assert.Nil(err, "listing the services should not cause an error")

	assert.NotEmpty(services, "the list of services should not be empty")
	assert.Equal(1, len(services), "we should have exactly one service")

	service = services[0]

	assertBasicServiceVars(assert, service)

	assert.Equal("newglobalvalue1", service.Vars["var1"], "the first var should be new from the global configuration")
	assert.Equal("globalvalue2", service.Vars["var2"], "the second var should no longer be overridden from local")
	assert.Nil(service.Vars["var3"], "the third var should be removed")
	assert.Equal("newlocalvalue4", service.Vars["var4"], "the fourth var should be updated from the local value")
	assert.Equal("globalvalue5", service.Vars["var5"], "the fifth var should be new from the global configuration")
	assert.Equal("localvalue6", service.Vars["var6"], "the sixth var should be new from the local value")

}

// Test a full installation
func TestEverything(t *testing.T) {

	assert := assert.New(t)
	config := initForTests()

	config.filterStacks = "*,-%HAS_SERVICE(monitor=false)"
	config.filterHosts = "*,-monitor=false"
	config.filterServices = "*,-monitor=false"

	tmpl, err := template.New("stackname").Parse(`mysite.rancher.{{.RancherEnvironment}}.{{.RancherStack}}`)
	assert.Nil(err)

	config.stackNameTemplate = tmpl

	config.rancher.AddEnvironment(client.Project{Name: "test", Resource: client.Resource{Id: "1a5"}})
	config.rancher.AddEnvironment(client.Project{Name: "prod", Resource: client.Resource{Id: "1b5"}})

	config.rancher.AddHost(client.Host{Hostname: "test1", AccountId: "1a5", Resource: client.Resource{Id: "2a1"}})
	config.rancher.AddHost(client.Host{Hostname: "test2", AccountId: "1a5", Resource: client.Resource{Id: "2a2"}})
	config.rancher.AddHost(client.Host{Hostname: "test3", AccountId: "1a5", Resource: client.Resource{Id: "2a3"}})

	config.rancher.AddHost(client.Host{Hostname: "prod1", AccountId: "1b5", Resource: client.Resource{Id: "2b1"}})
	config.rancher.AddHost(client.Host{Hostname: "prod2", AccountId: "1b5", Resource: client.Resource{Id: "2b2"}})
	config.rancher.AddHost(client.Host{Hostname: "prod3", AccountId: "1b5", Resource: client.Resource{Id: "2b3"}})
	config.rancher.AddHost(client.Host{Hostname: "prod4", AccountId: "1b5", Resource: client.Resource{Id: "2b4"}})
	config.rancher.AddHost(client.Host{Hostname: "prod5", AccountId: "1b5", Resource: client.Resource{Id: "2b5"},
		Labels: map[string]interface{}{"monitor": "false"}})

	// Test

	config.rancher.AddStack(client.Stack{
		Name:       "myapp",
		AccountId:  "1a5",
		Resource:   client.Resource{Id: "3a1"},
		ServiceIds: []string{"4a1", "4a2", "4a3"}})

	config.rancher.AddStack(client.Stack{
		Name:       "testing",
		AccountId:  "1a5",
		Resource:   client.Resource{Id: "3a2"},
		ServiceIds: []string{"4a4", "4a5"}})

	config.rancher.AddService(client.Service{
		Name:      "frontend",
		AccountId: "1a5",
		StackId:   "3a1",
		Resource:  client.Resource{Id: "4a1"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{
			"icinga.stack_notes_url":   "http://docs.mysite.com/myapp/myapp.html",
			"icinga.service_notes_url": "http://docs.mysite.com/myapp/frontend.html"}}})

	config.rancher.AddService(client.Service{
		Name:      "backend",
		AccountId: "1a5",
		StackId:   "3a1",
		Resource:  client.Resource{Id: "4a2"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{
			"icinga.service_notes_url": "http://docs.mysite.com/myapp/backend.html"}}})

	config.rancher.AddService(client.Service{
		Name:      "database",
		AccountId: "1a5",
		StackId:   "3a1",
		Resource:  client.Resource{Id: "4a3"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{
			"icinga.service_notes_url": "http://docs.mysite.com/myapp/database.html"}}})

	config.rancher.AddService(client.Service{
		Name:         "frontend",
		AccountId:    "1a5",
		StackId:      "3a2",
		Resource:     client.Resource{Id: "4a4"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{"monitor": "false"}}})

	config.rancher.AddService(client.Service{
		Name:         "backend",
		AccountId:    "1a5",
		StackId:      "3a2",
		Resource:     client.Resource{Id: "4a5"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{"monitor": "false"}}})

	// Prod

	config.rancher.AddStack(client.Stack{
		Name:       "myapp",
		AccountId:  "1b5",
		Resource:   client.Resource{Id: "3b1"},
		ServiceIds: []string{"4b1", "4b2", "4b3"}})

	config.rancher.AddStack(client.Stack{
		Name:       "anotherapp",
		AccountId:  "1b5",
		Resource:   client.Resource{Id: "3b2"},
		ServiceIds: []string{"4b4", "4b5"}})

	config.rancher.AddService(client.Service{
		Name:      "frontend",
		AccountId: "1b5",
		StackId:   "3b1",
		Resource:  client.Resource{Id: "4b1"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{
			"icinga.stack_notes_url":   "http://docs.mysite.com/myapp/myapp.html",
			"icinga.service_notes_url": "http://docs.mysite.com/myapp/frontend.html"}}})

	config.rancher.AddService(client.Service{
		Name:      "backend",
		AccountId: "1b5",
		StackId:   "3b1",
		Resource:  client.Resource{Id: "4b2"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{
			"icinga.service_notes_url": "http://docs.mysite.com/myapp/backend.html"}}})

	config.rancher.AddService(client.Service{
		Name:      "database",
		AccountId: "1b5",
		StackId:   "3b1",
		Resource:  client.Resource{Id: "4b3"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{
			"icinga.service_notes_url": "http://docs.mysite.com/myapp/database.html"}}})

	config.rancher.AddService(client.Service{
		Name:      "frontend",
		AccountId: "1b5",
		StackId:   "3b2",
		Resource:  client.Resource{Id: "4b4"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{
			"icinga.stack_vars":   "do_not_break=please",
			"icinga.service_vars": "do_not_break=please"}}})

	config.rancher.AddService(client.Service{
		Name:         "backend",
		AccountId:    "1b5",
		StackId:      "3b2",
		Resource:     client.Resource{Id: "4b5"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{"icinga.service_vars": "do_not_break=please"}}})

	err = sync(config)
	assert.Nil(err)

	assertEverything(assert, config)

	// If we sync again, it should still work.

	err = sync(config)
	assert.Nil(err)

	assertEverything(assert, config)

}

func assertEverything(assert *assert.Assertions, config *RancherIcingaConfig) {

	hostGroups, err := config.icinga.ListHostGroups()

	assert.Nil(err)
	assert.Equal(2, len(hostGroups))

	hosts, err := config.icinga.ListHosts()

	assert.Nil(err)
	assert.Equal(10, len(hosts)) // 8 agents (1 filtered), 4 stacks (1 filtered)

	services, err := config.icinga.ListServices()

	assert.Nil(err)
	assert.Equal(15, len(services)) // 8 agents (1 filtered), 10 services (2 filtered)

	expectAgents := map[string]bool{
		"test1": false,
		"test2": false,
		"test3": false,
		"prod1": false,
		"prod2": false,
		"prod3": false,
		"prod4": false,
	}

	expectStacks := map[string]bool{
		"mysite.rancher.test.myapp":      false,
		"mysite.rancher.prod.myapp":      false,
		"mysite.rancher.prod.anotherapp": false,
	}

	expectServices := map[string]bool{
		"mysite.rancher.test.myapp!frontend":      false,
		"mysite.rancher.test.myapp!backend":       false,
		"mysite.rancher.test.myapp!database":      false,
		"mysite.rancher.prod.myapp!frontend":      false,
		"mysite.rancher.prod.myapp!backend":       false,
		"mysite.rancher.prod.myapp!database":      false,
		"mysite.rancher.prod.anotherapp!frontend": false,
		"mysite.rancher.prod.anotherapp!backend":  false,
	}

	for _, host := range hosts {
		if host.Vars["rancher_object_type"] == "host" {
			if _, ok := expectAgents[host.Name]; ok {
				expectAgents[host.Name] = true
				assert.Equal("hostalive", host.CheckCommand)
			} else {
				assert.Fail("Unexpected host " + host.Name)
			}
		} else if host.Vars["rancher_object_type"] == "stack" {
			if _, ok := expectStacks[host.Name]; ok {
				expectStacks[host.Name] = true
				assert.Equal("check_rancher_stack", host.CheckCommand)

				switch host.Vars["rancher_stack"] {
				case "myapp":
					assert.Equal("http://docs.mysite.com/myapp/myapp.html", host.NotesURL)
				case "anotherapp":
					assert.Equal("please", host.Vars["do_not_break"])
					assert.Empty(host.NotesURL)
				default:
					assert.Empty(host.NotesURL)
					assert.Empty(host.Vars["do_not_break"])

				}

			} else {
				assert.Fail("Unexpected host " + host.Name)
			}
		} else {
			assert.Fail("Expected host " + host.Name + " to be either a host or a stack")
		}
	}

	for _, service := range services {
		if service.Vars["rancher_object_type"] == "service" {
			fullName := service.HostName + "!" + service.Name
			if _, ok := expectServices[fullName]; ok {
				expectServices[fullName] = true

				switch service.Vars["rancher_stack"] {
				case "myapp":
					switch service.Vars["rancher_service"] {
					case "frontend":
						assert.Equal("http://docs.mysite.com/myapp/frontend.html", service.NotesURL)
					case "backend":
						assert.Equal("http://docs.mysite.com/myapp/backend.html", service.NotesURL)
					case "database":
						assert.Equal("http://docs.mysite.com/myapp/database.html", service.NotesURL)
					default:
						assert.Empty(service.NotesURL)
					}
				case "anotherapp":
					assert.Equal("please", service.Vars["do_not_break"])
				default:
					assert.Fail("Unexpected service " + service.Name)
				}
			}
		} else if service.Vars["rancher_object_type"] == "rancher-agent" {
			if _, ok := expectAgents[service.HostName]; !ok {
				assert.Fail("Unexpected rancher-agent service on host " + service.HostName)
			}
		} else {
			assert.Fail("Expected service " + service.Name + " to be either a service or an rancher-agent service")
		}
	}

	for agent, found := range expectAgents {
		assert.True(found, "expected to find agent host "+agent)
	}

	for stack, found := range expectStacks {
		assert.True(found, "expected to find stack "+stack)
	}

	for service, found := range expectServices {
		assert.True(found, "expected to find service "+service)
	}

}

func TestCustomCheck(t *testing.T) {
	assert := assert.New(t)
	config := initForTests()

	config.rancher.AddEnvironment(client.Project{Name: "Default", Resource: client.Resource{Id: "1a5"}})
	config.rancher.AddStack(client.Stack{
		Name:       "mystack",
		AccountId:  "1a5",
		Resource:   client.Resource{Id: "2a1"},
		ServiceIds: []string{"3a1"}})
	config.rancher.AddService(client.Service{
		Name:      "service1",
		AccountId: "1a5",
		Resource:  client.Resource{Id: "3a1"},
		StackId:   "2a1",
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{
			"icinga.custom_checks": `- name: check1
  command: http
  notes_url: http://docs.mysite.com/check1.html
  vars:
    http_address: service1.mystack.rancher.internal
    http_port: 80
    http_uri: /health
- name: check2
  command: http
  notes_url: http://docs.mysite.com/check2.html
  vars:
    http_address: www.mysite.com
    http_port: 80
    http_uri: /health`}}})

	err := sync(config)
	assert.Nil(err)

	services, err := config.icinga.ListServices()

	assert.Nil(err, "listing the services should not cause an error")

	assert.NotEmpty(services, "the list of services should not be empty")
	assert.Equal(3, len(services), "we should have 3 monitored services")

	var foundService1, foundService2, foundCustom1, foundCustom2, foundCustom3 bool

	for _, service := range services {
		assert.Equal("mystack", service.Vars["rancher_stack"])
		switch service.Name {
		case "service1":
			foundService1 = true
			assert.Equal("check_rancher_service", service.CheckCommand)
			assert.Equal("service1", service.Vars["rancher_service"])
		case "check1":
			foundCustom1 = true
			assert.Equal("http", service.CheckCommand)
			assert.Equal("service1.mystack.rancher.internal", service.Vars["http_address"])
			assert.Equal("80", service.Vars["http_port"]) // yes, a string
			assert.Equal("service1", service.Vars["rancher_service"])
		case "check2":
			foundCustom2 = true
			assert.Equal("http", service.CheckCommand)
			assert.Equal("www.mysite.com", service.Vars["http_address"])
			assert.Equal("80", service.Vars["http_port"])
			assert.Equal("service1", service.Vars["rancher_service"])
		default:
			assert.Fail("Got an unexpected service name " + service.Name)
		}

	}

	assert.True(foundService1 && foundCustom1 && foundCustom2, "we did not find all 3 expected service checks")

	// Add a second service

	config.rancher.AddStack(client.Stack{
		Name:       "mystack",
		AccountId:  "1a5",
		Resource:   client.Resource{Id: "2a1"},
		ServiceIds: []string{"3a1", "3a2"}})
	config.rancher.AddService(client.Service{
		Name:      "service2",
		AccountId: "1a5",
		StackId:   "2a1",
		Resource:  client.Resource{Id: "3a2"},
		LaunchConfig: &client.LaunchConfig{Labels: map[string]interface{}{
			"icinga.custom_checks": `- name: check3
  command: http
  notes_url: http://docs.mysite.com/check3.html
  vars:
    http_address: service2.mystack.rancher.internal
    http_port: 80
    http_uri: /health`}}})

	err = sync(config)
	assert.Nil(err)

	services, err = config.icinga.ListServices()

	assert.Nil(err, "listing the services should not cause an error")

	assert.NotEmpty(services, "the list of services should not be empty")
	assert.Equal(5, len(services), "we should have 5 monitored services")

	foundService1, foundService2, foundCustom1, foundCustom2, foundCustom3 = false, false, false, false, false

	for _, service := range services {
		assert.Equal("mystack", service.Vars["rancher_stack"])
		switch service.Name {
		case "service1":
			foundService1 = true
			assert.Equal("check_rancher_service", service.CheckCommand)
			assert.Equal("service1", service.Vars["rancher_service"])
		case "service2":
			foundService2 = true
			assert.Equal("check_rancher_service", service.CheckCommand)
			assert.Equal("service2", service.Vars["rancher_service"])
		case "check1":
			foundCustom1 = true
			assert.Equal("http", service.CheckCommand)
			assert.Equal("service1.mystack.rancher.internal", service.Vars["http_address"])
			assert.Equal("80", service.Vars["http_port"])
			assert.Equal("service1", service.Vars["rancher_service"])
		case "check2":
			foundCustom2 = true
			assert.Equal("http", service.CheckCommand)
			assert.Equal("www.mysite.com", service.Vars["http_address"])
			assert.Equal("80", service.Vars["http_port"])
			assert.Equal("service1", service.Vars["rancher_service"])
		case "check3":
			foundCustom3 = true
			assert.Equal("http", service.CheckCommand)
			assert.Equal("service2.mystack.rancher.internal", service.Vars["http_address"])
			assert.Equal("80", service.Vars["http_port"])
			assert.Equal("service2", service.Vars["rancher_service"])
		default:
			assert.Fail("Got an unexpected service name " + service.Name)
		}
	}

	assert.True(foundService1 && foundService2 && foundCustom1 && foundCustom2 && foundCustom3,
		"we did not find all 5 expected service checks")

	// Remove the first service

	config.rancher.AddStack(client.Stack{
		Name:       "mystack",
		AccountId:  "1a5",
		Resource:   client.Resource{Id: "2a1"},
		ServiceIds: []string{"3a2"}})
	config.rancher.DeleteService("3a1")

	err = sync(config)
	assert.Nil(err)

	services, err = config.icinga.ListServices()

	assert.Nil(err, "listing the services should not cause an error")

	assert.NotEmpty(services, "the list of services should not be empty")
	assert.Equal(2, len(services), "we should have 2 monitored services")

	foundService1, foundService2, foundCustom1, foundCustom2, foundCustom3 = false, false, false, false, false

	for _, service := range services {
		assert.Equal("mystack", service.Vars["rancher_stack"])
		switch service.Name {
		case "service2":
			foundService2 = true
			assert.Equal("check_rancher_service", service.CheckCommand)
			assert.Equal("service2", service.Vars["rancher_service"])
		case "check3":
			foundCustom3 = true
			assert.Equal("http", service.CheckCommand)
			assert.Equal("service2.mystack.rancher.internal", service.Vars["http_address"])
			assert.Equal("80", service.Vars["http_port"])
			assert.Equal("service2", service.Vars["rancher_service"])
		default:
			assert.Fail("Got an unexpected service name " + service.Name)
		}
	}

	assert.True(foundService2 && foundCustom3,
		"we did not find all 2 expected service checks")

}
