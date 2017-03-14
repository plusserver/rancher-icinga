// TODO: Deal with environments we do not have access to, but whose stacks/services show up in the API.

package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/Nexinto/go-icinga2-client/icinga2"
	"github.com/rancher/go-rancher/v2"
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

func main() {

	// The names of the inciga2 Vars as constants.
	RANCHER_INSTALLATION := "rancher_installation"
	RANCHER_ENVIRONMENT := "rancher_environment"
	RANCHER_ACCESS_KEY := "rancher_access_key"
	RANCHER_SECRET_KEY := "rancher_secret_key"
	RANCHER_URL := "rancher_url"
	RANCHER_STACK := "rancher_stack"
	RANCHER_SERVICE := "rancher_service"
	RANCHER_OBJECT_TYPE := "rancher_object_type"

	hostCheckCommand := "hostalive"
	//	serverCheckCommand := "hostalive"
	stackCheckCommand := "check_rancher_stack"
	serviceCheckCommand := "check_rancher_service"

	rancherInstallation := "default"

	filterEnvironments := ""
	filterHosts := ""
	filterStacks := ""
	filterServices := ""

	icingaDefaultVars := map[string]interface{}{}

	//	if c := os.Getenv("SERVER_CHECK_COMMAND"); c != "" {
	//		serverCheckCommand = c
	//	}
	if c := os.Getenv("HOST_CHECK_COMMAND"); c != "" {
		hostCheckCommand = c
	}
	if c := os.Getenv("STACK_CHECK_COMMAND"); c != "" {
		stackCheckCommand = c
	}
	if c := os.Getenv("SERVICE_CHECK_COMMAND"); c != "" {
		serviceCheckCommand = c
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

	if os.Getenv("ICINGA_DEBUG") != "" {
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
			if !filterEnvironment(rancher, env, filterEnvironments) {
				continue
			}

			found := false
			for _, hg := range hostGroups {
				if hg.Vars[RANCHER_INSTALLATION] == rancherInstallation &&
					hg.Vars[RANCHER_OBJECT_TYPE] == "environment" &&
					hg.Vars[RANCHER_ENVIRONMENT] == env.Name {
					found = true
					continue
				}
			}
			if found == false {
				ic.CreateHostGroup(icinga2.HostGroup{
					Name: execTemplate(environmentNameTemplate, "", env.Name, "", ""),
					Vars: mergeMaps(icingaDefaultVars, map[string]interface{}{
						RANCHER_INSTALLATION: rancherInstallation,
						RANCHER_OBJECT_TYPE:  "environment",
						RANCHER_ENVIRONMENT:  env.Name})})
			}
		}

		for _, hg := range hostGroups {
			if hg.Vars[RANCHER_INSTALLATION] != rancherInstallation {
				continue // not created by rancher-icinga
			}
			found := false
			for _, env := range environments.Data {
				if filterEnvironment(rancher, env, filterEnvironments) &&
					hg.Vars[RANCHER_OBJECT_TYPE] == "environment" &&
					hg.Vars[RANCHER_ENVIRONMENT] == env.Name {
					found = true
					continue
				}
			}
			if found == false {
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
			if !filterHost(rancher, rh, filterHosts) {
				continue
			}

			found := false
			environmentName := rancher.GetEnvironment(rh.AccountId).Name

			for _, ih := range icingaHosts {
				if rh.Hostname == ih.Name &&
					ih.Vars[RANCHER_INSTALLATION] == rancherInstallation &&
					ih.Vars[RANCHER_OBJECT_TYPE] == "host" &&
					ih.Vars[RANCHER_ENVIRONMENT] == environmentName {
					found = true
					continue
				}
			}
			if found == false {
				ic.CreateHost(icinga2.Host{
					Name:         rh.Hostname,
					DisplayName:  rh.Hostname,
					Address:      rh.AgentIpAddress,
					Groups:       []string{environmentName},
					CheckCommand: hostCheckCommand,
					Vars: mergeMaps(icingaDefaultVars, map[string]interface{}{
						RANCHER_INSTALLATION: rancherInstallation,
						RANCHER_OBJECT_TYPE:  "host",
						RANCHER_ENVIRONMENT:  environmentName,
					})})
			}

		}

		for _, ih := range icingaHosts {
			if ih.Vars[RANCHER_INSTALLATION] != rancherInstallation ||
				ih.Vars[RANCHER_OBJECT_TYPE] != "host" {
				continue // not created by rancher-icinga
			}
			found := false
			for _, rh := range rancherHosts.Data {
				if filterHost(rancher, rh, filterHosts) &&
					ih.Name == rh.Hostname &&
					ih.Vars[RANCHER_INSTALLATION] == rancherInstallation &&
					ih.Vars[RANCHER_OBJECT_TYPE] == "host" &&
					ih.Vars[RANCHER_ENVIRONMENT] == rancher.GetEnvironment(rh.AccountId).Name {
					found = true
					//continue
				}
			}
			if found == false {
				ic.DeleteHost(ih.Name)
			}
		}

		// Synchronize stacks as hosts

		stacks, err := rancher.Stacks()
		if err != nil {
			panic(err)
		}

		for _, s := range stacks.Data {
			if !filterStack(rancher, s, filterStacks) {
				continue
			}

			found := false
			environmentName := rancher.GetEnvironment(s.AccountId).Name
			for _, ih := range icingaHosts {
				if ih.Vars[RANCHER_INSTALLATION] == rancherInstallation &&
					ih.Vars[RANCHER_OBJECT_TYPE] == "stack" &&
					ih.Vars[RANCHER_ENVIRONMENT] == environmentName &&
					ih.Vars[RANCHER_STACK] == s.Name {
					found = true
				}
			}
			if found == false {
				ic.CreateHost(icinga2.Host{
					Name:         execTemplate(stackNameTemplate, "", environmentName, s.Name, ""),
					DisplayName:  execTemplate(stackNameTemplate, "", environmentName, s.Name, ""),
					Groups:       []string{environmentName},
					CheckCommand: stackCheckCommand,
					Vars: mergeMaps(icingaDefaultVars, map[string]interface{}{
						RANCHER_INSTALLATION: rancherInstallation,
						RANCHER_OBJECT_TYPE:  "stack",
						RANCHER_ENVIRONMENT:  environmentName,
						RANCHER_STACK:        s.Name,
					})})

			}
		}

		for _, ih := range icingaHosts {
			if ih.Vars[RANCHER_INSTALLATION] != rancherInstallation ||
				ih.Vars[RANCHER_OBJECT_TYPE] != "stack" {
				continue // not created by rancher-icinga
			}
			found := false
			for _, s := range stacks.Data {
				environmentName := rancher.GetEnvironment(s.AccountId).Name
				if filterStack(rancher, s, filterStacks) &&
					ih.Vars[RANCHER_ENVIRONMENT] == environmentName &&
					ih.Vars[RANCHER_STACK] == s.Name {
					found = true
				}
			}
			if found == false {
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
			if !filterService(rancher, rs, filterServices) {
				continue
			}

			if !filterStack(rancher, rancher.GetStack(rs.StackId), filterStacks) {
				continue
			}

			found := false
			stackName := rancher.GetStack(rs.StackId).Name
			environmentName := rancher.GetEnvironment(rs.AccountId).Name
			for _, is := range icingaServices {
				if is.Vars[RANCHER_INSTALLATION] == rancherInstallation &&
					is.Vars[RANCHER_OBJECT_TYPE] == "service" &&
					is.Vars[RANCHER_STACK] == stackName &&
					is.Vars[RANCHER_SERVICE] == rs.Name &&
					is.Vars[RANCHER_ENVIRONMENT] == environmentName {
					found = true
				}
			}
			if found == false {
				ic.CreateService(icinga2.Service{
					Name:         rs.Name,
					HostName:     execTemplate(stackNameTemplate, "", environmentName, stackName, rs.Name),
					CheckCommand: serviceCheckCommand,
					Vars: mergeMaps(icingaDefaultVars, map[string]interface{}{
						RANCHER_INSTALLATION: rancherInstallation,
						RANCHER_OBJECT_TYPE:  "service",
						RANCHER_STACK:        stackName,
						RANCHER_SERVICE:      rs.Name,
						RANCHER_ENVIRONMENT:  environmentName,
					})})
			}
		}

		for _, is := range icingaServices {
			if is.Vars[RANCHER_INSTALLATION] != rancherInstallation ||
				is.Vars[RANCHER_OBJECT_TYPE] != "service" {
				continue // not created by rancher-icinga
			}
			found := false
			for _, rs := range rancherServices.Data {
				stackName := rancher.GetStack(rs.StackId).Name
				environmentName := rancher.GetEnvironment(rs.AccountId).Name
				if filterService(rancher, rs, filterServices) &&
					filterStack(rancher, rancher.GetStack(rs.StackId), filterStacks) &&
					is.Vars[RANCHER_STACK] == stackName &&
					is.Vars[RANCHER_SERVICE] == rs.Name &&
					is.Vars[RANCHER_ENVIRONMENT] == environmentName {
					found = true
				}
			}
			if found == false {
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
