OCHAMI-BSS(1) "OpenCHAMI" "Manual Page for ochami-bss"

# NAME

ochami-bss - Communicate with the Boot Script Service (BSS)

# SYNOPSIS

ochami bss [OPTIONS] COMMAND

# DATA STRUCTURE

The data structure for sending and receiving data with subcommands under the
*bss* command is (in JSON form):

```
{
  "kernel": "https://example.com/kernel",
  "initrd": "https://example.com/initrd",
  "params": "quiet nosplash"
}
```

# COMMANDS

## boot params

Manage boot parameters for components.

Subcommands for this command are as follows:

*add* ([--mac _mac_,...] [--nid _nid_,...] [--xname _xname_,...]) ([--initrd _initrd_] [--kernel _kernel_])++
*add* -f _file_ [-F _format_]++
*add* -f _-_ [-F _format_] < _file_
	Add new boot parameters for one or more components. If boot parameters
	already exist for the specified components, this command will fail.

	In the first form of the command, one or more of *--mac*, *--nid*, or
	*--xname* is required to identify which component(s) to add boot config for.
	One or more of *--initrd*, *--kernel*, or *--params* is also required to
	know which boot parameters to add for the specified components.  For any of
	these options, multiple arguments can be passed either by specifying the
	flag multiple times (e.g. *--mac* _mac1_ *--mac* _mac2_) or by using one
	flag and separating each argument by commas (e.g. *--mac* _mac1_,_mac2_).

	In the second form of the command, a file containing the payload data is
	passed. This is convenient in cases of dealing with many components at once.

	In the third form of the command, the payload data is read from standard
	input.

	This command sends a POST request to BSS's /bootparameters endpoint.

	This command accepts the following options:

	*-f, --payload* _file_
		Specify a file containing the data to send to BSS. The format of this
		file depends on _-F_ and is _json_ by default. If *-* is used as the
		argument to _-f_, the command reads the payload data from standard
		input.

	*-F, --payload-format* _format_
		Format of the file used with _-f_. Supported formats are:

		- _json_ (default)
		- _yaml_

	*-m, --mac* _mac_addr_,...
		One or more MAC addresses to add boot parameters for. For multiple MAC
		addresses, either this flag can be specified multiple times or this flag
		can be specified once and multiple MAC addresses can be specified,
		separated by commas.

	*-n, --nid* _nid_,...
		One or more node IDs to add boot parameters for. For multiple NIDs,
		either this flag can be specified multiple times or this flag can be
		specified once and multiple NIDs can be specified, separated by commas.

	*-x, --xname* _xname_,...
		One or more xnames to add boot parameters for. For multiple xnames,
		either this flag can be specified multiple times or this flag can be
		specified once and multiple xnames, separated by commas.

	*--initrd* _initrd_uri_
		URI from which to fetch the components' initrd.

	*--kernel* _kernel_uri_
		URI from which to fetch the components' kernel.

	*--params* _kernel_params_
		Command line arguments to pass to kernel for components.

*delete* [--force] ([--mac, _mac_,...] [--nid, _nid_,...] [--xname _xname_,...] [--kernel _kernel_] [--initrd _initrd_])++
*delete* [--force] -f _file_ [-F _format_]++
*delete* [--force] -f _-_ [-F _format_]
	Delete boot parameters for one or more components. Which boot parameters are
	deleted are determined by passed filters, which can be passed via CLI flag
	or within a payload file. Unless *--force* is passed, the user is asked to
	confirm deletion.

	In the first form of the command, one or more of *--mac*, *--nid*,
	*--xname*, *--kernel*, or *--initrd* is required to identify which
	component(s) whose boot parameters to delete. For any of these options,
	multiple arguments can be passed either by specifying the flag multiple
	times (e.g. *--mac* _mac1_ *--mac* _mac2_) or by using one flag and
	separating each argument by commas (e.g. *--mac* _mac1_,_mac2_).

	In the second form of the command, a file containing the payload data is
	passed. This is convenient in cases of dealing with many components at once.

	In the third form of the command, the payload data is read from standard
	input.

	This command sends a DELETE request to BSS's /bootparameters endoint.

	This command accepts the following options:

	*--force*
		Do not ask the user to confirm deletion. Use with caution.

	*-f, --payload* _file_
		Specify a file containing the data to send to BSS. The format of this
		file depends on _-F_ and is _json_ by default. If *-* is used as the
		argument to _-f_, the command reads the payload data from standard
		input.

	*-F, --payload-format* _format_
		Format of the file used with _-f_. Supported formats are:

		- _json_ (default)
		- _yaml_

	*-m, --mac* _mac_addr_,...
		One or more MAC addresses to delete boot parameters for. For multiple
		MAC addresses, either this flag can be specified multiple times or this
		flag can be specified once and multiple MAC addresses can be specified,
		separated by commas.

	*-n, --nid* _nid_,...
		One or more node IDs to delete boot parameters for. For multiple NIDs,
		either this flag can be specified multiple times or this flag can be
		specified once and multiple NIDs can be specified, separated by commas.

	*-x, --xname* _xname_,...
		One or more xnames to delete boot parameters for. For multiple xnames,
		either this flag can be specified multiple times or this flag can be
		specified once and multiple xnames, separated by commas.

	*--initrd* _initrd_uri_
		URI from which to fetch the components' initrd.

	*--kernel* _kernel_uri_
		URI from which to fetch the components' kernel.

	*--params* _kernel_params_
		Command line arguments to pass to kernel for components.

*get* [--output-format _format_] [--mac _mac_,...] [--nid _nid_,...] [--xname _xname_,...]
	Get boot parameters for all components or a subset of components, filtered
	by MAC address, node ID, and/or xname.

	This command sends a GET to BSS's /bootparameters endpoint.

	This command accepts the following options:

	*-F, --output-format* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _yaml_

	*-m, --mac* _mac_addr_,...
		One or more MAC addresses to filter boot parameters by. For multiple MAC
		addresses, either this flag can be specified multiple times or this flag
		can be specified once and multiple MAC addresses can be specified,
		separated by commas.

	*-n, --nid* _nid_,...
		One or more node IDs to filter boot parameters by. For multiple NIDs,
		either this flag can be specified multiple times or this flag can be
		specified once and multiple NIDs can be specified, separated by commas.

	*-x, --xname* _xname_,...
		One or more xnames to filter boot parameters by. For multiple xnames,
		either this flag can be specified multiple times or this flag can be
		specified once and multiple xnames, separated by commas.

*set* ([--mac _mac_,...] [--nid _nid_,...] [--xname _xname_,...]) ([--initrd _initrd_] [--kernel _kernel_])++
*set* -f _file_ [-F _format_]++
*set* -f _-_ [-F _format_] < _file_
	Set boot parameters for one or more components, even if boot parameters
	already exist for said components. This is handy if one knows what boot
	parameters to set for which components, but isn't sure if boot parameters
	have already been set for one or more of them.

	In the first form of the command, one or more of *--mac*, *--nid*, or
	*--xname* is required to identify which component(s) to set boot config for.
	One or more of *--initrd*, *--kernel*, or *--params* is also required to
	know which boot parameters to set for the specified components.  For any of
	these options, multiple arguments can be passed either by specifying the
	flag multiple times (e.g. *--mac* _mac1_ *--mac* _mac2_) or by using one
	flag and separating each argument by commas (e.g. *--mac* _mac1_,_mac2_).

	In the second form of the command, a file containing the payload data is
	passed. This is convenient in cases of dealing with many components at once.

	In the third form of the command, the payload data is read from standard
	input.

	This command sends a PUT request to BSS's /bootparameters endpoint.

	This command accepts the following options:

	*-f, --payload* _file_
		Specify a file containing the data to send to BSS. The format of this
		file depends on _-F_ and is _json_ by default. If *-* is used as the
		argument to _-f_, the command reads the payload data from standard
		input.

	*-F, --payload-format* _format_
		Format of the file used with _-f_. If unspecified, the payload format is
		_json_ by default. Supported formats are:

		- _yaml_

	*-m, --mac* _mac_addr_,...
		One or more MAC addresses to set boot parameters for. For multiple MAC
		addresses, either this flag can be specified multiple times or this flag
		can be specified once and multiple MAC addresses can be specified,
		separated by commas.

	*-n, --nid* _nid_,...
		One or more node IDs to set boot parameters for. For multiple NIDs,
		either this flag can be specified multiple times or this flag can be
		specified once and multiple NIDs can be specified, separated by commas.

	*-x, --xname* _xname_,...
		One or more xnames to set boot parameters for. For multiple xnames,
		either this flag can be specified multiple times or this flag can be
		specified once and multiple xnames, separated by commas.

	*--initrd* _initrd_uri_
		URI from which to fetch the components' initrd.

	*--kernel* _kernel_uri_
		URI from which to fetch the components' kernel.

	*--params* _kernel_params_
		Command line arguments to pass to kernel for components.

*update* ([--mac _mac_,...] [--nid _nid_,...] [--xname _xname_,...]) ([--initrd _initrd_] [--kernel _kernel_])++
*update* -f _file_ [-F _format_]++
*update* -f _-_ [-F _format_] < _file_
	Update boot parameters for existing components.

	In the first form of the command, one or more of *--mac*, *--nid*, or
	*--xname* is required to identify which component(s) to update boot config
	for. One or more of *--initrd*, *--kernel*, or *--params* is also required
	to know which boot parameters to update for the specified components.  For
	any of these options, multiple arguments can be passed either by specifying
	the flag multiple times (e.g. *--mac* _mac1_ *--mac* _mac2_) or by using one
	flag and separating each argument by commas (e.g. *--mac* _mac1_,_mac2_).

	In the second form of the command, a file containing the payload data is
	passed. This is convenient in cases of dealing with many components at once.

	In the third form of the command, the payload data is read from standard
	input.

	This command sends a PUT request to BSS's /bootparameters endpoint.

	This command accepts the following options:

	*-f, --payload* _file_
		Specify a file containing the data to send to BSS. The format of this
		file depends on _-F_ and is _json_ by default. If *-* is used as the
		argument to _-f_, the command reads the payload data from standard
		input.

	*-F, --payload-format* _format_
		Format of the file used with _-f_. If unspecified, the payload format is
		_json_ by default. Supported formats are:

		- _yaml_

	*-m, --mac* _mac_addr_,...
		One or more MAC addresses to update boot parameters for. For multiple
		MAC addresses, either this flag can be specified multiple times or this
		flag can be specified once and multiple MAC addresses can be specified,
		separated by commas.

	*-n, --nid* _nid_,...
		One or more node IDs to update boot parameters for. For multiple NIDs,
		either this flag can be specified multiple times or this flag can be
		specified once and multiple NIDs can be specified, separated by commas.

	*-x, --xname* _xname_,...
		One or more xnames to update boot parameters for. For multiple xnames,
		either this flag can be specified multiple times or this flag can be
		specified once and multiple xnames, separated by commas.

	*--initrd* _initrd_uri_
		URI from which to fetch the components' initrd.

	*--kernel* _kernel_uri_
		URI from which to fetch the components' kernel.

	*--params* _kernel_params_
		Command line arguments to pass to kernel for components.

## boot script

Manage boot scripts for components.

Subcommands for this command are as follows:

*get* ([--mac _mac_] [--nid _nid_] [--xname _xname_])
	Get the iPXE boot script for a component. Exactly one of *--mac*, *--nid*,
	or *--xname* is required to specify the component whose boot script to get.
	Note that only *one* component's boot script is fetched.

	This command sends a GET to BSS's /bootscript endpoint.

	This command accepts the following options:

	*-m, --mac* _mac_addr_
		MAC address corresponding to component whose boot script to get.

	*-n, --nid* _nid_
		Node ID corresponding to component whose boot script to get.

	*-x, --xname* _xname_
		Xname corresponding to component whose boot script to get.

## dumpstate

Dump internal state of BSS for debugging purposes. Return known hosts and
associated information, along with the known boot parameter info. The format of
the output is similar to the boot parameters struct above with the addition of a
components list.

The format of this command is:

*dumpstate* [--output-format _format_]

This command sends a GET to BSS's /dumpstate endpoint.

This command accepts the following options:

*-F, --output-format* _format_
	Output response data in specified _format_. Supported values are:

	- _json_ (default)
	- _yaml_

## history

Print endpoint access history. This command outputs a list of logs of accesses
to BSS endpoints with UNIX timestamps. Output can be filtered by component name
(xname) that made the access and/or the BSS endpoint accessed.

The format of the command is:

*history* [--output-format _format_] [--xname _xname_,...] [--endpoint _endpoint_,...]

This command sends a GET to BSS's /endpoint-history endpoint.

This command accepts the following options:

*-F, --output-format* _format_
	Output response data in specified _format_. Supported values are:

	- _json_ (default)
	- _yaml_

*--xname* _xname_,...
	One or more xnames to filter endpoint history results by. For multiple
	xnames, either this flag can be specified multiple times or this flag can be
	specified once and multiple xnames, separated by commas.

*--endpoint* _endpoint_,...
	One or more endpoint names (e.g. _bootscript_, _bootparameters_) to filter
	endpoint history results by. For multiple endpoints, either this flag can be
	specified multiple times or this flag can be specified once and multiple
	endpoints, separated by commas.

## hosts

Work with hosts in BSS.

Subcommands for this command are as follows:

*get* [--output-format _format_ ] [--mac _mac_,...] [--nid _nid_,...] [--xname _xname_,...]
	Get a list of hosts that BSS knows about that are in SMD. These results can
	be optionally filtered by MAC address, node ID, or xname. If no filters are
	specified, all results are returned.

	This command sends a GET to BSS's /hosts endpoint.

	This command accepts the following options:

	*-F, --output-format* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _yaml_

	*-m, --mac* _mac_addr_,...
		One or more MAC addresses to filter results by. For multiple MAC
		addresses, either this flag can be specified multiple times or this flag
		can be specified once and multiple MAC addresses can be specified,
		separated by commas.

	*-n, --nid* _nid_,...
		One or more node IDs to filter results by. For multiple NIDs, either
		this flag can be specified multiple times or this flag can be specified
		once and multiple NIDs can be specified, separated by commas.

	*-x, --xname* _xname_,...
		One or more xnames to filter results by. For multiple xnames, either
		this flag can be specified multiple times or this flag can be specified
		once and multiple xnames, separated by commas.

## status

Get BSS's status. This is useful for checking if BSS is running, if it is
connected to SMD, or checking the storage backend type/connection status.

The format of this command is:

*status* [--output-format _format_] [--all | --smd | --storage | --version]

This command sends a GET to endpoints under BSS's /service endpoint.

This command accepts the following options:

*--all*
	Print out all of the status information BSS knows about.

*-F, --output-format* _format_
	Output response data in specified _format_. Supported values are:

	- _json_ (default)
	- _yaml_

*--smd*
	Print out the status of BSS's connection to SMD.

*--storage*
	Print out the backend storage type and connection status of BSS to that
	storage backend.

*--version*
	Print out BSS's version.

# AUTHOR

Written by Devon T. Bautista and maintained by the OpenCHAMI developers.

; Vim modeline settings
; vim: set tw=80 noet sts=4 ts=4 sw=4 syntax=scdoc:
