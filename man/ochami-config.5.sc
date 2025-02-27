OCHAMI-CONFIG(5) "OpenCHAMI" "ochami: The OpenCHAMI CLI Tool"

# NAME

config.yaml - ochami CLI configuration file

# DESCRIPTION

*ochami* uses the YAML format for its configuration files. Configuration options
can be set via the *ochami config* command.

# GLOBAL CONFIGURATION OPTIONS

These configuration options are global configuration options.

*clusters*
	The list of cluster configurations. Each cluster configuration is a block
	that has a *name* field that contains a string uniquely identifying the cluster,
	as well as a *cluster* field that contains the cluster configuration.

	Importantly, the value of *name* is used when determining the environment
	variable to read when presenting a token for the cluster. The value is made
	upper case, hyphens are converted to underscores, and the result is
	prepended to *\_ACCESS_TOKEN*. For instance, the token environment variable
	for a cluster named *my-cluster* would be *MY_CLUSTER_ACCESS_TOKEN*.

	See *CLUSTER CONFIGURATION* below for details on cluster configuration.

	The format is:

	```
	clusters:
	- name: foobar
	  cluster:
	    <cluster_config>
	```

*default-cluster:* _cluster_name_
	The name of the default cluster to use when *--cluster* is not specified on
	the command line. A cluster configuration must exist for _cluster_name_ or
	further commands will fail.

*log*
	Logging options.

	*format:* _format_
		The format of log messages.

		Default: *json*
		Supported:

		- _basic_
		- _json_
		- _rfc3339_

	*level:* _level_
		Logging level.

		Default: *warning*
		Supported:

		- _info_
		- _warning_
		- _debug_

# CLUSTER CONFIGURATION

These configuration options apply only to cluster configuration, i.e. under the
*clusters* key. The value for the *cluster* key is an array with each item in
the array containing the below configuration options.

*name:* _cluster_name_
	The name of the cluster. This is what *--cluster* and the *default-cluster*
	key use to identify the cluster.

*<service>*
	The service-specific configuration for *<service>*. Currently recognized
	values of *<service>* are:

	- _bss_ (Boot Script Service)
	- _cloud-init_ (cloud-init)
	- _pcs_ (Power Control Service)
	- _smd_ (State Management Database)

	Each service config can have the following configuration options:

	*uri:* _absolute_uri_or_relative_path_
		Specify either the absolute base URI for the service (e.g.
		_https://foobar.openchami.cluster:8443/hsm/v2_) or a relative base path
		for the service (e.g. _/hsm/v2_). If an absolute URI is specified, this
		completely overrides any value set for *cluster.uri* and the absolute
		URI should also contain the desired service's base path. If a relative
		path is specified (with or without the leading forward slash), then this
		value overrides the service's default base path and is appended to
		*cluster.uri*, which is required to be set if a relative path is used
		here.

		This option should be used when either one or more of the OpenCHAMI
		services is using a custom base path or when it/they have an entirely
		different URI, as when running bare metal on localhost with different
		ports. *ochami* determines the base URI to use for each service by
		checking *cluster.uri* and then if an override is set by a
		*cluster.<service>.uri* directive for the *<service>*, so at least one
		of these need to be set.  Otherwise, the base URI is not able to be
		determined for that service.

*uri:* _absolute_uri_
	The base URI for the OpenCHAMI services for the cluster. This is
	normally used when most or all of the OpenCHAMI services are behind a
	single base URI (e.g. _https://foobar.openchami.cluster:8443_), and
	*ochami* will append the service base path (e.g. _/hsm/v2_) as well as
	the request endpoint onto this to fulfill the request for the specific
	service. If one or more OpenCHAMI services is running either with a
	custom base path or a custom URI altogether (e.g.  running on localhost
	under different ports), then *cluster.<service>.uri* can be used to override
	either the service base path or the entire URI.

	Thus, either *cluster.uri* must be specified with optional
	*cluster.<service>.uri* directives for overrides, or a
	*cluster.<service>.uri* must be specified for each *<service>*.

# EXAMPLES

## Cluster Config Variations

*1. All services using default base paths under a single base URI*

```
clusters:
    - cluster:
        uri: https://foobar.openchami.cluster
      name: foobar
default-cluster: foobar
log:
    format: json
    level: debug
```

*2. Using a different URI for each service*

```
clusters:
    - cluster:
        bss:
		  uri: https://localhost:27778/boot/v1
        cloud-init:
		  uri: https://localhost:27777/cloud-init
        pcs:
		  uri: https://localhost:28007/
        smd:
		  uri: https://localhost:27779/hsm/v2
      name: foobar
default-cluster: foobar
log:
    format: json
    level: debug
```

*3. Same as (1) with SMD using a custom base path*

```
clusters:
    - cluster:
        uri: https://foobar.openchami.cluster
        smd:
          uri: /smd
      name: foobar
default-cluster: foobar
log:
    format: json
    level: debug
```

*4. Same as (1) with SMD using an entirely different URI*

```
clusters:
    - cluster:
        uri: https://foobar.openchami.cluster
        smd:
          uri: https://smd.foobar.openchami.cluster/hsm/v2
      name: foobar
default-cluster: foobar
log:
    format: json
    level: debug
```

# FILES

_/etc/ochami/config.yaml_
_~/.config/ochami/config.yaml_

# AUTHOR

Written by Devon T. Bautista and maintained by the OpenCHAMI developers.

# SEE ALSO

*ochami*(1), *ochami-config*(1)

; Vim modeline settings
; vim: set tw=80 noet sts=4 ts=4 sw=4 syntax=scdoc:
