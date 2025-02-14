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

# COMMANDS

## status

Get PCS's status. This is useful for checking if PCS is running, if it is
connected to SMD, or checking the storage backend connection status.

The format of this command is:

*status* [--output-format _format_] [--all | --smd | --storage | --vault]

This command sends a GET to PCS's /readiness or /health endpoints.

This command accepts the following options:

*--all*
	Print out all of the status information PCS knows about.

*-F, --output-format* _format_
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

; Vim modeline settings
; vim: set tw=80 noet sts=4 ts=4 sw=4 syntax=scdoc:
