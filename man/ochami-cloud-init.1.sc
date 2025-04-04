OCHAMI-CLOUD-INIT(1) "OpenCHAMI" "Manual Page for ochami-cloud-init"

# NAME

ochami-cloud-init - Communicate with the cloud-init server

# SYNOPSIS

ochami cloud-init defaults get [OPTIONS]++
ochami cloud-init defaults set [OPTIONS]++
ochami cloud-init group add [OPTIONS]++
ochami cloud-init group delete [OPTIONS] ([-d (_data_ | @_path_)] [-f _format_]) | _group_...++
ochami cloud-init group get [OPTIONS] raw [_id_...]++
ochami cloud-init group get [OPTIONS] config [_id_...]++
ochami cloud-init group get [OPTIONS] meta-data [_id_...]++
ochami cloud-init group render _group_ _id_++
ochami cloud-init group set [OPTIONS]++
ochami cloud-init node get group [OPTIONS] _group_ _id_...++
ochami cloud-init node get meta-data [OPTIONS] _id_...++
ochami cloud-init node get user-data [OPTIONS] _id_...++
ochami cloud-init node get vendor-data [OPTIONS] _id_...++
ochami cloud-init node set [OPTIONS]

# DATA STRUCTURE

cloud-init uses different data structures in its API, depending on the endpoint.

## CLUSTER DEFAULTS

Certain values can be set to be used as fallback (default) values if not set by
any group or node. This structure is used with the
*/cloud-init/admin/cluster-defaults* endpoint.

An example of a structure of default values in JSON format is:

```
{
  "availability-zone": "string",
  "base-url": "http://demo.openchami.cluster:8081/cloud-init",
  "boot-subnet": "string",
  "cloud_provider": "string",
  "cluster-name": "demo",
  "nid-length": 3,
  "public-keys": [
    "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMLtQNuzGcMDatF+YVMMkuxbX2c5v2OxWftBhEVfFb+U user1@demo-head",
    "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIB4vVRvkzmGE5PyWX2fuzJEgEfET4PRLHXCnD1uFZ8ZL user2@demo-head"
  ],
  "region": "string",
  "short-name": "nid",
  "wg-subnet": "string"
}
```

## GROUP DATA

Group data contains, unsurprisingly, data for a group.

This structure is used for the */cloud-init/admin/groups* family of endpoints.

It has a *name* and *description* describing the group itself. There is also
*file* which contains the cloud-init configuration file for the group, which can
either be encoded plainly (*"encoding": "plain"*) or via base64 (*"encoding":
"base64"*). *content* contains the actual cloud-init configuration contents,
formatted as the value of *encoding*. *meta-data* contains key-value pairs
representing variables for the group that can be used in cloud-init configs if
it uses Jinja2 templating.

An example of a group data structure in JSON format:

```
{
  "description": "The compute group",
  "file": {
    "content": "IyMgdGVtcGxhdGU6IGppbmphCiNjbG91ZC1jb25maWcKbWVyZ2VfaG93OgotIG5hbWU6IGxpc3QKICBzZXR0aW5nczogW2FwcGVuZF0KLSBuYW1lOiBkaWN0CiAgc2V0dGluZ3M6IFtub19yZXBsYWNlLCByZWN1cnNlX2xpc3RdCnVzZXJzOgogIC0gbmFtZTogcm9vdAogICAgc3NoX2F1dGhvcml6ZWRfa2V5czoge3sgZHMubWV0YV9kYXRhLmluc3RhbmNlX2RhdGEudjEucHVibGljX2tleXMgfX0KZGlzYWJsZV9yb290OiBmYWxzZQo=",
    "encoding": "base64",
    "filename": "cloud-config.yaml"
  },
  "meta-data": {
    "foo": "bar"
  },
  "name": "compute"
}
```

## NODE META-DATA

Node-specific meta-data is used with the
*/cloud-init/impersonation/{id}/meta-data* and */cloud-init/meta-data*
endpoints. The structure represents all of the cloud-init meta-data for a
specific node, aggregated from group- and node-specific configs.

An example in JSON format is:

```
{
  "cluster-name": "demo",
  "hostname": "compute-1.demo.openchami.cluster",
  "instance-data": {
    "v1": {
      "availability-zone": "string",
      "cloud-name": "string",
      "cloud-provider": "string",
      "hostname": "string",
      "instance-id": "string",
      "instance-type": "string",
      "local-hostname": "string",
      "local-ipv4": "string",
      "public-keys": [
        "AAA[..snip...] user1@demo.openchami.cluster"
      ],
      "region": "string",
      "vendor-data": {
        "cabinet": "string",
        "cloud-init-base-url": "string",
        "cluster_name": "demo",
        "groups": {
          "compute": {
            "description": "The compute group"
          },
        },
        "location": "string",
        "nid": 0,
        "rack": "string",
        "role": "string",
        "sub-role": "string",
        "version": "string"
      }
    }
  },
  "instance-id": "string",
  "local-hostname": "compute-1"
}
```

## NODE USER-DATA

Node-specific user-data is used with the */cloud-init/impersonation/{id}/user-data* and */cloud-init/user-data* endpoints. In the OpenCHAMI cloud-init server, user-data is always empty and is not used.

For example, any node that requests its user-data will get the following back:

```
#cloud-config
```

## NODE VENDOR-DATA

Node-specific vendor-data is used with the
*/cloud-init/impersonation/{id}/vendor-data* and */cloud-init/vendor-data*
endpoints. In the OpenCHAMI cloud-init server, the purpose of vendor-data is to
generate a list of group data to include for the node.

For instance, if a node is a member of the *compute* and *slurm* groups and it
requests its vendor-data, the following cloud-init data is returned:

```
#include
http://172.16.0.254:8081/cloud-init/compute.yaml
http://172.16.0.254:8081/cloud-init/slurm.yaml
```

# GLOBAL FLAGS

The *cloud-init* command accepts the following global flags:

*--uri* _uri_
	Specify either the absolute base URI for the service (e.g.
	_https://foobar.openchami.cluster:8443/hsm/v2_) or a relative base path for
	the service (e.g. _/hsm/v2_). If an absolute URI is specified, this
	completely overrides any value set with the *--cluster-uri* flag or
	*cluster.uri* in the config file for the cluster. If using an absolute URI,
	it should contain the desired service's base path. If a relative path is
	specified (with or without the leading forward slash), then this value
	overrides the service's default base path and is appended to the cluster's
	base URI (set with the *--cluster-uri* flag or the *cluster.uri* cluster
	config option), which is required to be set if a relative path is used here.

	See *ochami*(1) for *--cluster-uri* and *ochami-config*(5) for details on
	cluster configuration options.

# COMMANDS

## defaults

Get and set cloud-init cluster-wide defaults. See *CLUSTER DEFAULTS* for details
on the data structure used with this command.

Subcommands for this command are as follows:

*get* [-F _format_]
	Get cluster-wide default meta-data.

	This command accepts the following options:

	*-F, --format-output* _format_
		Format the response output as _format_.

		Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

*set* [-f _format_] < _file_++
*set* [-f _format_] -d @_file_++
*set* [-f _format_] -d @- < _file_++
*set* [-f _format_] -d _data_
	Set cluster-wide default meta-data, overwriting and previously set values.

	In the first and third forms of the command, data is read from standard
	input.

	In the second form of the command, a file containing the payload data is
	passed.

	In the fourth form of the command, the payload is passed raw on the command
	line.

	This command sends a POST to the *cloud-init/admin/cluster-defaults*
	endpoint.

	This command accepts the following flags:

	*-d, --data* (_data_ | @_path_ | @-)
		Specify raw _data_ to send, the _path_ to a file to read payload data
		from, or to read the data from standard input (@-). The format of data
		read in any of these forms is JSON by default unless *-f* is specified
		to change it.

	*-f, --format-input* _format_
		Format of raw data being used by *-d* as the payload. Supported formats
		are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

## group

Get and manage cloud-init group data.

Subcommands for this command are as follows:

*add* [-f _format_] < _file_
*add* [-f _format_] -d @_file_++
*add* [-f _format_] -d @- < _file_++
*add* [-f _format_] -d _data_
	Add one or more new cloud-init groups. This command only accepts an array of
	group data (see *GROUP DATA*) and uses the *name* field to determine how to
	name the new group.

	In the first and third forms of the command, data is read from standard
	input.

	In the second form of the command, a file containing the payload data is
	passed.

	In the fourth form of the command, the payload is passed raw on the command
	line.

	This command sends a POST to the */cloud-init/admin/groups* endpoint.

	This command accepts the following flags:

	*-d, --data* (_data_ | @_path_ | @-)
		Specify raw _data_ to send, the _path_ to a file to read payload data
		from, or to read the data from standard input (@-). The format of data
		read in any of these forms is JSON by default unless *-f* is specified
		to change it.

	*-f, --format-input* _format_
		Format of raw data being used by *-d* as the payload. Supported formats
		are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

*delete* [--force] _id_...++
*delete* [--force] [-f _format_] -d @_file_++
*delete* [--force] [-f _format_] -d @- < _file_++
*delete* [--force] [-f _format_] -d _data_
	Delete one or more cloud-init groups, identified by one or more _id_
	arguments or *name* fields in payload data.

	In the first form of the command, the groups to delete are specified by
	their names on the command line.

	In the second form of the command, a file containing the payload data is
	passed.

	In the third form of the command, the payload data is read from standard
	input.

	In the fourth form of the command, the payload is passed raw on the command
	line.

	This command sends one or more DELETE requests to the
	*/cloud-init/admin/groups* endpoint.

	This command accepts the following flags:

	*-d, --data* (_data_ | @_path_ | @-)
		Specify raw _data_ to send, the _path_ to a file to read payload data
		from, or to read the data from standard input (@-). The format of data
		read in any of these forms is JSON by default unless *-f* is specified
		to change it.

	*--force*
		Do not ask the user to confirm deletion. Use with caution.

	*-f, --format-input* _format_
		Format of raw data being used by *-d* as the payload. Supported formats
		are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

*get*
	Get cloud-init group data.

	Commands under this one send one or more GET requests to the
	*/cloud-init/admin/groups* or */cloud-init/admin/groups/{id}* endpoints.

	This command has the following subcommands:

	*config* [--header | --no-header] [_id_...]
		Print the cloud-init config file for one or more groups, identified by
		_id_. If none are passed, the configurations for all known groups are
		printed.

		If more than one is printed, a header is printed for each that
		identifies which ID each belongs to, as well as how many configs are
		being printed. This behavior can be modified with the *--header* and
		*--no-header* flags. An example of the header is:

		```
		(1/5) group=compute
		```

		*1* is the index being printed, *5* is the total that are being printed,
		and *compute* is the id of the group being printed.

		This command accepts the following flags:

		*--header*
			Always print a header for each config returned, even if only one is
			printed.

		*--no-header*
			Never print a header for each config returned, even if more than one
			are printed.

	*meta-data* [-F _format_] [_id_...]
		Print the meta-data keys and values for one or more groups, identified
		by _id_. If none are passed, the meta-data for all known groups is
		printed.

		The output is an array of objects each with two keys: *name* and
		*meta-data*, for example, in JSON format:

		```
		[
			{
				"name": "group1",
				"meta-data": {
					"foo": "bar"
				}
			},
			...
		]
		```

		This command accepts the following flags:

		*-F, --format-output* _format_
			Format the response output as _format_.

			Supported values are:

			- _json_ (default)
			- _json-pretty_
			- _yaml_

	*raw* [-F _format_] [_id_...]
		Print the raw group data for one or more groups, identified by _id_. If
		none are passed, the raw data for all known groups is printed. The data
		returned is an array of group data objects (see *GROUP DATA*).

		This command accepts the following flags:

		*-F, --format-output* _format_
			Format the response output as _format_.

			Supported values are:

			- _json_ (default)
			- _json-pretty_
			- _yaml_

*render* _group_ _id_
	Print the cloud-init group configuration for _group_, impersonating node
	_id_, populating Jinja2 variables. Node _id_ must be a member of group
	_group_. This command is similar to the *cloud-init get config* command
	except that (1) Jinja2 variables are populated and (2) only one config at a
	time can be printed.

	This command works by fetching the group config for _group_ and _id_
	(*cloud-init node get group*), fetching the meta-data for _id_ (*cloud-init
	node get meta-data*), then using the meta-data to render the group config
	locally. Note that this command only renders the group configuration for a
	node and does not go through cloud-init's full render process.

	This command is meant as a troubleshooting tool.

	This command sends GET requests to the following cloud-init endpoints:

	- */cloud-init/admin/impersonation/{id}/{group}.yaml*
	- */cloud-init/admin/impersonation/{id}/meta-data*

*set* [-f _format_] < _file_++
*set* [-f _format_] -d @_file_++
*set* [-f _format_] -d @- < _file_++
*set* [-f _format_] -d _data_
	Set cloud-init group data for one or more groups, creating the group if
	non-existent or overwriting group data if the group exists. This command
	only accepts an array of group data (see *GROUP DATA*) and uses the *name*
	field to determine which groups to update.

	In the first and third forms of the command, data is read from standard
	input.

	In the second form of the command, a file containing the payload data is
	passed. This is convenient for dealing with many cloud-init configurations
	at once.

	In the fourth form of the command, the payload is passed raw on the command
	line. This data is passed raw to the server.

	This command sends a PUT to the */cloud-init/admin/groups* endpoint.

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
		- _json-pretty_
		- _yaml_

## node

Get and manage cloud-init node data.

Subcommands for this command are as follows:

*get*
	Get cloud-init node data. This command has the following subcommands:

	*group* [--header | --no-header] _id_ _group_...
		Print the cloud-init group data for node _id_ for each group _group_
		that it is a member of.

		If more than one is printed, a header is printed for each that
		identifies which group and node ID combination is being printed, as well
		as how many configs are being printed. This behavior can be modified
		with the *--header* and *--no-header* flags. An example of the header
		is:

		```
		--- (1/5) node=x3000c0s0b0n0 group=compute
		```

		*1* is the index being printed and *5* is the total that are being
		printed. *node=x3000c0s0b0n0 group=compute* means that the config being
		printed is the group *compute* config for node *x3000c0s0b0n0*.

		This command accepts the following flags:

		*--header*
			Always print a header for each config returned, even if only one is
			printed.

		*--no-header*
			Never print a header for each config returned, even if more than one
			are printed.

	*meta-data* [-F _format_] _id_...
		Print the meta-data keys and values for one or more nodes, identified by
		_id_. At least one _id_ is required. The result of this command is an
		array of node meta-data structures (see *NODE META-DATA*).

		This command sends a GET to the
		*/cloud-init/admin/impersonation/{id}/meta-data* endpoint.

		This command accepts the following flags:

		*-F, --format-output* _format_
			Format the response output as _format_.

			Supported values are:

			- _json_ (default)
			- _json-pretty_
			- _yaml_

	*user-data* [--header | --no-header] _id_...
		Print the user-data for one or more nodes, identified by _id_. At least
		one _id_ is required. The result of this command is cloud-init user-data
		(see *NODE USER-DATA*).

		If more than one is printed, a header is printed for each that
		identifies which ID each belongs to, as well as how many configs are
		being printed. This behavior can be modified with the *--header* and
		*--no-header* flags. An example of the header is:

		```
		(1/5) node=x3000c0s0b0n0
		```

		*1* is the index being printed, *5* is the total that are being printed,
		and *x3000c0s0b0n0* is the id of the node being printed.

		This command accepts the following flags:

		*--header*
			Always print a header for each config returned, even if only one is
			printed.

		*--no-header*
			Never print a header for each config returned, even if more than one
			are printed.

	*vendor-data* [--header | --no-header] _id_...
		Print the vendor-data for one or more nodes, identified by _id_. At
		least one _id_ is required. The result of this command is cloud-init
		user-data (see *NODE VENDOR-DATA*).

		If more than one is printed, a header is printed for each that
		identifies which ID each belongs to, as well as how many configs are
		being printed. This behavior can be modified with the *--header* and
		*--no-header* flags. An example of the header is:

		```
		(1/5) node=x3000c0s0b0n0
		```

		*1* is the index being printed, *5* is the total that are being printed,
		and *x3000c0s0b0n0* is the id of the node being printed.

		This command accepts the following flags:

		*--header*
			Always print a header for each config returned, even if only one is
			printed.

		*--no-header*
			Never print a header for each config returned, even if more than one
			are printed.

*set* [-f _format_] < _file_++
*set* [-f _format_] -d @_file_++
*set* [-f _format_] -d @- < _file_++
*set* [-f _format_] -d _data_
	Set node-specific meta-data for one or more nodes. This command only accepts
	an array of instance info (see *INSTANCE INFO*) and uses the *id* field to
	determine which nodes whose data to set.

	In the first and third forms of the command, data is read from standard
	input.

	In the second form of the command, a file containing the payload data is
	passed. This is convenient for dealing with many cloud-init node instance
	info at once.

	In the fourth form of the command, the payload is passed raw on the command
	line. This data is passed raw to the server.

	This command sends a PUT to the */cloud-init/admin/instance-info/{id}*
	endpoint for each *{id}*.

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
		- _json-pretty_
		- _yaml_

# AUTHOR

Written by Devon T. Bautista and maintained by the OpenCHAMI developers.

# SEE ALSO

*ochami*(1)

; Vim modeline settings
; vim: set tw=80 noet sts=4 ts=4 sw=4 syntax=scdoc:
