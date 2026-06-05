OCHAMI-METADATA(1) "OpenCHAMI" "Manual Page for ochami-metadata"

# NAME

ochami-metadata - Communicate with the Metadata Service

# SYNOPSIS

*ochami metadata* [_global-options_] _command_ [_command-options_] [_arguments_]

*ochami metadata* (*defaults* | *group* | *instance* | *peer*) *add* [-f _format_] [-d (_data_ | @_path_)]++
*ochami metadata* (*defaults* | *group* | *instance* | *peer*) *delete* [--no-confirm] _uid_...++
*ochami metadata* (*defaults* | *group* | *instance* | *peer*) *get* [-F _format_] _uid_++
*ochami metadata* (*defaults* | *group* | *instance* | *peer*) *list* [-F _format_]++
*ochami metadata* (*defaults* | *group* | *instance* | *peer*) *patch* [-f _format_] [-p _patch_method_] [-d (_data_ | @_path_ | @-)] _uid_++
*ochami metadata* (*defaults* | *group* | *instance* | *peer*) *patch* (--add _key_=_val_ | --remove _key_=_val_ | --set _key_=_val_ | --unset _key_)... _uid_++
*ochami metadata* (*defaults* | *group* | *instance* | *peer*) *set* [-f _format_] [-d (_data_ | @_path_)] _uid_++
*ochami metadata service status* [-F _format_]

# DATA STRUCTURE

## CLUSTER DEFAULTS

The data structure for sending and receiving cluster defaults is detailed in
JSON form below. Cluster defaults define cluster-wide default metadata values
that can be applied across all nodes in the cluster.

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

Required fields when creating/updating:

- *base_url* (string): Base URL for cloud-init service
- *cluster_name* (string): Name of the cluster

Optional fields:

- *description* (string): Human-readable description
- *cloud_provider* (string): Cloud provider name (e.g., "on-prem", "aws", "azure")
- *region* (string): Region or data center location
- *availability_zone* (string): Availability zone or rack location
- *short_name* (string): Short name prefix for node identifiers (e.g., "nid")
- *nid_length* (integer): Length of node ID numbers
- *public_keys* (string array): Array of SSH public keys for cluster access

## GROUP

The data structure for sending and receiving group specifications is detailed in
JSON form below. Groups define cloud-init templates that can be rendered with
metadata variables for nodes.

When creating groups, the *template* field is required and should contain a valid
cloud-init configuration. The template can use Jinja2-style variable substitution
(e.g. {{ cluster_name }}) which will be rendered when the group is applied to nodes.

For easier template management, YAML format is recommended as it preserves
multi-line strings without escaping.

```
{
  "apiVersion": "cloud-init.openchami.io/v1",
  "kind": "Group",
  "metadata": {
    "name": "compute-group-1",
    "uid": "group-01hzy7h9xq6b8m2p4v1n3r5t7w",
    "labels": {
      "role": "compute",
      "cluster": "demo"
    },
    "createdAt": "2026-01-15T18:30:00Z",
    "updatedAt": "2026-01-15T19:45:00Z"
  },
  "spec": {
    "description": "Compute node group configuration",
    "template": "#cloud-config\\n##template: jinja2\\npackage_update: true\\npackages:\\n  - nfs-common\\n  - chrony\\nruncmd:\\n  - echo \"Configured {{ cluster_name }}\"\\n",
    "metaData": {
      "role": "compute",
      "ntp_server": "10.1.1.100"
    },
    "osVersion": "ubuntu-22.04"
  },
  "status": {
    "valid": true,
    "lastApplied": "2026-01-15T19:45:00Z",
    "currentTemplateVersion": "v-a1b2c3d4",
    "requiredVariables": ["cluster_name"]
  }
}
```

Required fields when creating/updating:

- *template* (string): Cloud-init configuration template with optional Jinja2 variables

Optional fields:

- *description* (string): Human-readable description
- *metaData* (string map): Key-value pairs for template variable substitution
- *osVersion* (string): Target OS version

## INSTANCE INFORMATION

The data structure for sending and receiving instance information is detailed in
JSON form below:

```
{
  "apiVersion": "cloud-init.openchami.io/v1",
  "kind": "InstanceInfo",
  "metadata": {
    "name": "x1000c0s0b0n0-instance",
    "uid": "instanceinfo-01hzy7h9xq6b8m2p4v1n3r5t7w",
    "createdAt": "2026-01-15T18:30:00Z",
    "updatedAt": "2026-01-15T19:45:00Z"
  },
  "spec": {
    "description": "Compute node instance information",
    "instance_id": "x1000c0s0b0n0",
    "local_hostname": "nid001000",
    "hostname": "nid001000.demo.cluster",
    "cloud_init_base_url": "https://demo.openchami.cluster:8443/cloud-init",
    "public_keys": [
      "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMLtQNuzGcMDatF+YVMMkuxbX2c5v2OxWftBhEVfFb+U admin@demo"
    ],
    "default_profile": "compute"
  },
  "status": {
    "phase": "Ready",
    "message": "Instance info is active",
    "ready": true
  }
}
```

Required fields when creating/updating:

- *instance_id* (string): Unique instance identifier

Optional fields:

- *description* (string): Human-readable description (max 200 characters)
- *local_hostname* (string): Local hostname for the instance
- *hostname* (string): Fully qualified hostname
- *cloud_init_base_url* (string): Base URL for cloud-init service (must be valid URL)
- *public_keys* (string array): Array of SSH public keys
- *default_profile* (string): Default profile to use

## WIREGUARD PEER

The data structure for sending and receiving WireGuard peer information is
detailed in JSON form below:

```
{
  "apiVersion": "cloud-init.openchami.io/v1",
  "kind": "WireGuardPeer",
  "metadata": {
    "name": "peer-nid001000",
    "uid": "wireguardpeer-01hzy7h9xq6b8m2p4v1n3r5t7w",
    "createdAt": "2026-01-15T18:30:00Z",
    "updatedAt": "2026-01-15T19:45:00Z"
  },
  "spec": {
    "description": "WireGuard peer for nid001000",
    "public_key": "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=",
    "allowed_ip": "10.42.1.1/32"
  },
  "status": {
    "phase": "Ready",
    "message": "Peer is configured",
    "ready": true
  }
}
```

Required fields when creating/updating:

- *public_key* (string): WireGuard public key (base64 encoded)
- *allowed_ip* (string): Allowed IP address in CIDR notation

Optional fields:

- *description* (string): Human-readable description

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

The *defaults*, *group*, *instance*, and *peer* commands share a common set of
subcommands for creating, deleting, reading, listing, patching, and replacing
metadata-service resources. The *service* command provides operations for
metadata-service itself.

[[ *Resource*
:< *Subcommands*
:< *Description*
|  *defaults*
:  *add*, *delete*, *get*, *list*, *patch*, *set*
:  Manage cluster-wide default metadata values
|  *group*
:  *add*, *delete*, *get*, *list*, *patch*, *set*
:  Manage cloud-init group templates
|  *instance*
:  *add*, *delete*, *get*, *list*, *patch*, *set*
:  Manage instance information
|  *peer*
:  *add*, *delete*, *get*, *list*, *patch*, *set*
:  Manage WireGuard peer configurations
|  *service*
:  *status*
:  Check metadata-service status

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

*get* [-F _format_] _uid_
	Get details for cluster defaults identified by _uid_.

	This command sends a GET to metadata-service's cluster defaults endpoint.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

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

## group

Manage cloud-init group templates in the metadata service.

Subcommands for this command are as follows:

*add* [-f _format_] < _file_++
*add* [-f _format_] -d @_file_++
*add* [-f _format_] -d @- < _file_++
*add* [-f _format_] -d _data_
	Add one or more groups to metadata-service.

	In the first and third forms of the command, data is read from standard
	input.

	In the second form of the command, a file containing the payload data is
	passed.

	In the fourth form of the command, the payload is passed raw on the command
	line.

	This command sends one or more POST requests to metadata-service's group
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

	Examples:

	```
	# Add group with cloud-init template from YAML file
	ochami metadata group add -d @compute-group.yaml -f yaml

	# Add group with inline multi-line template (YAML via stdin)
	ochami metadata group add -f yaml -d - <<'EOF'
	template: |
	  #cloud-config
	  package_update: true
	  packages:
	    - nfs-common
	    - chrony
	metaData:
	  role: compute
	EOF

	# Add group using JSON
	ochami metadata group add -d '{
	  "template":"#cloud-config\npackages:\n  - vim\n",
	  "metaData":{"role":"storage"}
	}'

	# Add multiple groups using JSON array
	ochami metadata group add -d '[
	  {
	    "template":"#cloud-config\npackages:\n  - nfs-common\n"
	  },
	  {
	    "template":"#cloud-config\npackages:\n  - nfs-server\n"
	  }
	]'

	# Add multiple groups using YAML array
	ochami metadata group add -f yaml -d - <<'EOF'
	- template: |
	    #cloud-config
	    packages:
	      - nfs-common
	- template: |
	    #cloud-config
	    packages:
	      - nfs-server
	EOF

	# Add multiple groups from file
	ochami metadata group add -d @groups.json
	```

*delete* [--no-confirm] _uid_...
	Delete one or more groups identified by _uid_. Unless *--no-confirm* is
	passed, the user is asked to confirm deletion.

	This command sends one or more DELETE requests to metadata-service's group
	endpoint.

	This command accepts the following options:

	*--no-confirm*
		Do not ask the user to confirm deletion. Use with caution.

	Examples:

	```
	# Delete a group
	ochami metadata group delete group-d614b918

	# Delete multiple groups
	ochami metadata group delete group-d614b918 group-82c40109

	# Don't confirm deletion
	ochami metadata group delete --no-confirm group-d614b918
	```

*get* [-F _format_] _uid_
	Get details for a group identified by _uid_.

	This command sends a GET to metadata-service's group endpoint.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

	Examples:

	```
	# Get info about a group
	ochami metadata group get group-773d99bf

	# Get group in YAML format
	ochami metadata group get group-773d99bf -F yaml
	```

*list* [-F _format_]
	List groups known to metadata-service.

	This command sends a GET to metadata-service's group endpoint.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

	Examples:

	```
	# List all groups
	ochami metadata group list

	# List groups in YAML format
	ochami metadata group list -F yaml
	```

*patch* ([--add _key_=_val_]... | [--remove _key_=_val_]... | [--set _key_=_val_]... | [--unset _key_]...) _uid_++
*patch* [-f _format_] [-p _patch_method_] -d @_file_ _uid_++
*patch* [-f _format_] [-p _patch_method_] -d @- _uid_ < _file_++
*patch* [-f _format_] [-p _patch_method_] _uid_ < _file_
	Using various patch methods, patch the specification for an existing group
	identified by _uid_.

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

	This command sends a PATCH request to metadata-service's group endpoint.

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

	Examples:

	```
	# Patch using JSON patch (RFC 6902)
	ochami metadata group patch group-d614b918 --patch-method rfc6902 --data '[
	  {"op":"replace","path":"/osVersion","value":"ubuntu-24.04"}
	]'

	# Patch using JSON merge patch (RFC 7386)
	ochami metadata group patch group-d614b918 --patch-method rfc7386 --data '{"osVersion":"ubuntu-24.04"}'

	# Patch using dot notation (keyval)
	ochami metadata group patch group-d614b918 --patch-method keyval --set osVersion='ubuntu-24.04'

	# Patch using payload file
	ochami metadata group patch group-d614b918 -d @payload.json
	```

*set* [-f _format_] _uid_ < _file_++
*set* [-f _format_] -d @_file_ _uid_++
*set* [-f _format_] -d @- _uid_ < _file_++
*set* [-f _format_] -d _data_ _uid_
	Set the specification of a group identified by _uid_. The entire
	specification for the group is replaced with the specification that is
	passed.

	In the first and third forms of the command, data is read from standard
	input.

	In the second form of the command, a file containing the payload data is
	passed.

	In the fourth form of the command, the payload is passed raw on the command
	line.

	This command sends a PUT request to metadata-service's group endpoint.

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

	Examples:

	```
	# Set group details using YAML file
	ochami metadata group set group-d614b918 -d @group.yaml -f yaml

	# Set group details using JSON
	ochami metadata group set group-d614b918 -d '{
	  "template":"#cloud-config\npackages:\n  - vim\n",
	  "metaData":{"role":"compute"}
	}'
	```

## instance

Manage instance information in the metadata service.

Subcommands for this command are as follows:

*add* [-f _format_] < _file_++
*add* [-f _format_] -d @_file_++
*add* [-f _format_] -d @- < _file_++
*add* [-f _format_] -d _data_
	Add one or more instance infos to metadata-service.

	In the first and third forms of the command, data is read from standard
	input.

	In the second form of the command, a file containing the payload data is
	passed.

	In the fourth form of the command, the payload is passed raw on the command
	line.

	This command sends one or more POST requests to metadata-service's instance
	info endpoint.

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

	Examples:

	```
	# Add instance info using JSON
	ochami metadata instance add -d '{
	  "instance_id": "x1000c0s0b0n0",
	  "hostname": "nid001000.demo.cluster",
	  "local_hostname": "nid001000"
	}'

	# Add multiple instance infos using JSON array
	ochami metadata instance add -d '[
	  {
	    "instance_id": "x1000c0s0b0n0"
	  },
	  {
	    "instance_id": "x1000c0s0b0n1"
	  }
	]'

	# Add multiple instance infos using YAML array
	ochami metadata instance add -f yaml -d - <<'EOF'
	- instance_id: "x1000c0s0b0n0"
	- instance_id: "x1000c0s0b0n1"
	EOF

	# Add instance from YAML file
	ochami metadata instance add -d @instance.yaml -f yaml

	# Add multiple instances from file
	ochami metadata instance add -d @instances.json
	```

*delete* [--no-confirm] _uid_...
	Delete one or more instance infos identified by _uid_. Unless *--no-confirm*
	is passed, the user is asked to confirm deletion.

	This command sends one or more DELETE requests to metadata-service's instance
	info endpoint.

	This command accepts the following options:

	*--no-confirm*
		Do not ask the user to confirm deletion. Use with caution.

	Examples:

	```
	# Delete an instance info
	ochami metadata instance delete instanceinfo-d614b918

	# Delete multiple instance infos
	ochami metadata instance delete instanceinfo-d614b918 instanceinfo-82c40109

	# Don't confirm deletion
	ochami metadata instance delete --no-confirm instanceinfo-d614b918
	```

*get* [-F _format_] _uid_
	Get details for an instance info identified by _uid_.

	This command sends a GET to metadata-service's instance info endpoint.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

	Examples:

	```
	# Get info about an instance
	ochami metadata instance get instanceinfo-773d99bf

	# Get instance info in YAML format
	ochami metadata instance get instanceinfo-773d99bf -F yaml
	```

*list* [-F _format_]
	List instance infos known to metadata-service.

	This command sends a GET to metadata-service's instance info endpoint.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

	Examples:

	```
	# List all instance infos
	ochami metadata instance list

	# List instance infos in YAML format
	ochami metadata instance list -F yaml
	```

*patch* ([--add _key_=_val_]... | [--remove _key_=_val_]... | [--set _key_=_val_]... | [--unset _key_]...) _uid_++
*patch* [-f _format_] [-p _patch_method_] -d @_file_ _uid_++
*patch* [-f _format_] [-p _patch_method_] -d @- _uid_ < _file_++
*patch* [-f _format_] [-p _patch_method_] _uid_ < _file_
	Using various patch methods, patch the specification for an existing instance
	info identified by _uid_.

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

	This command sends a PATCH request to metadata-service's instance info
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

	Examples:

	```
	# Patch using JSON patch (RFC 6902)
	ochami metadata instance patch instanceinfo-d614b918 --patch-method rfc6902 --data '[
	  {"op":"replace","path":"/hostname","value":"nid002000.demo.cluster"}
	]'

	# Patch using JSON merge patch (RFC 7386)
	ochami metadata instance patch instanceinfo-d614b918 --patch-method rfc7386 --data '{"hostname":"nid002000.demo.cluster"}'

	# Patch using dot notation (keyval)
	ochami metadata instance patch instanceinfo-d614b918 --set hostname='nid002000.demo.cluster'

	# Patch using payload file
	ochami metadata instance patch instanceinfo-d614b918 -d @payload.yaml -f yaml
	```

*set* [-f _format_] _uid_ < _file_++
*set* [-f _format_] -d @_file_ _uid_++
*set* [-f _format_] -d @- _uid_ < _file_++
*set* [-f _format_] -d _data_ _uid_
	Set the specification of an instance info identified by _uid_. The entire
	specification for the instance info is replaced with the specification that
	is passed.

	In the first and third forms of the command, data is read from standard
	input.

	In the second form of the command, a file containing the payload data is
	passed.

	In the fourth form of the command, the payload is passed raw on the command
	line.

	This command sends a PUT request to metadata-service's instance info endpoint.

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

	Examples:

	```
	# Set instance info using YAML file
	ochami metadata instance set instanceinfo-d614b918 -d @instance.yaml -f yaml

	# Set instance info using JSON
	ochami metadata instance set instanceinfo-d614b918 -d '{
	  "instance_id": "x1000c0s0b0n0",
	  "hostname": "nid001000.demo.cluster"
	}'
	```

## peer

Manage WireGuard peer configurations in the metadata service.

Subcommands for this command are as follows:

*add* [-f _format_] < _file_++
*add* [-f _format_] -d @_file_++
*add* [-f _format_] -d @- < _file_++
*add* [-f _format_] -d _data_
	Add one or more WireGuard peers to metadata-service.

	In the first and third forms of the command, data is read from standard
	input.

	In the second form of the command, a file containing the payload data is
	passed.

	In the fourth form of the command, the payload is passed raw on the command
	line.

	This command sends one or more POST requests to metadata-service's WireGuard
	peer endpoint.

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

	Examples:

	```
	# Add WireGuard peer using JSON
	ochami metadata peer add -d '{
	  "public_key": "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=",
	  "allowed_ip": "10.42.1.1/32",
	  "description": "Peer for nid001000"
	}'

	# Add peer from YAML
	ochami metadata peer add -f yaml -d - <<'EOF'
	public_key: xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=
	allowed_ip: 10.42.1.1/32
	description: Compute node peer
	EOF

	# Add multiple WireGuard peers using JSON array
	ochami metadata peer add -d '[
	  {
	    "public_key": "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=",
	    "allowed_ip": "10.42.1.1/32"
	  },
	  {
	    "public_key": "yUJCB6sbcpVwoI5iupekc7f798RkMFSu2OBC5nArq9Eh=",
	    "allowed_ip": "10.42.1.2/32"
	  }
	]'

	# Add multiple WireGuard peers using YAML array
	ochami metadata peer add -f yaml -d - <<'EOF'
	- public_key: "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg="
	  allowed_ip: "10.42.1.1/32"
	- public_key: "yUJCB6sbcpVwoI5iupekc7f798RkMFSu2OBC5nArq9Eh="
	  allowed_ip: "10.42.1.2/32"
	EOF

	# Add multiple peers from file
	ochami metadata peer add -d @peers.json
	```

*delete* [--no-confirm] _uid_...
	Delete one or more WireGuard peers identified by _uid_. Unless *--no-confirm*
	is passed, the user is asked to confirm deletion.

	This command sends one or more DELETE requests to metadata-service's WireGuard
	peer endpoint.

	This command accepts the following options:

	*--no-confirm*
		Do not ask the user to confirm deletion. Use with caution.

	Examples:

	```
	# Delete a WireGuard peer
	ochami metadata peer delete wireguardpeer-d614b918

	# Delete multiple WireGuard peers
	ochami metadata peer delete wireguardpeer-d614b918 wireguardpeer-82c40109

	# Don't confirm deletion
	ochami metadata peer delete --no-confirm wireguardpeer-d614b918
	```

*get* [-F _format_] _uid_
	Get details for a WireGuard peer identified by _uid_.

	This command sends a GET to metadata-service's WireGuard peer endpoint.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

	Examples:

	```
	# Get info about a WireGuard peer
	ochami metadata peer get wireguardpeer-773d99bf

	# Get WireGuard peer in YAML format
	ochami metadata peer get wireguardpeer-773d99bf -F yaml
	```

*list* [-F _format_]
	List WireGuard peers known to metadata-service.

	This command sends a GET to metadata-service's WireGuard peer endpoint.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

	Examples:

	```
	# List all WireGuard peers
	ochami metadata peer list

	# List WireGuard peers in YAML format
	ochami metadata peer list -F yaml
	```

*patch* ([--add _key_=_val_]... | [--remove _key_=_val_]... | [--set _key_=_val_]... | [--unset _key_]...) _uid_++
*patch* [-f _format_] [-p _patch_method_] -d @_file_ _uid_++
*patch* [-f _format_] [-p _patch_method_] -d @- _uid_ < _file_++
*patch* [-f _format_] [-p _patch_method_] _uid_ < _file_
	Using various patch methods, patch the specification for an existing WireGuard
	peer identified by _uid_.

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

	This command sends a PATCH request to metadata-service's WireGuard peer
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

	Examples:

	```
	# Patch using JSON patch (RFC 6902)
	ochami metadata peer patch wireguardpeer-d614b918 --patch-method rfc6902 --data '[
	  {"op":"replace","path":"/allowed_ip","value":"10.42.2.1/32"}
	]'

	# Patch using JSON merge patch (RFC 7386)
	ochami metadata peer patch wireguardpeer-d614b918 --patch-method rfc7386 --data '{"allowed_ip":"10.42.2.1/32"}'

	# Patch using dot notation (keyval)
	ochami metadata peer patch wireguardpeer-d614b918 --set allowed_ip='10.42.2.1/32'

	# Patch using payload file
	ochami metadata peer patch wireguardpeer-d614b918 -d @payload.json
	```

*set* [-f _format_] _uid_ < _file_++
*set* [-f _format_] -d @_file_ _uid_++
*set* [-f _format_] -d @- _uid_ < _file_++
*set* [-f _format_] -d _data_ _uid_
	Set the specification of a WireGuard peer identified by _uid_. The entire
	specification for the WireGuard peer is replaced with the specification that
	is passed.

	In the first and third forms of the command, data is read from standard
	input.

	In the second form of the command, a file containing the payload data is
	passed.

	In the fourth form of the command, the payload is passed raw on the command
	line.

	This command sends a PUT request to metadata-service's WireGuard peer endpoint.

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

	Examples:

	```
	# Set WireGuard peer using YAML file
	ochami metadata peer set wireguardpeer-d614b918 -d @peer.yaml -f yaml

	# Set WireGuard peer using JSON
	ochami metadata peer set wireguardpeer-d614b918 -d '{
	  "public_key": "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=",
	  "allowed_ip": "10.42.1.1/32"
	}'
	```

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
