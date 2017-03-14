package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gobwas/glob"
	"github.com/rancher/go-rancher/v2"
)

func filterEnvironment(rancher *RancherContext, env client.Project, filter string) (match bool) {
	match = false
	re := regexp.MustCompile("^([+-]?)([a-zA-Z0-9\\.=\\*%-]*)(!L)?$")
	for _, r := range strings.Split(filter, ",") {
		ruleParts := re.FindStringSubmatch(r)
		if ruleParts == nil {
			panic("failed to match " + r)
		}
		ie, rule, mod := ruleParts[1], ruleParts[2], ruleParts[3]

		var ifMatched bool
		if ie == "" || ie == "+" {
			ifMatched = true
		} else if ie == "-" {
			ifMatched = false
		}

		if r == "" {
			match = true
		} else if glob.MustCompile(rule).Match(env.Name) {
			match = ifMatched
			if mod == "!L" {
				return
			}
		}
	}
	return
}

func filterHost(rancher *RancherContext, host client.Host, filter string) (match bool) {
	match = false
	re := regexp.MustCompile("^([+-]?)([a-zA-Z0-9\\.=\\*%-]*)(!L)?$")
	for _, r := range strings.Split(filter, ",") {
		ruleParts := re.FindStringSubmatch(r)
		if ruleParts == nil {
			panic("failed to match " + r)
		}
		ie, rule, mod := ruleParts[1], ruleParts[2], ruleParts[3]

		var ifMatched bool
		if ie == "" || ie == "+" {
			ifMatched = true
		} else if ie == "-" {
			ifMatched = false
		}

		if r == "" {
			match = true
		} else if m := regexp.MustCompile("^%ENV=([a-zA-Z\\.\\*]+)$").FindStringSubmatch(rule); m != nil {
			if glob.MustCompile(m[1]).Match(rancher.GetEnvironment(host.AccountId).Name) {
				match = ifMatched
				if mod == "!L" {
					return
				}
			}
		} else if m := regexp.MustCompile("^([a-zA-Z\\.\\*]+)=([a-zA-Z\\.\\*]+)$").FindStringSubmatch(rule); m != nil {
			for l, v := range host.Labels {
				if glob.MustCompile(rule).Match(fmt.Sprintf("%s=%s", l, v)) {
					match = ifMatched
					if mod == "!L" {
						return
					}
				}
			}
		} else if glob.MustCompile(rule).Match(host.Hostname) {
			match = ifMatched
			if mod == "!L" {
				return
			}
		}
	}
	return
}

func filterStack(rancher *RancherContext, stack client.Stack, filter string) (match bool) {
	match = false
	re := regexp.MustCompile("^([+-]?)([a-zA-Z0-9\\.=\\_*%\\(\\)-]*)(!L)?$")
	for _, r := range strings.Split(filter, ",") {
		ruleParts := re.FindStringSubmatch(r)
		if ruleParts == nil {
			panic("failed to match " + r)
		}
		ie, rule, mod := ruleParts[1], ruleParts[2], ruleParts[3]

		var ifMatched bool
		if ie == "" || ie == "+" {
			ifMatched = true
		} else if ie == "-" {
			ifMatched = false
		}

		if r == "" {
			match = true
		} else if m := regexp.MustCompile("^%ENV=([a-zA-Z\\.\\*]+)$").FindStringSubmatch(rule); m != nil {
			if glob.MustCompile(m[1]).Match(rancher.GetEnvironment(stack.AccountId).Name) {
				match = ifMatched
				if mod == "!L" {
					return
				}
			}
		} else if rule == "%SYSTEM" {
			if stack.System {
				match = ifMatched
				if mod == "!L" {
					return
				}
			}
		} else if m := regexp.MustCompile("^%HAS_SERVICE\\(([a-zA-Z0-9\\.\\*_-]+)\\)$").FindStringSubmatch(rule); m != nil {
			for _, s := range stack.ServiceIds {
				if glob.MustCompile(m[1]).Match(rancher.GetService(s).Name) {
					match = ifMatched
					if mod == "!L" {
						return
					}
				}
			}
		} else if m := regexp.MustCompile("^%HAS_SERVICE\\(([a-zA-Z0-9\\.\\*_-]+)=([a-zA-Z0-9\\.\\*_-]+)\\)$").FindStringSubmatch(rule); m != nil {
			for _, s := range stack.ServiceIds {
				service := rancher.GetService(s)
				for l, v := range service.LaunchConfig.Labels {
					if glob.MustCompile(fmt.Sprintf("%s=%s", m[1], m[2])).Match(fmt.Sprintf("%s=%s", l, v)) {
						match = ifMatched
						if mod == "!L" {
							return
						}
					}
				}
			}
		} else if glob.MustCompile(rule).Match(stack.Name) {
			match = ifMatched
			if mod == "!L" {
				return
			}
		}
	}
	return
}

func filterService(rancher *RancherContext, service client.Service, filter string) (match bool) {
	match = false
	re := regexp.MustCompile("^([+-]?)([a-zA-Z0-9\\.=\\_*%\\(\\)-]*)(!L)?$")
	for _, r := range strings.Split(filter, ",") {
		ruleParts := re.FindStringSubmatch(r)
		if ruleParts == nil {
			panic("failed to match " + r)
		}
		ie, rule, mod := ruleParts[1], ruleParts[2], ruleParts[3]

		var ifMatched bool
		if ie == "" || ie == "+" {
			ifMatched = true
		} else if ie == "-" {
			ifMatched = false
		}

		if r == "" {
			match = true
		} else if m := regexp.MustCompile("^%ENV=([a-zA-Z\\.\\*]+)$").FindStringSubmatch(rule); m != nil {
			if glob.MustCompile(m[1]).Match(rancher.GetEnvironment(service.AccountId).Name) {
				match = ifMatched
				if mod == "!L" {
					return
				}

			}
		} else if rule == "%SYSTEM" {
			if service.System {
				match = ifMatched
				if mod == "!L" {
					return
				}
			}
		} else if m := regexp.MustCompile("^%STACK=([a-zA-Z\\.\\*]+)$").FindStringSubmatch(rule); m != nil {
			if glob.MustCompile(m[1]).Match(rancher.GetStack(service.StackId).Name) {
				match = ifMatched
				if mod == "!L" {
					return
				}
			}
		} else if m := regexp.MustCompile("^([a-zA-Z\\.\\*]+)=([a-zA-Z\\.\\*]+)$").FindStringSubmatch(rule); m != nil {
			for l, v := range service.LaunchConfig.Labels {
				if glob.MustCompile(rule).Match(fmt.Sprintf("%s=%s", l, v)) {
					match = ifMatched
					if mod == "!L" {
						return
					}
				}
			}
		} else if glob.MustCompile(rule).Match(service.Name) {
			match = ifMatched
			if mod == "!L" {
				return
			}
		}
	}
	return
}
