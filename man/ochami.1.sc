OCHAMI(1) "OpenCHAMI" "Manual Page for ochami"

# NAME

ochami - OpenCHAMI command line interface

# SYNOPSIS

ochami [OPTIONS] COMMAND

# DESCRIPTION

*ochami* is a command line interface tool that interacts with the OpenCHAMI API.
It is meant to provide a convenience so one need not craft *curl* incantations,
but also provide additional features.

List of available commands:

[[ *Command*
:< *Description*
|  *bss*
:  Communicate with the Boot Script Service (BSS)
|  *cloud-init*
:  Manage cloud-init configurations
|  *discover*
:  Simulate discovery of BMCs and nodes to populate SMD by reading an input file
|  *smd*
:  Communicate with the State Management Database (SMD)
|  *config*
:  Manage ochami CLI configuration, including cluster configuration

## Top-Level Commands

Top-level commands can be thought of as more abstract and admin-friendly.
These commands tend not to correspond 1:1 with single HTTP requests. For
instance, the *discover* command makes multiple requests to different SMD
endpoints, some of them iteratively.

## Low-Level Commands

Commands that correspond to single HTTP requests ("low-level" commands) are
organized under metacommands corresponding to the service that they send the
request to, under subcommands corresponding to the endpoint of the request. For
instance, the *smd* command has the *component* subcommand that deals with SMD's
_/Components_ endpoint. This subcommand has further subcommands *add*, *delete*,
and *get* which send POST, DELETE, and GET HTTP requests, respectively. These
commands are primarily used for manually adding/getting/modifying/deleting data
structures, e.g. when troubleshooting.

# GETTING STARTED

Upon first running *ochami*, a config file will need to be generated and a basic
cluster configuration will need to be specified. Both can be done with the
following command:

```
ochami config --user cluster set --default foobar cluster.uri https://foobar.openchami.cluster
```

This will create a cluster called _foobar_ and set its base URI to
_https://foobar.openchami.cluster_, placing this config in
_~/.config/ochami/config.yaml_ (the user config file). Since *ochami* supports
multiple cluster configurations, the _--default_ tells *ochami* to set this
cluster as the default cluster, which means that this cluster's configuration
will be used if _--cluster_ is not specified on the command line.

If _--config_ is not passed, the configuration is merged from the system
configuration with the user configuration. See *FILES* below for the location of
these files. If none of these exist, compile-time default values are used.

Once the cluster configuration has been specified, one will need to store a
token to be able to be used to authenticate to protected endpoints without
having to specify _--token_ every time. *ochami* will look for an environment
variable named *\<CLUSTER_NAME\>_ACCESS_TOKEN*, where *\<CLUSTER_NAME\>* is
replaced by the name of the cluster in all capital letters and dashes replaced
by underscores. In the above example, this variable would be named
*FOOBAR_ACCESS_TOKEN*:

```
export FOOBAR_ACCESS_TOKEN=...
```

Once these steps are completed, *ochami* should be ready to use with cluster
_foobar_.

# GLOBAL OPTIONS

*--cacert* _cacert_
	Specify the path to a certificate authority (CA) certificate file to use to
	verify TLS certificates. Must be PEM-formatted.

*-C, --cluster* _cluster_name_
	Specify the name of a cluster to use. The cluster corresponding to the
	passed cluster name must exist in a config file.

*-u, --cluster-uri* _uri_
	Specify cluster base URI to use. This is required to be an absolute URI
	since the base path of the service(s) being communicated with will be
	appended to this URI. Using the *--uri* flag on a service command or
	*cluster.<service>.uri* in the config file for a cluster can override this
	value for the specific service. The *--cluster-uri* flag overrides the
	*cluster.uri* config file option for the cluster.

	See *ochami-config*(5) for details on cluster config options, as well as the
	manual pages for the services in *ochami*(1) for details on *--uri*.

*-c, --config* _config_file_
	Specify the path to a config file to use. By default, the configuration is
	merged from the system config with the user config (see *FILES* below). The
	format of this file should be YAML.

*--ignore-config*
	Do not read configuration from any configuration file.

*-k, --insecure*
	Do not verify TLS certificates.

*-L, --log-format* _format_
	Specify the format of log messages, overriding what is set in the config
	file. Defaults to _json_.

	Supported log formats are:

	- _basic_
	- _json_
	- _rfc3339_

*-l, --log-level* _level_
	Specify the level to print log messages at, overriding what is set in the
	config file. Defaults to _warning_.

	Supported log levels are:

	- _info_
	- _warning_
	- _debug_

*-t, --token* _token_
	Access token to include in request headers for authentication to protected
	service endpoints. Overrides token set in environment variable.

# FILES

_/usr/share/doc/ochami/config.example.yaml_
	An example configuration file that can be used for reference.

# AUTHOR

Written by Devon T. Bautista and maintained by the OpenCHAMI developers.

# SEE ALSO

*ochami-bss*(1), *ochami-cloud-init*(1), *ochami-config*(1),
*ochami-discover*(1), *ochami-smd*(1), *ochami-config*(5)

; Vim modeline settings
; vim: set tw=80 noet sts=4 ts=4 sw=4 syntax=scdoc:
