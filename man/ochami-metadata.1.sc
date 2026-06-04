OCHAMI-METADATA(1) "OpenCHAMI" "Manual Page for ochami-metadata"

# NAME

ochami-metadata - Communicate with the Metadata Service

# SYNOPSIS

*ochami metadata* [_global-options_] _command_ [_command-options_] [_arguments_]

*ochami metadata defaults add* [-f _format_] [-d (_data_ | @_path_)]++
*ochami metadata defaults list* [-F _format_]++
*ochami metadata service status* [-F _format_]

# DATA STRUCTURE

## CLUSTER DEFAULTS

The whole data structure used with cluster defaults is in JSON Form below:

```
{
  "apiVersion": "cloud-init.openchami.io/v1",
  "kind": "ClusterDefaults",
  "metadata": {
    "name": "demo-cluster-defaults",
    "uid": "clusterdefaults-demo-01hzy7h9xq6b8m2p4v1n3r5t7w",
    "labels": {
      "cluster": "demo",
      "environment": "production"
    },
    "annotations": {
      "contact.email": "hpc-ops@example.com",
      "deployment.notes": "Default metadata for the demo OpenCHAMI cluster"
    },
    "createdAt": "2026-01-15T18:30:00Z",
    "updatedAt": "2026-01-15T19:45:00Z"
  },
  "spec": {
    "description": "Cluster-wide defaults for the demo OpenCHAMI environment",
    "base_url": "https://demo.openchami.cluster:8443/cloud-init",
    "cloud_provider": "on-prem",
    "region": "us-west-dc1",
    "availability_zone": "rack-row-a",
    "cluster_name": "demo",
    "short_name": "nid",
    "nid_length": 4,
    "public_keys": [
      "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMLtQNuzGcMDatF+YVMMkuxbX2c5v2OxWftBhEVfFb+U hpc-admin@demo-login",
      "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIB4vVRvkzmGE5PyWX2fuzJEgEfET4PRLHXCnD1uFZ8ZL automation@demo-login"
    ]
  },
  "status": {
    "phase": "Ready",
    "message": "Cluster defaults are active",
    "ready": true
  }
}
```

The above is an example of what is returned when fetching cluster defaults.

When creating/updating cluster defaults, only the *spec* portion is used. The
required fields are:

- *base_url*
- *cluster_name*

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

## defaults

Manage cluster defaults in the metadata service.

Subcommands for this command are as follows:

*add* [-f _format_] < _file_++
*add* [-f _format_] -d @_file_++
*add* [-f _format_] -d @- < _file_++
*add* [-f _format_] -d _data_
	Set one or more cluster defaults in metadata-service

	In the first and third forms of the command, data is read from standard
	input.

	In the second form of the command, a file containing the payload data is
	passed.

	In the fourth form of the command, the payload is passed raw on the command
	line.

	This command sends one or more POST requests to metadata-service's cluster
	defaults endpoint.

	This command accepts the following flags:

	*-d, --data* (_data_ | @_path_ | @-)
		Specify raw _data_ to send, the _path_ to a file to read payload data
		from, or to read the data from standard input (@-). The format of data
		read in any of these forms is JSON by default unless *-f* is specified
		to change it.

	*-f, --format-input* _format_
		Format of raw data being used by stdin/*-d* as the payload. Supported
		formats are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

*list* [-F _format_]
	List cluster defaults known to metadata-service.

	This command sends a GET to metadata-service's cluster defaults endpoint.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

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
