OCHAMI-PCS(1) "OpenCHAMI" "Manual Page for ochami-pcs"

# NAME

ochami-pcs - Communicate with the Power Control Service (PCS)

# SYNOPSIS

ochami pcs [OPTIONS] COMMAND

# DATA STRUCTURE

The data structure for sending and receiving data with subcommands under the
*pcs* command is (in JSON form):

```
{
  "pcs": "ready",
  "storage": "connected, responsive",
  "smd": "connected, responsive",
  "vault": "connected, responsive"
}
```

# GLOBAL FLAGS

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

## status

Get PCS's status. This is useful for checking if PCS is running, if it is
connected to SMD, or checking the storage backend connection status.

The format of this command is:

*status* [-F _format_] [--all | --smd | --storage | --vault]

This command sends a GET to PCS's /readiness or /health endpoints.

This command accepts the following options:

*--all*
	Print out all of the status information PCS knows about.

*-F, --format-output* _format_
	Output response data in specified _format_. Supported values are:

	- _json_ (default)
	- _yaml_

*--smd*
	Print out the status of PCS's connection to SMD.

*--storage*
	Print out the backend storage connection status of PCS.

*--vault*
	Print out the backend vault connection status of PCS.

# AUTHOR

Written by Chris Harris and maintained by the OpenCHAMI developers.

# SEE ALSO

*ochami*(1)

; Vim modeline settings
; vim: set tw=80 noet sts=4 ts=4 sw=4 syntax=scdoc:
