# rancher-icinga

**rancher-icinga** registers Rancher resources in Icinga2 for monitoring. Right now, each rancher agent, stack and service
is individually monitored. Rancher environments are created as Icinga2 hostsgroups. Rancher agents are monitored
as simple hosts. Rancher stacks are hosts and services are created as services for those hosts.

Requires Rancher 1.2.0 or later.

## Getting started

You will need a running Icinga2 that uses https://github.com/Nexinto/check-rancher. See examples/rancher.conf for how to
configure the check_rancher commands. (You can configure alternative check commands using the XXX_CHECK_COMMAND environment variables
(see below).

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
- **FILTER...** See below (Filtering)
- **REGISTER_CHANGES** See below (Registering change events)

The following values are available for templates:

- Hostname
- RancherUrl
- RancherAccessKey
- RancherSecretKey
- RancherEnvironment
- RancherStack
- RancherService

## Filtering

By default, all Rancher environments, agents, stacks and services are added to Icinga. Filters can be set to limit which objects
are monitored. These are set using environment variables:

- **FILTER_ENVIRONMENTS**
- **FILTER_HOSTS**
- **FILTER_STACKS**
- **FILTER_SERVICES**

Each value is a comma-seperated list of filter expressions. Match is last. Use a suffix of `!L` to stop processing at that rule.
A `-` prefix negates the filter expression.

The most obvious way to filter is using labels. Unfortunately, only hosts and services support labels, stacks don't. 

The following filters are supported:

- `*` matches everything.
- A glob expression matches the name of the agent / stack / service.
- `LABEL=VALUE` matches a label value. glob is supported for both LABEL and VALUE.
- `%SYSTEM` matches a system stack or service.
- `%ENV=ENVNAME` matches is the host, stack or service is deployed in the environment ENVNAME. glob is supported.
- `%HAS_SERVICE(SERVICENAME)` matches a stack that has a service named SERVICENAME. glob is supported.
- `%HAS_SERVICE(LABEL=VALUE)` matches a stack that has a service that has a label LABEL with value VALUE. glob is supported for both LABEL and VALUE.
- `%STACK=STACKNAME` matches if the service is a member of the stack STACKNAME. glob is supported for both LABEL and VALUE.

If a stack does not match a filter, no services will be monitored for this stack. There is no similar behaviour for
hosts.

Example 1:

Two environments, "prod" and "dev". We would like to monitor all hosts in all environments, all stacks and services in "prod", 
but only system stacks in "dev".

```
FILTER_HOSTS="*"
FILTER_STACKS="*,-%ENV=dev,%SYSTEM"
FILTER_SERVICES="*,-%ENV=dev,%SYSTEM"
```

FILTER_SERVICES can be left empty or set to "*" in this example, because there are no services that should not be monitored in a stack
that is monitored.

FILTER_STACKS could also be written as:

```
FILTER_STACKS="*,%SYSTEM!L,-%ENV=dev"
```

Example 2:

One environment "prod", where all hosts should be monitored. All System stacks should be monitored. Only
services labeled "monitor=true" should be monitored. Stacks that have such services should be monitored).

```
FILTER_HOSTS="*"
FILTER_STACKS="-*,%SYSTEM,%HAS_SERVICE(monitor=true)"
FILTER_SERVICES="-*,%SYSTEM,monitor=true"
```

(see filter_test.go for more about these two examples)

## Registering change events

Set the environment variable REGISTER_CHANGES to an URL that will receive a POST request with every change that
rancher-icinga makes. A JSON object will be posted with the following fields:
- **operation** - the type of the change (created, delete)
- **name** - the name of the object being created or deleted
- **type** - the object type
- **vars** - the "vars" of the icinga object
