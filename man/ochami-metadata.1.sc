OCHAMI-METADATA(1) "OpenCHAMI" "Manual Page for ochami-metadata"

# NAME

ochami-metadata - Communicate with the Metadata Service

# SYNOPSIS

*ochami metadata* [_global-options_] _command_ [_command-options_] [_arguments_]

*ochami metadata defaults add* [-f _format_] [-d (_data_ | @_path_)]++
*ochami metadata defaults delete* [--no-confirm] _uid_...++
*ochami metadata defaults list* [-F _format_]++
*ochami metadata defaults patch* [-f _format_] [-p _patch_method_] [-d (_data_ | @_path_ | @-)] _uid_++
*ochami metadata defaults patch* (--add _key_=_val_ | --remove _key_=_val_ | --set _key_=_val_ | --unset _key_)... _uid_++
*ochami metadata defaults set* [-f _format_] [-d (_data_ | @_path_)] _uid_++
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

*delete* [--no-confirm] _uid_...
	Delete one or more cluster defaults identified by _uid_. Unless
	*--no-confirm* is passed, the user is asked to confirm deletion.

	This command sends one or more DELETE requests to metadata-service's cluster
	defaults endpoint.

	This command accepts the following options:

	*--no-confirm*
		Do not ask the user to confirm deletion. Use with caution.

*list* [-F _format_]
	List cluster defaults known to metadata-service.

	This command sends a GET to metadata-service's cluster defaults endpoint.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

*set* [-f _format_] < _file_++
*set* [-f _format_] -d @_file_++
*set* [-f _format_] -d @- < _file_++
*set* [-f _format_] -d _data_
	Set the specification of a cluster defaults identified by _uid_. The entire
	specification for the cluster defaults is replaced with the specification
	that is passed.

	In the first and third forms of the command, data is read from standard
	input.

	In the second form of the command, a file containing the payload data is
	passed.

	In the fourth form of the command, the payload is passed raw on the command
	line.

	This command sends a PUT request to metadata-service's cluster defaults
	endpoint.

	This command accepts the following options:

	*-d, --data* (_data_ | @_path_ | @-)
		Specify raw _data_ to send, the _path_ to a file to read payload data
		from, or to read the data from standard input (@-). The format of data
		read in any of these forms is JSON by default unless *-f* is specified
		to change it.

	*-f, --format-input* _format_
		Format of raw data being used by *-d* as the payload. Supported formats
		are:

		- _json_ (default)
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

*patch* ([--add _key_=_val_]... | [--remove _key_=_val_]... | [--set _key_=_val_]... | [--unset _key_]...) _uid_++
*patch* [ -f _format_] [ -p _patch_method_] -d @_file_ _uid_++
*patch* [ -f _format_] [ -p _patch_method_] -d @- _uid_ < _file_++
*patch* [ -f _format_] [ -p _patch_method_] _uid_ < _file_
	Using various patch methods, patch the specification for an existing cluster
	defaults identified by _uid_.

	*IMPORTANT:* Only the spec portion of the resource can be patched.  Metadata
	(name, labels, annotations) and status are managed by the API.  Attempts to
	patch metadata or status fields will be ignored.

	In the first form of the command, at least one of *--add*, *--remove*,
	*--set*, or *--unset* is passed. Each of these flags can be specified more
	than once, but at least one of them must be passed in this form. This method
	uses add/remove/set/unset flags to perform the patch. For _key_, dot
	notation is used for subkeys (e.g. _key.subkey_).

	In the second through fourth forms of the command, patch data is supplied
	along with an optional *--patch-method* flag to specify the patch method.

	This command sends a PATCH request to metadata-service's cluster defaults
	endpoint.

	This command accepts the following options:

	*--add* _key_[[._subkey_]...]=_val_
		Add value to array field, creating the field if necessary. Only can be
		used with _keyval_ patch method (automatic if any of
		*--add*/*--remove*/*--set*/*--unset* are specified).

	*-d, --data* (_data_ | @_path_ | @-)
		Specify raw _data_ to send, the _path_ to a file to read payload data
		from, or to read the data from standard input (@-). The format of data
		read in any of these forms is JSON by default unless *-f* is specified
		to change it.

	*-f, --format-input* _format_
		Format of raw data being used by stdin/*-d* as the payload. Supported
		formats are:

		- _json_ (default)
		- _yaml_

	*-p, --patch-method* _patch_method_
		Specify patch method for patch data. Supported methods are:

		- _rfc7386_ (default): RFC 7386 JSON Merge Patch
		- _rfc6902_: RFC 6902 JSON Patch
		- _keyval_: key=value format using dot notation for subkeys

	*--remove* _key_[[._subkey_]...]=_val_
		Remove value from array field. Only can be used with _keyval_ patch
		method (automatic if any of
		*--add*/*--remove*/*--set*/*--unset* are specified).

	*--set* _key_[[._subkey_]...]=_val_
		Set key with its value, overwriting any previous value and creating if the
		key doesn't exist. Only can be used with _keyval_ patch method (automatic
		if any of *--add*/*--remove*/*--set*/*--unset* are specified).

	*--unset* _key_[[._subkey_]...]
		Unset key (and its value). Only can be used with _keyval_ patch method
		(automatic if any of
		*--add*/*--remove*/*--set*/*--unset* are specified).

*set* [-f _format_] _uid_ < _file_++
*set* [-f _format_] -d @_file_ _uid_++
*set* [-f _format_] -d @- < _file_ _uid_++
*set* [-f _format_] -d _data_ _uid_
	Set the spec for an existing cluster defaults in metadata-service, specified
	by UID.

	In the first and third forms of the command, data is read from standard
	input.

	In the second form of the command, a file containing the payload data is
	passed.

	In the fourth form of the command, the payload is passed raw on the command
	line.

	This command sends a POST request to metadata-service's cluster defaults
	endpoint.

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

# AUTHOR

Written by Devon T. Bautista and maintained by the OpenCHAMI developers.

# SEE ALSO

*ochami*(1)

; Vim modeline settings
; vim: set tw=80 noet sts=4 ts=4 sw=4 syntax=scdoc:
