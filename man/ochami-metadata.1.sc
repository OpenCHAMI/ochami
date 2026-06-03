OCHAMI-METADATA(1) "OpenCHAMI" "Manual Page for ochami-metadata"

# NAME

ochami-metadata - Communicate with the Metadata Service

# SYNOPSIS

*ochami metadata* [_global-options_] _command_ [_command-options_] [_arguments_]

*ochami metadata defaults list* [OPTIONS]++
*ochami metadata service status* [-F _format_]

# DATA STRUCTURE

## CLUSTER DEFAULTS

The data structure for sending and receiving cluster defaults is detailed in
JSON form below:

```
```

## GROUP

The data structure for sending and receiving group specifications is detailed in
JSON form below:

```
```

## INSTANCE INFORMATION

The data structure for receiving instance information is detailed in JSON form
below:

```
```

## WIREGUARD PEER

The data structure for sending and receiving WireGuard peer information is
detailed in JSON form below:

```
```

# GLOBAL FLAGS

*--api-version* _version_
	Version of the API to use in the request. Example values are *v1*,
	*v2beta1*. The default is to use the latest stable API version.

*--timeout* _duration_
	Time out duration for making requests. _duration_ is any time duration
	string supported by the Go *time* library.

	The default is *30s* for 30 seconds.

*--uri* _uri_
	Specify either the absolute base URI for the service (e.g.
	_https://foobar.openchami.cluster:8443/metadata_) or a relative base path
	for the service (e.g. _/metadata_). If an absolute URI is specified, this
	completely overrides any value set with the *--cluster-uri* flag or
	*cluster.uri* in the config file for the cluster. If using an absolute URI,
	it should contain the desired service's base path. If a relative path is
	specified (with or without the leading forward slash), then this value
	overrides the service's default base path and is appended to the cluster's
	base URI (set with the *--cluster-uri* flag or the *cluster.uri* cluster
	config option), which is required to be set if a relative path is used here.

	The metadata service has a base path of */metadata-service* by default.

	See *ochami*(1) for *--cluster-uri* and *ochami-config*(5) for details on
	cluster configuration options.

# COMMANDS

## service

Manage and check metadata-service itself.

Subcommands for this command are as follows:

*status* [-F _format_]
	Display status of the metadata service.

	This command sends a GET to metadata-service's health endpoint.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

# AUTHOR

Written by Devon T. Bautista and maintained by the OpenCHAMI developers.

# SEE ALSO

*ochami*(1)

; Vim modeline settings
; vim: set tw=80 noet sts=4 ts=4 sw=4 syntax=scdoc:
