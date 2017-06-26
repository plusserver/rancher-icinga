// TODO: Deal with environments we do not have access to, but whose stacks/services show up in the API.

package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/Nexinto/go-icinga2-client/icinga2"
	"github.com/rancher/go-rancher/v2"

	"gopkg.in/jmcvetta/napping.v3"
)

// The names of the inciga2 Vars as constants.
const RANCHER_INSTALLATION = "rancher_installation"
const RANCHER_ENVIRONMENT = "rancher_environment"
const RANCHER_ACCESS_KEY = "rancher_access_key"
const RANCHER_SECRET_KEY = "rancher_secret_key"
const RANCHER_URL = "rancher_url"
const RANCHER_STACK = "rancher_stack"
const RANCHER_SERVICE = "rancher_service"
const RANCHER_HOST = "rancher_host"
const RANCHER_OBJECT_TYPE = "rancher_object_type"

const HOST_NOTES_URL_LABEL = "icinga.host_notes_url"
const STACK_NOTES_URL_LABEL = "icinga.stack_notes_url"
const SERVICE_NOTES_URL_LABEL = "icinga.service_notes_url"

const HOST_VARS_LABEL = "icinga.host_vars"
const STACK_VARS_LABEL = "icinga.stack_vars"
const SERVICE_VARS_LABEL = "icinga.service_vars"

type RancherCheckParameters struct {
	Hostname           string
	RancherUrl         string
	RancherAccessKey   string
	RancherSecretKey   string
	RancherEnvironment string
	RancherStack       string
	RancherService     string
}

type IcingaEvent struct {
	Operation  string       `json:"operation"`
	Name       string       `json:"name"`
	IcingaType string       `json:"type"`
	Vars       icinga2.Vars `json:"vars"`
	Object     interface{}  `json:"object"`
}

type RancherIcingaConfig struct {
	hostCheckCommand         string
	stackCheckCommand        string
	serviceCheckCommand      string
	agentServiceCheckCommand string

	rancherInstallation string

	filterEnvironments string
	filterHosts        string
	filterStacks       string
	filterServices     string

	hostgroupDefaultIcingaVars icinga2.Vars
	hostDefaultIcingaVars      icinga2.Vars
	stackDefaultIcingaVars     icinga2.Vars
	serviceDefaultIcingaVars   icinga2.Vars

	refreshInterval int

	debugMode, insecureTLS bool

	environmentNameTemplate *template.Template
	stackNameTemplate       *template.Template

	icinga  icinga2.Client
	rancher RancherGenClient
}

func NewBaseConfig() (cc *RancherIcingaConfig, err error) {
	cc = new(RancherIcingaConfig)

	if c := os.Getenv("HOST_CHECK_COMMAND"); c != "" {
		cc.hostCheckCommand = c
	} else {
		cc.hostCheckCommand = "hostalive"
	}
	if c := os.Getenv("STACK_CHECK_COMMAND"); c != "" {
		cc.stackCheckCommand = c
	} else {
		cc.stackCheckCommand = "check_rancher_stack"
	}
	if c := os.Getenv("SERVICE_CHECK_COMMAND"); c != "" {
		cc.serviceCheckCommand = c
	} else {
		cc.serviceCheckCommand = "check_rancher_service"
	}
	if c := os.Getenv("AGENT_SERVICE_CHECK_COMMAND"); c != "" {
		cc.agentServiceCheckCommand = c
	} else {
		cc.agentServiceCheckCommand = "check_rancher_host"
	}

	if c := os.Getenv("RANCHER_INSTALLATION"); c != "" {
		cc.rancherInstallation = c
	} else {
		cc.rancherInstallation = "default"
	}

	if c := os.Getenv("FILTER_ENVIRONMENTS"); c != "" {
		cc.filterEnvironments = c
	}
	if c := os.Getenv("FILTER_HOSTS"); c != "" {
		cc.filterHosts = c
	}
	if c := os.Getenv("FILTER_STACKS"); c != "" {
		cc.filterStacks = c
	}
	if c := os.Getenv("FILTER_SERVICES"); c != "" {
		cc.filterServices = c
	}

	if c := os.Getenv("HOSTGROUP_DEFAULT_ICINGA_VARS"); c != "" {
		cc.hostgroupDefaultIcingaVars = unpackVars(c)
	} else {
		cc.hostgroupDefaultIcingaVars = make(icinga2.Vars)
	}
	if c := os.Getenv("HOST_DEFAULT_ICINGA_VARS"); c != "" {
		cc.hostDefaultIcingaVars = unpackVars(c)
	} else {
		cc.hostDefaultIcingaVars = make(icinga2.Vars)
	}
	if c := os.Getenv("STACK_DEFAULT_ICINGA_VARS"); c != "" {
		cc.stackDefaultIcingaVars = unpackVars(c)
	} else {
		cc.stackDefaultIcingaVars = make(icinga2.Vars)
	}
	if c := os.Getenv("SERVICE_DEFAULT_ICINGA_VARS"); c != "" {
		cc.serviceDefaultIcingaVars = unpackVars(c)
	} else {
		cc.serviceDefaultIcingaVars = make(icinga2.Vars)
	}

	cc.hostgroupDefaultIcingaVars[RANCHER_URL] = os.Getenv("RANCHER_URL")
	cc.hostgroupDefaultIcingaVars[RANCHER_ACCESS_KEY] = os.Getenv("RANCHER_ACCESS_KEY")
	cc.hostgroupDefaultIcingaVars[RANCHER_SECRET_KEY] = os.Getenv("RANCHER_SECRET_KEY")

	cc.hostDefaultIcingaVars[RANCHER_URL] = os.Getenv("RANCHER_URL")
	cc.hostDefaultIcingaVars[RANCHER_ACCESS_KEY] = os.Getenv("RANCHER_ACCESS_KEY")
	cc.hostDefaultIcingaVars[RANCHER_SECRET_KEY] = os.Getenv("RANCHER_SECRET_KEY")

	cc.stackDefaultIcingaVars[RANCHER_URL] = os.Getenv("RANCHER_URL")
	cc.stackDefaultIcingaVars[RANCHER_ACCESS_KEY] = os.Getenv("RANCHER_ACCESS_KEY")
	cc.stackDefaultIcingaVars[RANCHER_SECRET_KEY] = os.Getenv("RANCHER_SECRET_KEY")

	cc.serviceDefaultIcingaVars[RANCHER_URL] = os.Getenv("RANCHER_URL")
	cc.serviceDefaultIcingaVars[RANCHER_ACCESS_KEY] = os.Getenv("RANCHER_ACCESS_KEY")
	cc.serviceDefaultIcingaVars[RANCHER_SECRET_KEY] = os.Getenv("RANCHER_SECRET_KEY")

	if c := os.Getenv("REFRESH_INTERVAL"); c != "" {
		fmt.Sscanf(c, "%d", &cc.refreshInterval)
	} else {
		cc.refreshInterval = 0
	}

	if os.Getenv("ICINGA_DEBUG") == "3" {
		cc.debugMode = true
	} else {
		cc.debugMode = false
	}
	if os.Getenv("ICINGA_INSECURE_TLS") != "" {
		cc.insecureTLS = true
	} else {
		cc.insecureTLS = false
	}

	cc.environmentNameTemplate, cc.stackNameTemplate, err = makeTemplates()

	if err != nil {
		panic(err)
	}

	return
}

func NewConfig() (cc *RancherIcingaConfig, err error) {

	cc, err = NewBaseConfig()

	if err != nil {
		panic(err)
	}

	rancherClient, err := client.NewRancherClient(&client.ClientOpts{
		Url:       os.Getenv("RANCHER_URL"),
		AccessKey: os.Getenv("RANCHER_ACCESS_KEY"),
		SecretKey: os.Getenv("RANCHER_SECRET_KEY"),
		Timeout:   10 * time.Second})

	if err != nil {
		panic(err)
	}

	cc.rancher = NewRancherWebClient(rancherClient)

	cc.icinga, err = icinga2.New(icinga2.WebClient{
		URL:         os.Getenv("ICINGA_URL"),
		Username:    os.Getenv("ICINGA_USER"),
		Password:    os.Getenv("ICINGA_PASSWORD"),
		Debug:       cc.debugMode,
		InsecureTLS: cc.insecureTLS})

	if err != nil {
		panic(err)
	}

	return
}

func debugLog(s string, l int) {

	// TODO: learn how to log properly
	if l == 1 && os.Getenv("ICINGA_DEBUG") != "" ||
		l == 2 && (os.Getenv("ICINGA_DEBUG") == "2" || os.Getenv("ICINGA_DEBUG") == "3") {
		fmt.Println(s)
	}
}

func syncRancherEnvironments(config *RancherIcingaConfig) {
	environments, err := config.rancher.Environments()
	if err != nil {
		panic(err)
	}

	hostGroups, err := config.icinga.ListHostGroups()
	if err != nil {
		panic(err)
	}

	for _, env := range environments.Data {
		debugLog("Syncing environment "+env.Name, 2)
		if !filterEnvironment(config.rancher, env, config.filterEnvironments) {
			debugLog("  disabled by filter", 2)
			continue
		}

		found := false
		for _, hg := range hostGroups {
			debugLog("  Checking hostgroup "+hg.Name, 2)
			if config.matches(hg.Vars, "environment", env.Name, "", "") {
				debugLog("    found", 2)
				found = true
				continue
			}
		}
		if found == false {
			name := execTemplate(config.environmentNameTemplate, "", env.Name, "", "")
			vars := varsForEnvironment(config, env)
			config.icinga.CreateHostGroup(icinga2.HostGroup{Name: name, Vars: vars})
			debugLog("Creating host group "+name+" for environment", 1)
			registerChange("create", name, "hostgroup", vars, icinga2.HostGroup{Name: name, Vars: vars})
		}
	}

}

func syncIcingaHostgroups(config *RancherIcingaConfig) {
	environments, err := config.rancher.Environments()
	if err != nil {
		panic(err)
	}

	hostGroups, err := config.icinga.ListHostGroups()
	if err != nil {
		panic(err)
	}

	for _, hg := range hostGroups {
		debugLog("Syncing hostgroup "+hg.Name, 2)
		if !config.matches(hg.Vars, "environment", "", "", "") {
			debugLog("  skipping, was not created for our rancher installation", 2)
			continue // not created by rancher-icinga
		}
		found := false
		for _, env := range environments.Data {
			debugLog("  Checking environment "+env.Name, 2)
			if filterEnvironment(config.rancher, env, config.filterEnvironments) &&
				config.matches(hg.Vars, "environment", env.Name, "", "") {
				debugLog("    found", 2)
				found = true
				continue
			}
		}
		if found == false {
			registerChange("delete", hg.Name, "hostgroup", icinga2.Vars{}, hg)
			debugLog("Remove hostgroup "+hg.Name+" for environment", 1)
			defer config.icinga.DeleteHostGroup(hg.Name)
		}
	}

}

func syncRancherHosts(config *RancherIcingaConfig) {
	rancherHosts, err := config.rancher.Hosts()
	if err != nil {
		panic(err)
	}

	icingaHosts, err := config.icinga.ListHosts()
	if err != nil {
		panic(err)
	}

	icingaServices, err := config.icinga.ListServices()
	if err != nil {
		panic(err)
	}

	for _, rh := range rancherHosts.Data {
		debugLog("Syncing host "+rh.Hostname, 2)
		if !filterHost(config.rancher, rh, config.filterHosts) {
			debugLog("  disabled by filter", 2)
			continue
		}

		found := false
		environmentName := config.rancher.GetEnvironment(rh.AccountId).Name

		var notesURL string

		if n, ok := rh.Labels[HOST_NOTES_URL_LABEL].(string); ok {
			notesURL = n
		}

		for _, ih := range icingaHosts {
			debugLog("  Checking icinga host "+ih.Name, 2)
			if config.matches(ih.Vars, "host", environmentName, "", "") && rh.Hostname == ih.Name {
				debugLog("    found", 2)
				found = true

				needUpdate := false

				if notesURL != ih.NotesURL {
					debugLog("Updating host "+ih.Name+" with notes_url "+notesURL, 1)
					ih.NotesURL = notesURL
					needUpdate = true
				}

				newVars := varsForHost(config, rh, environmentName)

				if varsNeedUpdate(newVars, ih.Vars) {
					ih.Vars = newVars
					needUpdate = true
				}

				if needUpdate {
					debugLog("    update "+ih.Name, 1)
					config.icinga.UpdateHost(ih)
					registerChange("update", ih.Name, "host", ih.Vars, ih)
				}
			}
		}
		if found == false {
			vars := varsForHost(config, rh, environmentName)
			ih := icinga2.Host{
				Name:         rh.Hostname,
				DisplayName:  rh.Hostname,
				Address:      rh.AgentIpAddress,
				Groups:       []string{environmentName},
				CheckCommand: config.hostCheckCommand,
				NotesURL:     notesURL,
				Vars:         vars}
			config.icinga.CreateHost(ih)
			debugLog("Creating rancher agent host "+rh.Hostname, 1)
			registerChange("create", rh.Hostname, "host", vars, ih)
		}

		// Create a rancher-agent service for each agent host

		found = false

		for _, is := range icingaServices {
			debugLog("  Checking service "+is.Name, 2)
			if config.matches(is.Vars, "rancher-agent", environmentName, "", "") &&
				rh.Hostname == is.HostName &&
				is.Vars[RANCHER_HOST] == rh.Hostname {
				debugLog("    found", 2)
				found = true

				needUpdate := false

				if notesURL != is.NotesURL {
					debugLog("Updating rancher agent service "+is.Name+" with notes_url "+notesURL, 1)
					is.NotesURL = notesURL
					needUpdate = true
				}

				if needUpdate {
					debugLog("    update "+is.Name, 1)
					config.icinga.UpdateService(is)
					registerChange("update", is.Name, "service", is.Vars, is)
				}
			}
		}

		if found == false {
			vars := varsForAgentService(config, rh.Hostname, environmentName)

			debugLog("Creating agent service check for host "+rh.Hostname, 1)
			is := icinga2.Service{
				Name:         rh.Hostname + "!rancher-agent",
				HostName:     rh.Hostname,
				CheckCommand: config.agentServiceCheckCommand,
				NotesURL:     notesURL,
				Vars:         vars}
			config.icinga.CreateService(is)
			registerChange("create", is.Name, "service", vars, is)
		}
	}
}

func syncIcingaHosts(config *RancherIcingaConfig) {
	rancherHosts, err := config.rancher.Hosts()
	if err != nil {
		panic(err)
	}

	icingaHosts, err := config.icinga.ListHosts()
	if err != nil {
		panic(err)
	}

	for _, ih := range icingaHosts {
		debugLog("Syncing icinga host "+ih.Name, 2)
		if !config.matches(ih.Vars, "host", "", "", "") {
			debugLog("  skipping, type or installation do not match", 2)
			continue // wrong type or not created by rancher-icinga
		}
		found := false
		for _, rh := range rancherHosts.Data {
			if filterHost(config.rancher, rh, config.filterHosts) &&
				ih.Name == rh.Hostname &&
				ih.Vars[RANCHER_INSTALLATION] == config.rancherInstallation &&
				ih.Vars[RANCHER_OBJECT_TYPE] == "host" &&
				ih.Vars[RANCHER_ENVIRONMENT] == config.rancher.GetEnvironment(rh.AccountId).Name {
				debugLog("    found", 2)
				found = true
			}
		}
		if found == false {
			registerChange("delete-cascade", ih.Name, "host", icinga2.Vars{}, ih)
			debugLog("Removing rancher agent host "+ih.Name, 1)
			defer config.icinga.DeleteHost(ih.Name)
		}
	}
}

func syncRancherStacks(config *RancherIcingaConfig) {
	stacks, err := config.rancher.Stacks()
	if err != nil {
		panic(err)
	}

	icingaHosts, err := config.icinga.ListHosts()
	if err != nil {
		panic(err)
	}

	for _, s := range stacks.Data {
		environmentName := config.rancher.GetEnvironment(s.AccountId).Name
		debugLog("Syncing stack ["+environmentName+"] "+s.Name, 2)
		if !filterStack(config.rancher, s, config.filterStacks) {
			debugLog("  disabled by filter", 2)
			continue
		}

		notesURL := ""
		services, err := config.servicesOf(s)
		if err != nil {
			panic(err)
		}

		for _, service := range services.Data {
			if l, ok := service.LaunchConfig.Labels[STACK_NOTES_URL_LABEL].(string); ok {
				notesURL = l
			}
		}

		found := false
		for _, ih := range icingaHosts {
			debugLog("  Checking icinga host "+ih.Name, 2)
			if config.matches(ih.Vars, "stack", environmentName, s.Name, "") {
				debugLog("    found", 2)
				found = true

				needUpdate := false

				if notesURL != ih.NotesURL {
					debugLog("Updating host "+ih.Name+" with notes_url "+notesURL+" original is "+ih.NotesURL, 1)
					ih.NotesURL = notesURL
					needUpdate = true
				}

				newVars := varsForStack(config, s, environmentName)

				if varsNeedUpdate(newVars, ih.Vars) {
					ih.Vars = newVars
					needUpdate = true
				}

				if needUpdate {
					debugLog("    update "+ih.Name, 1)
					config.icinga.UpdateHost(ih)
					registerChange("update", ih.Name, "host", ih.Vars, ih)
				}

			}
		}
		if found == false {
			name := execTemplate(config.stackNameTemplate, "", environmentName, s.Name, "")
			vars := varsForStack(config, s, environmentName)
			ih := icinga2.Host{
				Name:         name,
				DisplayName:  name,
				Groups:       []string{environmentName},
				CheckCommand: config.stackCheckCommand,
				NotesURL:     notesURL,
				Vars:         vars}
			config.icinga.CreateHost(ih)
			debugLog("Creating host "+name+" for stack "+s.Name, 1)
			registerChange("create", name, "host", vars, ih)
		}

	}

	for _, ih := range icingaHosts {
		debugLog("Syncing icinga host "+ih.Name, 2)
		if ih.Vars[RANCHER_INSTALLATION] != config.rancherInstallation ||
			ih.Vars[RANCHER_OBJECT_TYPE] != "stack" {
			debugLog("  skipping, type or installation do not match", 2)
			continue // not created by rancher-icinga
		}
		found := false
		for _, s := range stacks.Data {
			environmentName := config.rancher.GetEnvironment(s.AccountId).Name
			debugLog("  Checking stack ["+environmentName+"] "+s.Name, 2)
			if filterStack(config.rancher, s, config.filterStacks) &&
				ih.Vars[RANCHER_ENVIRONMENT] == environmentName &&
				ih.Vars[RANCHER_STACK] == s.Name {
				debugLog("    found", 2)
				found = true
			}
		}
		if found == false {
			debugLog("Removing host "+ih.Name+" for stack", 1)
			registerChange("delete-cascade", ih.Name, "host", icinga2.Vars{}, ih)
			config.icinga.DeleteHost(ih.Name)
		}
	}

}

func syncRancherServices(config *RancherIcingaConfig) {
	rancherServices, err := config.rancher.Services()
	if err != nil {
		panic(err)
	}

	icingaServices, err := config.icinga.ListServices()
	if err != nil {
		panic(err)
	}

	for _, rs := range rancherServices.Data {
		stackName := config.rancher.GetStack(rs.StackId).Name
		environmentName := config.rancher.GetEnvironment(rs.AccountId).Name

		debugLog("Syncing service ["+environmentName+"] "+stackName+"/"+rs.Name, 2)

		if !filterService(config.rancher, rs, config.filterServices) {
			debugLog("  service disabled by filter", 2)
			continue
		}

		if !filterStack(config.rancher, config.rancher.GetStack(rs.StackId), config.filterStacks) {
			debugLog("  stack disabled by filter", 2)
			continue
		}

		found := false

		for _, is := range icingaServices {
			debugLog("  Checking icinga service "+is.Name, 2)
			if config.matches(is.Vars, "service", environmentName, stackName, rs.Name) {
				debugLog("    found", 2)
				found = true

				needUpdate := false

				if notesURL, ok := rs.LaunchConfig.Labels[SERVICE_NOTES_URL_LABEL].(string); ok {
					if notesURL != is.NotesURL {
						debugLog("Updating service "+is.Name+" with notes_url "+notesURL, 1)
						is.NotesURL = notesURL
						needUpdate = true
					}
				}

				newVars := varsForService(config, rs, environmentName, stackName)

				if varsNeedUpdate(newVars, is.Vars) {
					is.Vars = newVars
					needUpdate = true
				}

				if needUpdate {
					debugLog("    update "+is.Name, 1)
					config.icinga.UpdateService(is)
					registerChange("update", is.Name, "service", is.Vars, is)
				}
			}
		}
		if found == false {
			notesURL, _ := rs.LaunchConfig.Labels[SERVICE_NOTES_URL_LABEL].(string)
			vars := varsForService(config, rs, environmentName, stackName)
			hostname := execTemplate(config.stackNameTemplate, "", environmentName, stackName, rs.Name)
			is := icinga2.Service{
				Name:         hostname + "!" + rs.Name,
				HostName:     hostname,
				CheckCommand: config.serviceCheckCommand,
				NotesURL:     notesURL,
				Vars:         vars}
			config.icinga.CreateService(is)
			debugLog("Creating service "+is.Name+" for service "+stackName+"/"+rs.Name, 1)
			registerChange("create", is.Name, "service", vars, is)
		}
	}

}

func syncIcingaServices(config *RancherIcingaConfig) {
	rancherServices, err := config.rancher.Services()
	if err != nil {
		panic(err)
	}

	icingaServices, err := config.icinga.ListServices()
	if err != nil {
		panic(err)
	}

	rancherHosts, err := config.rancher.Hosts()
	if err != nil {
		panic(err)
	}

	for _, is := range icingaServices {
		debugLog("Syncing icinga service "+is.Name, 2)
		if !config.matches(is.Vars, "rancher-agent/service", "", "", "") {
			debugLog("  skipping, type or installation do not match", 2)
			continue // not created by rancher-icinga
		}
		found := false
		for _, rs := range rancherServices.Data {
			stackName := config.rancher.GetStack(rs.StackId).Name
			environmentName := config.rancher.GetEnvironment(rs.AccountId).Name
			debugLog("  Checking service ["+environmentName+"] "+stackName+"/"+rs.Name, 2)
			if config.matches(is.Vars, "service", environmentName, stackName, rs.Name) &&
				filterService(config.rancher, rs, config.filterServices) &&
				filterStack(config.rancher, config.rancher.GetStack(rs.StackId), config.filterStacks) {
				debugLog("    found", 2)
				found = true
			}
		}

		for _, rh := range rancherHosts.Data {
			environmentName := config.rancher.GetEnvironment(rh.AccountId).Name
			debugLog("  Checking host "+rh.Hostname, 2)
			if config.matches(is.Vars, "rancher-agent", environmentName, "", "") &&
				filterHost(config.rancher, rh, config.filterHosts) &&
				is.Vars[RANCHER_HOST] == rh.Hostname &&
				is.HostName == rh.Hostname {
				debugLog("    found", 2)
				found = true
			}
		}

		if found == false {
			debugLog("Removing service "+is.Name, 1)
			registerChange("delete", is.Name, "service", icinga2.Vars{}, is)
			defer config.icinga.DeleteService(is.Name)
		}
	}

}

func sync(config *RancherIcingaConfig) {
	syncRancherEnvironments(config)
	syncIcingaHostgroups(config)

	syncRancherHosts(config)
	syncRancherStacks(config)
	syncRancherServices(config)

	syncIcingaHosts(config)
	syncIcingaServices(config)
}

func main() {

	config, err := NewConfig()

	if err != nil {
		panic(err)
	}

	for {
		fmt.Printf("Refreshing at %s\n", time.Now().Local())
		sync(config)

		if config.refreshInterval <= 0 {
			break
		} else {
			time.Sleep(time.Duration(config.refreshInterval) * time.Second)
		}
	}

}

func execTemplate(t *template.Template, hostname string, environment string, stack string, service string) string {
	checkParams := RancherCheckParameters{
		Hostname:           hostname,
		RancherUrl:         os.Getenv("RANCHER_URL"),
		RancherAccessKey:   os.Getenv("RANCHER_ACCESS_KEY"),
		RancherSecretKey:   os.Getenv("RANCHER_SECRET_KEY"),
		RancherEnvironment: environment,
		RancherStack:       stack,
		RancherService:     service,
	}

	var buffer bytes.Buffer

	err := t.Execute(&buffer, checkParams)
	if err != nil {
		panic(err)
	}

	return buffer.String()
}

func makeTemplates() (environmentNameTemplate *template.Template, stackNameTemplate *template.Template, err error) {
	environmentName := os.Getenv("ENVIRONMENT_NAME_TEMPLATE")
	if len(environmentName) == 0 {
		environmentName = "{{.RancherEnvironment}}"
	}
	environmentNameTemplate, err = template.New("environmentname").Parse(environmentName)
	if err != nil {
		err = fmt.Errorf("Failed to parse environment name template: %q", err.Error())
		return
	}

	stackName := os.Getenv("STACK_NAME_TEMPLATE")
	if len(stackName) == 0 {
		stackName = "{{.RancherEnvironment}}.{{.RancherStack}}"
	}
	stackNameTemplate, err = template.New("stackname").Parse(stackName)
	if err != nil {
		err = fmt.Errorf("Failed to parse stack name template: %q", err.Error())
		return
	}

	return
}

func mergeVars(a icinga2.Vars, b icinga2.Vars) (r icinga2.Vars) {
	r = make(icinga2.Vars)
	for k, v := range a {
		r[k] = v
	}
	for k, v := range b {
		r[k] = v
	}
	return
}

func registerChange(operation string, name string, icingatype string, vars icinga2.Vars, object interface{}) {
	if url := os.Getenv("REGISTER_CHANGES"); url != "" {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: transport}

		naps := napping.Session{
			Client: client,
			Header: &http.Header{"Content-Type": []string{"application/json"}},
		}

		ev := IcingaEvent{Operation: operation, Name: name, IcingaType: icingatype, Vars: vars, Object: object}

		resp, err := naps.Post(url, ev, nil, nil)
		if err != nil {
			fmt.Println("error sending change event", err)
		} else if resp.HttpResponse().StatusCode >= 400 {
			body, _ := ioutil.ReadAll(io.LimitReader(resp.HttpResponse().Body, 1048576))
			fmt.Printf("%s %s\n", resp.HttpResponse().Status, body)
		}
	}
}

// unpacks a list of icinga variables from a string (command separated key=value pairs)
func unpackVars(input string) (res icinga2.Vars) {
	res = make(icinga2.Vars)
	for _, p := range strings.Split(input, ",") {
		a := strings.Split(p, "=")
		if len(a) == 2 {
			res[a[0]] = a[1]
		}
	}
	return
}

// Checks if the vars of an icinga object matches with the configured rancher installation, the current
// and environment and is the correct object type.
// If the type is "rancher-agent/service" or "stack/host" it will match both types as those icinga
// object types are used for different rancher object types.
func (config *RancherIcingaConfig) matches(vars icinga2.Vars, typ, env, stack, service string) bool {
	var matchesInst, matchesType, matchesEnvironment, matchesStack, matchesService bool

	if vars[RANCHER_INSTALLATION] == config.rancherInstallation {
		matchesInst = true
	} else {
		matchesInst = false
	}

	if typ == "" {
		matchesType = true
	} else if vars[RANCHER_OBJECT_TYPE] == typ {
		matchesType = true
	} else if typ == "rancher-agent/service" && (vars[RANCHER_OBJECT_TYPE] == "service" || vars[RANCHER_OBJECT_TYPE] == "rancher-agent") {
		matchesType = true
	} else if typ == "host/stack" && (vars[RANCHER_OBJECT_TYPE] == "host" || vars[RANCHER_OBJECT_TYPE] == "stack") {
		matchesType = true
	} else {
		matchesType = false
	}

	if env == "" {
		matchesEnvironment = true
	} else if vars[RANCHER_ENVIRONMENT] == env {
		matchesEnvironment = true
	} else {
		matchesEnvironment = false
	}

	if stack == "" {
		matchesStack = true
	} else if vars[RANCHER_STACK] == stack {
		matchesStack = true
	} else {
		matchesStack = false
	}

	if service == "" {
		matchesService = true
	} else if vars[RANCHER_SERVICE] == service {
		matchesService = true
	} else {
		matchesService = false
	}

	return matchesInst && matchesType && matchesEnvironment && matchesStack && matchesService
}

// Returns true if an icinga object's vars need updating.
func varsNeedUpdate(newVars icinga2.Vars, vars icinga2.Vars) bool {
	for k, v := range newVars {
		if vars[k] != v {
			return true
		}
	}

	for k, v := range vars {
		if newVars[k] != v {
			return true
		}
	}

	return false
}

// Generates the vars for a hostgroup
func varsForEnvironment(config *RancherIcingaConfig, environment client.Project) icinga2.Vars {
	return mergeVars(config.hostgroupDefaultIcingaVars, icinga2.Vars{
		RANCHER_INSTALLATION: config.rancherInstallation,
		RANCHER_OBJECT_TYPE:  "environment",
		RANCHER_ENVIRONMENT:  environment.Name})
}

// Generates the vars for a rancher host
func varsForHost(config *RancherIcingaConfig, host client.Host, environment string) (vars icinga2.Vars) {
	vars = mergeVars(config.hostDefaultIcingaVars, icinga2.Vars{
		RANCHER_INSTALLATION: config.rancherInstallation,
		RANCHER_OBJECT_TYPE:  "host",
		RANCHER_ENVIRONMENT:  environment,
		RANCHER_HOST:         host.Hostname})

	labels := host.Labels

	if labels[HOST_VARS_LABEL] != nil && labels[HOST_VARS_LABEL] != "" {
		vars = mergeVars(vars, unpackVars(labels[HOST_VARS_LABEL].(string)))
	}

	return
}

// Generates the vars for the service that describes a rancher agent
func varsForAgentService(config *RancherIcingaConfig, hostname, environment string) (vars icinga2.Vars) {
	vars = mergeVars(config.serviceDefaultIcingaVars, icinga2.Vars{
		RANCHER_INSTALLATION: config.rancherInstallation,
		RANCHER_OBJECT_TYPE:  "rancher-agent",
		RANCHER_ENVIRONMENT:  environment})

	if hostname != "" {
		vars[RANCHER_HOST] = hostname
	}

	return
}

// Generates the vars for a stack
func varsForStack(config *RancherIcingaConfig, stack client.Stack, environment string) (vars icinga2.Vars) {
	vars = mergeVars(config.stackDefaultIcingaVars, icinga2.Vars{
		RANCHER_INSTALLATION: config.rancherInstallation,
		RANCHER_OBJECT_TYPE:  "stack",
		RANCHER_ENVIRONMENT:  environment,
		RANCHER_STACK:        stack.Name})

	services, err := config.servicesOf(stack)
	if err != nil {
		panic(err)
	}

	for _, service := range services.Data {
		labels := service.LaunchConfig.Labels

		if labels[STACK_VARS_LABEL] != nil && labels[STACK_VARS_LABEL] != "" {
			vars = mergeVars(vars, unpackVars(labels[STACK_VARS_LABEL].(string)))
		}
	}

	return
}

// Generates the vars for a service
func varsForService(config *RancherIcingaConfig, service client.Service, environment, stack string) (vars icinga2.Vars) {
	vars = mergeVars(config.serviceDefaultIcingaVars, icinga2.Vars{
		RANCHER_INSTALLATION: config.rancherInstallation,
		RANCHER_OBJECT_TYPE:  "service",
		RANCHER_SERVICE:      service.Name})

	labels := service.LaunchConfig.Labels

	if labels[SERVICE_VARS_LABEL] != nil && labels[SERVICE_VARS_LABEL] != "" {
		vars = mergeVars(vars, unpackVars(labels[SERVICE_VARS_LABEL].(string)))
	}

	if environment != "" {
		vars[RANCHER_ENVIRONMENT] = environment
	}
	if stack != "" {
		vars[RANCHER_STACK] = stack
	}

	return
}

// Find the services for a stack
func (config *RancherIcingaConfig) servicesOf(stack client.Stack) (*client.ServiceCollection, error) {
	coll := make([]client.Service, 0, len(stack.ServiceIds))

	for _, id := range stack.ServiceIds {
		coll = append(coll, config.rancher.GetService(id))
	}

	return &client.ServiceCollection{Data: coll}, nil
}
