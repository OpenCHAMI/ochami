OCHAMI-RCS(1) "OpenCHAMI" "Manual Page for ochami-rcs"

# NAME

ochami-rcs - Communicate with the remote-console service

# SYNOPSIS

ochami rcs [OPTIONS] COMMAND

# GLOBAL FLAGS

*--uri* _uri_
	Specify either the absolute base URI for the remote-console service (e.g.
	_https://foobar.openchami.cluster:8443/remote-console_) or a relative base
	path for the service (e.g. _/remote-console_). If an absolute URI is
	specified, this completely overrides any value set with the *--cluster-uri*
	flag or *cluster.rcs.uri* in the config file for the cluster. If using an
	absolute URI, it should contain the desired service's base path. If a
	relative path is specified (with or without the leading forward slash), then
	this value overrides the service's default base path and is appended to the
	cluster's base URI (set with the *--cluster-uri* flag or *cluster.uri* in the
	config file), which is required to be set if a relative path is used here.

	See *ochami*(1) for *--cluster-uri* and *ochami-config*(5) for details on
	cluster configuration options.

# COMMANDS

## service

Check the remote-console service itself.

Subcommands for this command are as follows:

*status* [-F _format_]
	Returns the status of the remote-console service.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

## console

Manage remote console sessions.

Subcommands for this command are as follows:

*list* [-F _format_]
	List available consoles.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

*show* [-F _format_] [--follow] [--lines _n_] _nodeID_
	Show console output for the specified node.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

	*-f, --follow*
		Follow the console output in real-time.

	*--lines* _n_
		Number of lines to show from history. Defaults to 100.

	*nodeID*
		Node ID of the console to show.

*connect* _nodeID_
	Start an interactive session with the console of the specified node.

	*nodeID*
		Node ID of the console to connect to.

# AUTHOR

Written by Chris Harris and maintained by the OpenCHAMI developers.

# SEE ALSO

*ochami*(1)

; Vim modeline settings
; vim: set tw=80 noet sts=4 ts=4 sw=4 syntax=scdoc:
