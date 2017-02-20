# rancher-icinga

**rancher-icinga** registers Rancher resources in Icinga2 for monitoring. Right now, each rancher agent, stack and service
is individually monitored. Rancher environments are created as Icinga2 hostsgroups. Rancher agents are monitored
as simple hosts. Rancher stacks are hosts and services are created as services for those hosts.

Requires Rancher 1.2.0 or later.

## Getting started

You will need a running Icinga2 and https://github.com/Nexinto/check-rancher. See examples/rancher.conf for how to
configure the check_rancher commands.

## Configuration

Use the following environment variables to configure rancher-icinga:

Required:

- **RANCHER_URL** Rancher API URL. Uses the v2-beta API.
- **RACHER_ACCESS_KEY** / **RANCHER_SECRET_KEY** API Keys with enough access to monitor all environments.
- **ICINGA_URL** Icinga2 API URL
- **ICINGA_USER** / **ICINGA_PASSWORD** Icinga2 API username, password

Optional:

- **ENVIRONMENT_NAME_TEMPLATE** Go Template for the name of Icinga2 hostgroups that represent Rancher environments (default: `{{.RancherEnvironment}}`)
- **STACK_NAME_TEMPLATE** Go Template for the name of Icinga2 hosts that represent Rancher stacks (default: `[{{.RancherEnvironment}}] {{.RancherStack}}`)
- **HOST_CHECK_COMMAND** Name of the command Icinga2 uses to check the health of hosts (default: hostalive)
- **STACK_CHECK_COMMAND** Name of the check command used to monitor a Rancher stack (default: check_rancher_stack)
- **SERVICE_CHECK_COMMAND** Name of the check command used to monitor a Rancher service (default: check_rancher_stack)
- **RANCHER_INSTALLATION** If you would like to register more than one Rancher installation with Icinga2, give each of them a name.
- **ICINGA_DEFAULT_VARS** If you would like to add custom variables to the objects created in Icinga2, add comma separated k=v values here.
- **REFRESH_INTERVAL** If 0 (the default), update Icinga once and then exit. If > 0, run in an endless loop and update every that many seconds.
- **ICINGA_DEBUG** Add debug output (default: disabled)
- **ICINGA_INSECURE_TLS** Set to 1 to disable strict TLS cert checking when connection to the Icinga2 API (default: disabled)

The following values are available for templates:

- Hostname
- RancherUrl
- RancherAccessKey
- RancherSecretKey
- RancherEnvironment
- RancherStack
- RancherService
