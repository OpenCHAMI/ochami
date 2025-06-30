OCHAMI-DISCOVER(1) "OpenCHAMI" "Manual Page for ochami-discover"

# NAME

ochami-discover - Populate SMD using a file

# SYNOPSIS

ochami discover static [--overwrite] [-d (_data_ | @_path_)] [-f _format_]

# DESCRIPTION

The *discover* command is a metacommand for discovering nodes and populating SMD
with the data.

# DATA STRUCTURE

The format of the data for *static* discovery is a *nodes* object containing an
array of node data.  An example of such an array containing one node in YAML
format is as follows:

```
nodes:
- name: node01
	nid: 1
	xname: x1000c1s7b0n0
	bmc_mac: de:ca:fc:0f:ee:ee
	bmc_ip: 172.16.0.101
	group: compute
	interfaces:
	- mac_addr: de:ad:be:ee:ee:f1
		ip_addrs:
		- name: internal
			ip_addr: 172.16.0.1
	- mac_addr: de:ad:be:ee:ee:f2
		ip_addrs:
		- name: external
			ip_addr: 10.15.3.100
	- mac_addr: 02:00:00:91:31:b3
		ip_addrs:
		- name: HSN
			ip_addr: 192.168.0.1
```

A description of each key in the above is as follows:

- *name* - A short name identifying the node. This is used as the
RedfishEndpoint name and is used to generate a short description of the node for
the description field in EthernetInterfaces it creates.
- *nid* - The node ID number unique to the node. This used in the NID field in
the Component that is created for the node.
- *xname* - The xname unique to the node. It is important that this is a node
xname (see *XNAMES*) because this is used to calculate a BMC xname for the
RedfishEndpoint and Component structures created for the BMC for the node. This
is used as the unique identifier for the node within the Component that gets
created for node.
- *bmc_mac* - MAC address of node's BMC.
- *bmc_ip* - Desired IP address of node's BMC.
- *group* - Optional group to add node to. This will get created during
discovery if it does not exist.
- *interfaces* - A list of network interfaces for the node.
	- *mac_addr* - MAC address of network interface.
	- *ip_addrs* - List of IP addresses assigned to interface.
		- *name* - Short name identifying the network for the IP address.
		- *ip_addr* - IP address for interface.

# COMMANDS

## static

Populate SMD using static data from a file or standard input.

The format of this command is:

*static* [--overwrite] [-d (_data_ | @_path_)] [-f _format_]

The *static* subcommand provides a way to use structured data (from standard
input or a file) to emulate the SMD discovery process in a reproducable way
without performing dynamic discovery via Redfish. Sometimes, discovery via
Redfish may not be possible or feasible using dynamic methods (e.g. no Redfish
support), or storing node data in a user-friendly file that can be used to
populate SMD is preferred.

If *-d* is not specified, then the data is read from standard input. The format
of the input data is JSON by default, but *-f* can be used to specify a
different format.

The data should contain a list of "nodes", each with its own configuration (see
*DATA STRUCTURE*). The *static* command reads this data and creates the SMD
RedfishEndpoints, EthernetInterfaces, Components, and groups data in SMD
corresponding to each node. It also creates Components corresponding to each
node's BMC which corresponds to each RedfishEndpoint created.

The *--discovery-version* sets which discovery method to use when running the
*static* subcommand. If the version is set to 1, an additional request is made
to create the EthernetInterfaces separately in SMD. If set to 2 (the default),
the EthernetInterfaces are created with the first discovery request.
This flag is only for backward compatibility with earlier versions of SMD and
may be deprecated in a later version of ochami.
This command accepts the following options:

*-d, --data* (_data_ | @_path_ | @-)
	Specify raw _data_ to send, the _path_ to a file to read payload data from,
	or to read the data from standard input (@-). The format of data read in any
	of these forms is JSON by default unless *-f* is specified to change it.

*-f, --format-input* _format_
	Format of the input data. If unspecified, the payload format is _json_ by
	default. Supported formats are:

	- _json_ (default)
	- _json-pretty_
	- _yaml_

*--overwrite*
	Instead of failing if data already exists, overwrite it with new data
	contained in the payload.

*--discovery-version*
	Set the version of the discovery method to use for static discovery.
	Set the version of the discovery method to use for static discovery.
	 
				Possible values are:
				- _1_
				- _2_ (default)

# XNAMES

An *xname* is a structured and succinct way to identify a node based on its type
and location. Information goes from general to specific from left to right. Each
of the letters in an xname identify a type while the number to the right of each
character identifies the number of that type. For instance:

```
x1000c1s7b0n0
^    ^ ^ ^ ^
|    | | | `- Node 0
|    | | +--- BMC 0
|    | +----- Compute Module 7
|    +------- Chassis 1
+------------ Cabinet 1000
```

The only important parts as far as the *discover* command is concerned are the
*b* (BMC) and *n* (node) segments. If an xname ends with an *n* segment, it is a
node xname. If an xname ends with a *b* segment, it is a BMC xname.

The concepts of xnames comes from HPE/Cray. See the following for more
information on xnames: https://github.com/Cray-HPE/hms-xname

# AUTHOR

Written by Devon T. Bautista and maintained by the OpenCHAMI developers.

# SEE ALSO

*ochami*(1)

; Vim modeline settings
; vim: set tw=80 noet sts=4 ts=4 sw=4 syntax=scdoc:
