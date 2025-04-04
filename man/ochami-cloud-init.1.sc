OCHAMI-CLOUD-INIT(1) "OpenCHAMI" "Manual Page for ochami-cloud-init"

# NAME

ochami-cloud-init - Communicate with the cloud-init server

# SYNOPSIS

ochami cloud-init [--secure] config add [OPTIONS] -d (_payload_data_ | @_payload_file_ | @-)++
ochami cloud-init [--secure] config delete [OPTIONS] _id_...++
ochami cloud-init [--secure] config get [OPTIONS] [_id_...]++
ochami cloud-init [--secure] config add [OPTIONS] -d (_payload_data_ | @_payload_file_ | @-)++
ochami cloud-init [--secure] data get [OPTIONS] _id_...

# DATA STRUCTURE

An example of the data structure for sending and receiving data with subcommands
under the *cloud-init* command is (in JSON form):

```
[
  "name": "compute",
  "compute": {
    "cloud-init": {
      "metadata": {
        "instance-id": "ochami-compute"
      },
      "userdata": {
        "runcmd": [
          "echo hello",
        ],
        "ssh_deletekeys": false,
        "write_files": [
          {
            "content": "aGVsbG8K",
            "encoding": "base64",
            "path": "/opt/test"
          },
          {
            "content": "SLURMD_OPTIONS=--conf-server 172.16.0.254:6817\n",
            "path": "/etc/sysconfig/slurmd"
          },
          }
        ]
      },
      "vendordata": null
    }
  },
  ...
]
```

## GLOBAL FLAGS

The *cloud-init* command accepts the following global flags:

*--secure*
	Use the secure cloud-init endpoint instead of the open one. A token is
	required.

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

## config

Get and manage cloud-init configurations. Configuration tells cloud-init which
data to serve to which clients.

Subcommands for this command are as follows:

*add* -d @_file_ [-f _format_]++
*add* -d @- [-f _format_] < _file_++
*add* -d _data_
	Add cloud-init configuration for one or more IDs. This command only accepts
	payload data and uses the *name* field to determine which ID to add the data
	for.

	In the first form of the command, a file containing the payload data is
	passed. This is convenient for dealing with many cloud-init configurations
	at once.

	In the second form of the command, the payload data is read from standard
	input.

	In the third form of the command, the payload is passed raw on the command
	line. This data is passed raw to the server.

	This command sends a POST to the /cloud-init endpoint, or /cloud-init-secure
	if *--secure* is passed.

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

*delete* [--force] _id_...
	Delete one or more cloud-init configurations, identified by _id_.

	This command sends one or more DELETE requests to the /cloud-init endpoint,
	or /cloud-init-secure if *--secure* is passed.

	This command accepts the following flags:

	*--force*
		Do not ask the user to confirm deletion. Use with caution.

*get* [--format-output _format_] [_id_...]
	Get cloud-init configuration for one or more _id_. If no IDs are specified,
	all cloud-init configurations are retrieved.

	This command sends a GET request to the /cloud-init endpoint, or
	/cloud-init-secure if *--secure* is passed.

	This command accepts the following options:

	*-F, --format-output* _format_
		Format the response output as _format_.

		Supported values are:

		- _json_ (default)
		- _yaml_

*update* -d @_file_ [-f _format_]++
*update* -d @- [-f _format_] < _file_++
*update* -d _data_
	Update one or more existing cloud-init configurations. This command only
	accepts payload data and uses the *name* field to determine which ID to
	update.

	In the first form of the command, a file containing the payload data is
	passed. This is convenient for dealing with many cloud-init configurations
	at once.

	In the second form of the command, the payload data is read from standard
	input.

	In the third form of the command, the payload is passed raw on the command
	line. This data is passed raw to the server.

	This command sends a PUT to the /cloud-init endpoint, or /cloud-init-secure
	if *--secure* is passed.

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

## data

View cloud-init data. cloud-init data is the raw data that is received by a
client when requesting its data. There are three types of data: *user-data*,
*meta-data*, and *vendor-data*.

Subcommands for this command are as follows:

*get* [--meta | --user | --vendor] _id_...
	Get cloud-init data for one or more _id_. By default, or if *--user* is
	passed, cloud-init user-data is retrieved.

	This command accepts the following options:

	*--meta*
		Fetch cloud-init meta-data.

	*--user*
		Fetch cloud-init user-data.

	*--vendor*
		Fetch cloud-init vendor-data

# AUTHOR

Written by Devon T. Bautista and maintained by the OpenCHAMI developers.

# SEE ALSO

*ochami*(1)

; Vim modeline settings
; vim: set tw=80 noet sts=4 ts=4 sw=4 syntax=scdoc:
