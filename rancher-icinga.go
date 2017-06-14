// TODO: Deal with environments we do not have access to, but whose stacks/services show up in the API.

package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/Nexinto/go-icinga2-client/icinga2"
	"github.com/rancher/go-rancher/v2"

	"gopkg.in/jmcvetta/napping.v3"
)

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
	Operation  string                 `json:"operation"`
	Name       string                 `json:"name"`
	IcingaType string                 `json:"type"`
	Vars       map[string]interface{} `json:"vars"`
}

func debugLog(s string, l int) {

	// TODO: learn how to log properly
	if l == 1 && os.Getenv("ICINGA_DEBUG") != "" ||
		l == 2 && (os.Getenv("ICINGA_DEBUG") == "2" || os.Getenv("ICINGA_DEBUG") == "3") {
		fmt.Println(s)
	}
}

func main() {

	// The names of the inciga2 Vars as constants.
	RANCHER_INSTALLATION := "rancher_installation"
	RANCHER_ENVIRONMENT := "rancher_environment"
	RANCHER_ACCESS_KEY := "rancher_access_key"
	RANCHER_SECRET_KEY := "rancher_secret_key"
	RANCHER_URL := "rancher_url"
	RANCHER_STACK := "rancher_stack"
	RANCHER_SERVICE := "rancher_service"
	RANCHER_HOST := "rancher_host"
	RANCHER_OBJECT_TYPE := "rancher_object_type"

	hostCheckCommand := "hostalive"
	stackCheckCommand := "check_rancher_stack"
	serviceCheckCommand := "check_rancher_service"
	agentServiceCheckCommand := "check_rancher_host"

	rancherInstallation := "default"

	filterEnvironments := ""
	filterHosts := ""
	filterStacks := ""
	filterServices := ""

	icingaDefaultVars := map[string]interface{}{}

	if c := os.Getenv("HOST_CHECK_COMMAND"); c != "" {
		hostCheckCommand = c
	}
	if c := os.Getenv("STACK_CHECK_COMMAND"); c != "" {
		stackCheckCommand = c
	}
	if c := os.Getenv("SERVICE_CHECK_COMMAND"); c != "" {
		serviceCheckCommand = c
	}
	if c := os.Getenv("AGENT_SERVICE_CHECK_COMMAND"); c != "" {
		agentServiceCheckCommand = c
	}

	if c := os.Getenv("RANCHER_INSTALLATION"); c != "" {
		rancherInstallation = c
	}

	if c := os.Getenv("FILTER_ENVIRONMENTS"); c != "" {
		filterHosts = c
	}
	if c := os.Getenv("FILTER_HOSTS"); c != "" {
		filterHosts = c
	}
	if c := os.Getenv("FILTER_STACKS"); c != "" {
		filterStacks = c
	}
	if c := os.Getenv("FILTER_SERVICES"); c != "" {
		filterServices = c
	}

	if c := os.Getenv("ICINGA_DEFAULT_VARS"); c != "" {
		for _, p := range strings.Split(c, ",") {
			a := strings.Split(p, "=")
			if len(a) == 2 {
				icingaDefaultVars[a[0]] = a[1]
			}
		}
	}

	icingaDefaultVars[RANCHER_URL] = os.Getenv("RANCHER_URL")
	icingaDefaultVars[RANCHER_ACCESS_KEY] = os.Getenv("RANCHER_ACCESS_KEY")
	icingaDefaultVars[RANCHER_SECRET_KEY] = os.Getenv("RANCHER_SECRET_KEY")

	environmentNameTemplate, stackNameTemplate, err := makeTemplates()
	if err != nil {
		panic(err)
	}

	var refreshInterval int
	if c := os.Getenv("REFRESH_INTERVAL"); c != "" {
		fmt.Sscanf(c, "%d", &refreshInterval)
	} else {
		refreshInterval = 0
	}

	var debugMode, insecureTLS bool

	if os.Getenv("ICINGA_DEBUG") == "3" {
		debugMode = true
	} else {
		debugMode = false
	}
	if os.Getenv("ICINGA_INSECURE_TLS") != "" {
		insecureTLS = true
	} else {
		insecureTLS = false
	}

	rancherC, err := client.NewRancherClient(&client.ClientOpts{
		Url:       os.Getenv("RANCHER_URL"),
		AccessKey: os.Getenv("RANCHER_ACCESS_KEY"),
		SecretKey: os.Getenv("RANCHER_SECRET_KEY"),
		Timeout:   10 * time.Second})
	if err != nil {
		panic(err)
	}

	rancher := NewRancherContext(rancherC)

	ic, err := icinga2.New(icinga2.Server{
		URL:         os.Getenv("ICINGA_URL"),
		Username:    os.Getenv("ICINGA_USER"),
		Password:    os.Getenv("ICINGA_PASSWORD"),
		Debug:       debugMode,
		InsecureTLS: insecureTLS})
	if err != nil {
		panic(err)
	}

	for {
		fmt.Printf("Refreshing at %s\n", time.Now().Local())

		// Synchronize environments with host groups

		environments, err := rancher.Environments()
		if err != nil {
			panic(err)
		}

		hostGroups, err := ic.ListHostGroups()
		if err != nil {
			panic(err)
		}

		for _, env := range environments.Data {
			debugLog("Syncing environment "+env.Name, 2)
			if !filterEnvironment(rancher, env, filterEnvironments) {
				debugLog("  disabled by filter", 2)
				continue
			}

			found := false
			for _, hg := range hostGroups {
				debugLog("  Checking hostgroup "+hg.Name, 2)
				if hg.Vars[RANCHER_INSTALLATION] == rancherInstallation &&
					hg.Vars[RANCHER_OBJECT_TYPE] == "environment" &&
					hg.Vars[RANCHER_ENVIRONMENT] == env.Name {
					debugLog("    found", 2)
					found = true
					continue
				}
			}
			if found == false {
				name := execTemplate(environmentNameTemplate, "", env.Name, "", "")
				vars := mergeMaps(icingaDefaultVars, map[string]interface{}{
					RANCHER_INSTALLATION: rancherInstallation,
					RANCHER_OBJECT_TYPE:  "environment",
					RANCHER_ENVIRONMENT:  env.Name})
				ic.CreateHostGroup(icinga2.HostGroup{Name: name, Vars: vars})
				debugLog("Create host group "+name+" for environment", 1)
				registerChange("create", name, "hostgroup", vars)
			}
		}

		for _, hg := range hostGroups {
			debugLog("Syncing hostgroup "+hg.Name, 2)
			if hg.Vars[RANCHER_INSTALLATION] != rancherInstallation {
				debugLog("  skipping, was not created for our rancher installation", 2)
				continue // not created by rancher-icinga
			}
			found := false
			for _, env := range environments.Data {
				debugLog("  Checking environment "+env.Name, 2)
				if filterEnvironment(rancher, env, filterEnvironments) &&
					hg.Vars[RANCHER_OBJECT_TYPE] == "environment" &&
					hg.Vars[RANCHER_ENVIRONMENT] == env.Name {
					debugLog("    found", 2)
					found = true
					continue
				}
			}
			if found == false {
				registerChange("delete", hg.Name, "hostgroup", map[string]interface{}{})
				debugLog("Remove hostgroup "+hg.Name+" for environment", 1)
				ic.DeleteHostGroup(hg.Name)
			}
		}

		// Synchronize rancher agents

		rancherHosts, err := rancher.rancher.Host.List(nil)
		if err != nil {
			panic(err)
		}

		icingaHosts, err := ic.ListHosts()
		if err != nil {
			panic(err)
		}

		for _, rh := range rancherHosts.Data {
			debugLog("Syncing host "+rh.Hostname, 2)
			if !filterHost(rancher, rh, filterHosts) {
				debugLog("  disabled by filter", 2)
				continue
			}

			found := false
			environmentName := rancher.GetEnvironment(rh.AccountId).Name

			for _, ih := range icingaHosts {
				debugLog("  Checking icinga host "+ih.Name, 2)
				if rh.Hostname == ih.Name &&
					ih.Vars[RANCHER_INSTALLATION] == rancherInstallation &&
					ih.Vars[RANCHER_OBJECT_TYPE] == "host" &&
					ih.Vars[RANCHER_ENVIRONMENT] == environmentName {
					debugLog("    found", 2)
					found = true
					continue
				}
			}
			if found == false {
				vars := mergeMaps(icingaDefaultVars, map[string]interface{}{
					RANCHER_INSTALLATION: rancherInstallation,
					RANCHER_OBJECT_TYPE:  "host",
					RANCHER_ENVIRONMENT:  environmentName,
				})
				ic.CreateHost(icinga2.Host{
					Name:         rh.Hostname,
					DisplayName:  rh.Hostname,
					Address:      rh.AgentIpAddress,
					Groups:       []string{environmentName},
					CheckCommand: hostCheckCommand,
					Vars:         vars})
				debugLog("Creating rancher agent host "+rh.Hostname, 1)
				registerChange("create", rh.Hostname, "host", vars)
			}

			// Create a rancher-agent service for each agent host

			icingaServices, err := ic.ListServices()
			if err != nil {
				panic(err)
			}

			for _, is := range icingaServices {
				debugLog("  Checking service "+is.Name, 2)
				if rh.Hostname == is.HostName &&
					is.Vars[RANCHER_INSTALLATION] == rancherInstallation &&
					is.Vars[RANCHER_OBJECT_TYPE] == "agent-service" &&
					is.Vars[RANCHER_HOST] == rh.Hostname {
					debugLog("    found", 2)
					found = true
				}
			}

			if found == false {
				vars := mergeMaps(icingaDefaultVars, map[string]interface{}{
					RANCHER_INSTALLATION: rancherInstallation,
					RANCHER_OBJECT_TYPE:  "agent-service",
					RANCHER_HOST:         rh.Hostname})

				debugLog("Creating agent service check for host "+rh.Hostname, 1)
				ic.CreateService(icinga2.Service{
					Name:         "rancher-agent",
					HostName:     rh.Hostname,
					CheckCommand: agentServiceCheckCommand,
					Vars:         vars})
			}

		}

		for _, ih := range icingaHosts {
			debugLog("Syncing icinga host "+ih.Name, 2)
			if ih.Vars[RANCHER_INSTALLATION] != rancherInstallation ||
				ih.Vars[RANCHER_OBJECT_TYPE] != "host" {
				debugLog("  skipping, type or installation do not match", 2)
				continue // wrong type or not created by rancher-icinga
			}
			found := false
			for _, rh := range rancherHosts.Data {
				if filterHost(rancher, rh, filterHosts) &&
					ih.Name == rh.Hostname &&
					ih.Vars[RANCHER_INSTALLATION] == rancherInstallation &&
					ih.Vars[RANCHER_OBJECT_TYPE] == "host" &&
					ih.Vars[RANCHER_ENVIRONMENT] == rancher.GetEnvironment(rh.AccountId).Name {
					debugLog("    found", 2)
					found = true
				}
			}
			if found == false {
				registerChange("delete-cascade", ih.Name, "host", map[string]interface{}{})
				debugLog("Removing rancher agent host "+ih.Name, 1)
				ic.DeleteHost(ih.Name)
			}
		}

		// Synchronize stacks as hosts

		stacks, err := rancher.Stacks()
		if err != nil {
			panic(err)
		}

		for _, s := range stacks.Data {
			environmentName := rancher.GetEnvironment(s.AccountId).Name
			debugLog("Syncing stack ["+environmentName+"] "+s.Name, 2)
			if !filterStack(rancher, s, filterStacks) {
				debugLog("  disabled by filter", 2)
				continue
			}

			found := false
			for _, ih := range icingaHosts {
				debugLog("  Checking icinga host "+ih.Name, 2)
				if ih.Vars[RANCHER_INSTALLATION] == rancherInstallation &&
					ih.Vars[RANCHER_OBJECT_TYPE] == "stack" &&
					ih.Vars[RANCHER_ENVIRONMENT] == environmentName &&
					ih.Vars[RANCHER_STACK] == s.Name {
					debugLog("    found", 2)
					found = true
				}
			}
			if found == false {
				name := execTemplate(stackNameTemplate, "", environmentName, s.Name, "")
				vars := mergeMaps(icingaDefaultVars, map[string]interface{}{
					RANCHER_INSTALLATION: rancherInstallation,
					RANCHER_OBJECT_TYPE:  "stack",
					RANCHER_ENVIRONMENT:  environmentName,
					RANCHER_STACK:        s.Name,
				})
				ic.CreateHost(icinga2.Host{
					Name:         name,
					DisplayName:  name,
					Groups:       []string{environmentName},
					CheckCommand: stackCheckCommand,
					Vars:         vars})
				debugLog("Creating host "+name+" for stack "+s.Name, 1)
				registerChange("create", name, "host", vars)
			}
		}

		for _, ih := range icingaHosts {
			debugLog("Syncing icinga host "+ih.Name, 2)
			if ih.Vars[RANCHER_INSTALLATION] != rancherInstallation ||
				ih.Vars[RANCHER_OBJECT_TYPE] != "stack" {
				debugLog("  skipping, type or installation do not match", 2)
				continue // not created by rancher-icinga
			}
			found := false
			for _, s := range stacks.Data {
				environmentName := rancher.GetEnvironment(s.AccountId).Name
				debugLog("  Checking stack ["+environmentName+"] "+s.Name, 2)
				if filterStack(rancher, s, filterStacks) &&
					ih.Vars[RANCHER_ENVIRONMENT] == environmentName &&
					ih.Vars[RANCHER_STACK] == s.Name {
					debugLog("    found", 2)
					found = true
				}
			}
			if found == false {
				debugLog("Removing host "+ih.Name+" for stack", 1)
				registerChange("delete-cascade", ih.Name, "host", map[string]interface{}{})
				ic.DeleteHost(ih.Name)
			}
		}

		// Synchronize services

		rancherServices, err := rancher.Services()
		if err != nil {
			panic(err)
		}

		icingaServices, err := ic.ListServices()
		if err != nil {
			panic(err)
		}

		for _, rs := range rancherServices.Data {
			stackName := rancher.GetStack(rs.StackId).Name
			environmentName := rancher.GetEnvironment(rs.AccountId).Name
			debugLog("Syncing service ["+environmentName+"] "+stackName+"/"+rs.Name, 2)
			if !filterService(rancher, rs, filterServices) {
				debugLog("  service disabled by filter", 2)
				continue
			}

			if !filterStack(rancher, rancher.GetStack(rs.StackId), filterStacks) {
				debugLog("  stack disabled by filter", 2)
				continue
			}

			found := false
			for _, is := range icingaServices {
				debugLog("  Checking icinga service " + is.Name, 2)
				if is.Vars[RANCHER_INSTALLATION] == rancherInstallation &&
					is.Vars[RANCHER_OBJECT_TYPE] == "service" &&
					is.Vars[RANCHER_STACK] == stackName &&
					is.Vars[RANCHER_SERVICE] == rs.Name &&
					is.Vars[RANCHER_ENVIRONMENT] == environmentName {
					debugLog("    found", 2)
					found = true

					if notesURL, ok := rs.LaunchConfig.Labels["icinga.notes_url"].(string); ok {
						if notesURL != is.NotesURL {
							debugLog("Updating service "+is.Name+" with notes_url "+notesURL, 1)
							is.NotesURL = notesURL
							ic.UpdateService(is)
						}
					}
				}
			}
			if found == false {
				notesURL, _ := rs.LaunchConfig.Labels["icinga.notes_url"].(string)
				vars := mergeMaps(icingaDefaultVars, map[string]interface{}{
					RANCHER_INSTALLATION: rancherInstallation,
					RANCHER_OBJECT_TYPE:  "service",
					RANCHER_STACK:        stackName,
					RANCHER_SERVICE:      rs.Name,
					RANCHER_ENVIRONMENT:  environmentName,
				})
				hostname := execTemplate(stackNameTemplate, "", environmentName, stackName, rs.Name)
				ic.CreateService(icinga2.Service{
					Name:         rs.Name,
					HostName:     hostname,
					CheckCommand: serviceCheckCommand,
					NotesURL:     notesURL,
					Vars:         vars})
				debugLog("Creating service "+fmt.Sprintf("%s!%s", hostname, rs.Name)+" for service "+stackName+"/"+rs.Name, 1)
				registerChange("create", fmt.Sprintf("%s!%s", hostname, rs.Name), "service", vars)
			}
		}

		for _, is := range icingaServices {
			debugLog("Syncing icinga service "+is.Name, 2)
			if is.Vars[RANCHER_INSTALLATION] != rancherInstallation ||
				is.Vars[RANCHER_OBJECT_TYPE] != "service" {
				debugLog("  skipping, type or installation do not match", 2)
				continue // not created by rancher-icinga
			}
			found := false
			for _, rs := range rancherServices.Data {
				stackName := rancher.GetStack(rs.StackId).Name
				environmentName := rancher.GetEnvironment(rs.AccountId).Name
				debugLog("  Checking service ["+environmentName+"] "+stackName+"/"+rs.Name, 2)
				if filterService(rancher, rs, filterServices) &&
					filterStack(rancher, rancher.GetStack(rs.StackId), filterStacks) &&
					is.Vars[RANCHER_STACK] == stackName &&
					is.Vars[RANCHER_SERVICE] == rs.Name &&
					is.Vars[RANCHER_ENVIRONMENT] == environmentName {
					debugLog("    found", 2)
					found = true
				}
			}
			if found == false {
				debugLog("Removing service "+is.Name, 1)
				registerChange("delete", is.Name, "service", map[string]interface{}{})
				ic.DeleteService(is.Name)
			}
		}

		if refreshInterval <= 0 {
			break
		} else {
			time.Sleep(time.Duration(refreshInterval) * time.Second)
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
		stackName = "[{{.RancherEnvironment}}] {{.RancherStack}}"
	}
	stackNameTemplate, err = template.New("stackname").Parse(stackName)
	if err != nil {
		err = fmt.Errorf("Failed to parse stack name template: %q", err.Error())
		return
	}

	return
}

func mergeMaps(a map[string]interface{}, b map[string]interface{}) (r map[string]interface{}) {
	r = make(map[string]interface{})
	for k, v := range a {
		r[k] = v
	}
	for k, v := range b {
		r[k] = v
	}
	return
}

func registerChange(operation string, name string, icingatype string, vars map[string]interface{}) {
	if url := os.Getenv("REGISTER_CHANGES"); url != "" {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: transport}

		naps := napping.Session{
			Client: client,
		}

		ev := IcingaEvent{Operation: operation, Name: name, IcingaType: icingatype, Vars: vars}

		fmt.Printf("Sending event to %s: %q\n", url, ev)

		_, err := naps.Post(url, ev, nil, nil)
		if err != nil {
			fmt.Println("error sending change event", err)
		}
	}
}
