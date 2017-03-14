package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gobwas/glob"
	"github.com/rancher/go-rancher/v2"
)

func filterEnvironment(rancher *RancherContext, env client.Project, filter string) bool {
	return filterSomething(rancher, env, filter1Environment, filter)
}

func filterHost(rancher *RancherContext, host client.Host, filter string) bool {
	return filterSomething(rancher, host, filter1Host, filter)
}

func filterStack(rancher *RancherContext, stack client.Stack, filter string) bool {
	return filterSomething(rancher, stack, filter1Stack, filter)
}

func filterService(rancher *RancherContext, service client.Service, filter string) bool {
	return filterSomething(rancher, service, filter1Service, filter)
}

func filterSomething(rancher *RancherContext,
	obj interface{},
	filterFunc func(*RancherContext, interface{}, string) bool,
	filter string) (match bool) {

	match = false
	re := regexp.MustCompile("^([+-]?)([a-zA-Z0-9\\.=\\_*%\\(\\)-]*)(!L)?$")
	for _, r := range strings.Split(filter, ",") {
		ruleParts := re.FindStringSubmatch(r)
		if ruleParts == nil {
			panic("failed to match " + r)
		}
		ie, rule, mod := ruleParts[1], ruleParts[2], ruleParts[3]

		if filterFunc(rancher, obj, rule) {
			if ie == "-" {
				match = false
			} else {
				match = true
			}
			if mod == "!L" {
				return
			}
		}
	}
	return
}

func filter1Environment(rancher *RancherContext, obj interface{}, rule string) bool {
	env := obj.(client.Project)

	if rule == "" {
		return true
	} else if glob.MustCompile(rule).Match(env.Name) {
		return true
	}

	return false
}

func filter1Host(rancher *RancherContext, obj interface{}, rule string) bool {
	host := obj.(client.Host)

	if rule == "" {
		return true
	} else if m := regexp.MustCompile("^%ENV=([a-zA-Z\\.\\*]+)$").FindStringSubmatch(rule); m != nil {
		if glob.MustCompile(m[1]).Match(rancher.GetEnvironment(host.AccountId).Name) {
			return true
		}
	} else if m := regexp.MustCompile("^([a-zA-Z\\.\\*]+)=([a-zA-Z\\.\\*]+)$").FindStringSubmatch(rule); m != nil {
		for l, v := range host.Labels {
			if glob.MustCompile(rule).Match(fmt.Sprintf("%s=%s", l, v)) {
				return true
			}
		}
	} else if glob.MustCompile(rule).Match(host.Hostname) {
		return true
	}

	return false
}

func filter1Stack(rancher *RancherContext, obj interface{}, rule string) bool {
	stack := obj.(client.Stack)

	if rule == "" {
		return true
	} else if m := regexp.MustCompile("^%ENV=([a-zA-Z\\.\\*]+)$").FindStringSubmatch(rule); m != nil {
		if glob.MustCompile(m[1]).Match(rancher.GetEnvironment(stack.AccountId).Name) {
			return true
		}
	} else if rule == "%SYSTEM" {
		if stack.System {
			return true
		}
	} else if m := regexp.MustCompile("^%HAS_SERVICE\\(([a-zA-Z0-9\\.\\*_-]+)\\)$").FindStringSubmatch(rule); m != nil {
		for _, s := range stack.ServiceIds {
			if glob.MustCompile(m[1]).Match(rancher.GetService(s).Name) {
				return true
			}
		}
	} else if m := regexp.MustCompile("^%HAS_SERVICE\\(([a-zA-Z0-9\\.\\*_-]+)=([a-zA-Z0-9\\.\\*_-]+)\\)$").FindStringSubmatch(rule); m != nil {
		for _, s := range stack.ServiceIds {
			service := rancher.GetService(s)
			for l, v := range service.LaunchConfig.Labels {
				if glob.MustCompile(fmt.Sprintf("%s=%s", m[1], m[2])).Match(fmt.Sprintf("%s=%s", l, v)) {
					return true
				}
			}
		}
	} else if glob.MustCompile(rule).Match(stack.Name) {
		return true
	}

	return false
}

func filter1Service(rancher *RancherContext, obj interface{}, rule string) bool {
	service := obj.(client.Service)
	if rule == "" {
		return true
	} else if m := regexp.MustCompile("^%ENV=([a-zA-Z\\.\\*]+)$").FindStringSubmatch(rule); m != nil {
		if glob.MustCompile(m[1]).Match(rancher.GetEnvironment(service.AccountId).Name) {
			return true
		}
	} else if rule == "%SYSTEM" {
		if service.System {
			return true
		}
	} else if m := regexp.MustCompile("^%STACK=([a-zA-Z\\.\\*]+)$").FindStringSubmatch(rule); m != nil {
		if glob.MustCompile(m[1]).Match(rancher.GetStack(service.StackId).Name) {
			return true
		}
	} else if m := regexp.MustCompile("^([a-zA-Z\\.\\*]+)=([a-zA-Z\\.\\*]+)$").FindStringSubmatch(rule); m != nil {
		for l, v := range service.LaunchConfig.Labels {
			if glob.MustCompile(rule).Match(fmt.Sprintf("%s=%s", l, v)) {
				return true
			}
		}
	} else if glob.MustCompile(rule).Match(service.Name) {
		return true
	}

	return false
}
