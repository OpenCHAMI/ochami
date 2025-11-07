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

## service

Manage and check PCS itself.

Subcommands for this command are as follows:

*status* [-F _format_] [--all | --smd | --storage | --vault]
	Send a GET to PCS's /readiness or /health endpoints.

	This command accepts the following options:

	*--all*
		Print out all of the status information PCS knows about.

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

	*--smd*
		Print out the status of PCS's connection to SMD.

	*--storage*
		Print out the backend storage connection status of PCS.

	*--vault*
		Print out the backend vault connection status of PCS.

## status

Display power status.

Subcommands for this command are as follows:

*show* [-F _format_] _xname_
	Show the power status of a component.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

	*xname*
		XName of the component to show the power status for.

*list* [-F _format_]
	List the power status of all components.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

	*-x, --xname* _xname_,...
		Comma-separated list of xnames to show power status for. If not
		specified, status for all components will be shown.

	*-p, --power-filter* _state_
		power state filter. Supported values are:

		- _on_
		- _off_
		- _undefined_

	*-m, --mgmt-filter* _state_
		management state filter. Supported values are:

		- _available_
		- _unavailable_

## transitions

Manages PCS transitions.

Subcommands for this command are as follows:

*start*  [-F _format_] [-x _xname1,xname2,..._]... _operation_
	Starts a power transition on one or more nodes.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

	*-x, --xname* _xname_,...
		Comma-separated list of xnames to transition.

	*operation*
		Operation to perform. Supported operations are:

		- _on_
		- _off_
		- _soft-restart_
		- _hard-restart_
		- _init_
		- _force-off_
		- _soft-off_

*list* [-F _format_]
	List the active power transitions.

	This command accepts the following options:

		*-F, --format-output* _format_
			Output response data in specified _format_. Supported values are:

			- _json_ (default)
			- _json-pretty_
			- _yaml_

*show* [-F _format_] _id_
	Show the details of a power transition.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

	*id*
		ID of the power transition to show.


*monitor* _id_
	Monitor active power transitions and provide progress information

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

	*id*
		ID of the power transition to monitor.

*abort* [-F _format_] _id_
	Abort or terminate an active power transition.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

	*id*
		ID of the power transition to abort.


# AUTHOR

Written by Chris Harris and maintained by the OpenCHAMI developers.

# SEE ALSO

*ochami*(1)

; Vim modeline settings
; vim: set tw=80 noet sts=4 ts=4 sw=4 syntax=scdoc:
