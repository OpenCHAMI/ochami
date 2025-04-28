OCHAMI-SMD(1) "OpenCHAMI" "Manual Page for ochami-smd"

# NAME

ochami-smd - Communicate with the State Management Database (SMD)

# SYNOPSIS

ochami smd [OPTIONS] COMMAND

# DATA STRUCTURE

SMD uses several data structures depending on which endpoint is being used.

## ComponentEndpoint

The *ComponentEndpoint* is the sort of "glue" between the *Component* and
*RedfishEndpoint*.

Below is an example of a single *ComponentEndpoint* in JSON form. Note that the
data structure is a single *ComponentEndpoints* object containing an array.

```
{
  "ComponentEndpoints": [
    {
      "ID": "x0c0s0b0n0",
      "Type": "Node",
      "Domain": "mgmt.example.domain.com",
      "FQDN": "x0c0s0b0n0.mgmt.example.domain.com",
      "RedfishType": "ComputerSystem",
      "RedfishSubtype": "Physical",
      "ComponentEndpointType": "ComponentEndpointComputerSystem",
      "MACAddr": "d0:94:66:00:aa:37",
      "UUID": "bf9362ad-b29c-40ed-9881-18a5dba3a26b",
      "OdataID": "/redfish/v1/Systems/System.Embedded.1",
      "RedfishEndpointID": "x0c0s0b0",
      "RedfishEndpointFQDN": "x0c0s0b0.mgmt.example.domain.com",
      "RedfishURL": "x0c0s0b0.mgmt.example.domain.com/redfish/v1/Systems/System.Embedded.1",
      "RedfishSystemInfo": {
        "Name": "System Embedded 1",
        "Actions": {
          "#ComputerSystem.Reset": {
            "AllowableValues": [
              "On",
              "ForceOff"
            ],
            "target": "/redfish/v1/Systems/System.Embedded.1/Actions/ComputerSystem.Reset"
          }
        },
        "EthernetNICInfo": [
          {
            "RedfishId": "1",
            "@odata.id": "/redfish/v1/Systems/System.Embedded.1/EthernetInterfaces/1",
            "Description": "Management Network Interface",
            "InterfaceEnabled": true,
            "MACAddress": "d0:94:66:00:aa:37,",
            "PermanentMACAddress": "d0:94:66:00:aa:37"
          },
          {
            "RedfishId": "2",
            "@odata.id": "/redfish/v1/Systems/System.Embedded.1/EthernetInterfaces/2",
            "Description": "Management Network Interface",
            "InterfaceEnabled": true,
            "MACAddress": "d0:94:66:00:aa:38",
            "PermanentMACAddress": "d0:94:66:00:aa:38"
          }
        ]
      }
    }
  ]
}
```

## Component

The *Component* object contains information for a device. This can be a _Node_,
_NodeBMC_, or other type.

Below is an example of a single *Component* in JSON form. Note that the
structure contains a single *Components* object containing an array.

```
{
  "Components": [
    {
      "ID": "x0c0s0b0n0",
      "Type": "Node",
      "State": "Ready",
      "Flag": "OK",
      "Enabled": true,
      "SoftwareStatus": "string",
      "Role": "Compute",
      "SubRole": "Worker",
      "NID": 1,
      "Subtype": "string",
      "NetType": "Sling",
      "Arch": "X86",
      "Class": "River",
      "ReservationDisabled": false,
      "Locked": false
    }
  ]
}
```

## Group

The *Group* keeps track of *Component* objects organized within groups in SMD.

Below is an example of a single *Group* in JSON form.

```
[
  {
    "label": "blue",
    "description": "This is the blue group",
    "tags": [
      "optional_tag1",
      "optional_tag2"
    ],
    "exclusiveGroup": "optional_excl_group",
    "members": {
      "ids": [
        "x1c0s1b0n0",
        "x1c0s1b0n1",
        "x1c0s2b0n0",
        "x1c0s2b0n1"
      ]
    }
  }
]
```

If performing a PUT on group membership, e.g. with *ochami smd group member
set*, then the form uses _label_ and _ids_ as:

```
{
  "label": "blue",
  "ids": [
    "x1c0s1b0n0",
    "x1c0s1b0n1",
    "x1c0s2b0n0",
    "x1c0s2b0n1"
  ]
}
```

## EthernetInterface

The *EthernetInterface* contains information on a network interface for a
*Component*.

Below is an example of a single *EthernetInterface* in JSON form.

```
[
  {
    "ID": "a4bf012b7310",
    "Description": "string",
    "MACAddress": "string",
    "IPAddresses": [
      {
        "IPAddress": "10.252.0.1",
        "Network": "HMN"
      }
    ],
    "LastUpdate": "2020-05-13T19:18:45.524974Z",
    "ComponentID": "x0c0s1b0n0",
    "Type": "Node"
  }
]
```

## RedfishEndpoint

The *RedfishEndpoint* contains information about a *Component*'s BMC that has
been discovered, e.g. by _magellan_.

Below is an example of a single *RedfishEndpoint* in JSON form. Note that the
structure contains a single *RedfishEndpoints* object containing an array.

```
{
  "RedfishEndpoints": [
    {
      "ID": "x0c0s0b0",
      "Type": "Node",
      "Name": "string",
      "Hostname": "string",
      "Domain": "string",
      "FQDN": "string",
      "Enabled": true,
      "UUID": "bf9362ad-b29c-40ed-9881-18a5dba3a26b",
      "User": "string",
      "Password": "string",
      "UseSSDP": true,
      "MacRequired": true,
      "MACAddr": "ae:12:e2:ff:89:9d",
      "IPAddress": "10.254.2.10",
      "RediscoverOnUpdate": true,
      "TemplateID": "string",
      "DiscoveryInfo": {
        "LastAttempt": "2024-11-20T19:05:44.253Z",
        "LastStatus": "EndpointInvalid",
        "RedfishVersion": "string"
      }
    }
  ]
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

## compep

Manage component endpoints.

Subcommands for this command are as follows:

*delete* [--no-confirm] --all++
*delete* [--no-confirm] _xname_...++
*delete* [--no-confirm] -d _data_ [-f _format_]++
*delete* [--no-confirm] -d @_file_ [-f _format_]++
*delete* [--no-confirm] -d @- [-f _format_]
	Delete one or more component endpoints. Unless *--no-confirm* is passed, the
	user is asked to confirm deletion.

	In the first form of the command, all component endpoints are deleted. *BE
	CAREFUL!*

	In the second form of the command, one or more xnames identifying the
	component(s) whose component endpoint(s) to delete is/are specified.

	In the third form of the command, raw data is passed as an argument to be
	the payload.

	In the fourth form of the command, a file containing the payload data (see
	the *ComponentEndpoint* data structure above) is passed. This is convenient
	in cases of dealing with many component endpoints at once.

	In the fifth form of the command, the payload data is read from standard
	input.

	This command sends one or more DELETE requests to SMD's /ComponentEndpoints
	endpoint.

	This command accepts the following options:

	*-a, --all*
		Delete *all* component endpoints in SMD. *BE CAREFUL!*

	*-d, --data* (_data_ | @_path_ | @-)
		Specify raw _data_ to send, the _path_ to a file to read payload data
		from, or to read the data from standard input (@-). The format of data
		read in any of these forms is JSON by default unless *-f* is specified
		to change it.

	*--no-confirm*
		Do not ask the user to confirm deletion. Use with caution.

	*-f, --format-input* _format_
		Format of raw data being used by *-d* as the payload. Supported formats
		are:

		- _json_ (default)
		- _yaml_

*get* [-F _format_] [_xname_]...
	Get all or a subset of component endpoints.

	If no arguments are passed, all component endpoints are returned. Otherwise,
	the results are filtered by one or more passed xnames.

	This command sends a GET request to SMD's /ComponentEndpoints endpoint.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _yaml_

## component

Manage components.

Subcommands for this command are as follows:

*add* [--arch _arch_] [--enabled] [--role _role_] [--state _state_] _xname_ _node_id_++
*add* -d _data_ [-f _format_]++
*add* -d @_file_ [-f _format_]++
*add* -d @- [-f _format_]
	Add one or more new components to SMD. If a component already exists with
	the same xname, this command will fail.

	In the first form of the command, an _xname_ and _node_id_ is required to
	identify the component to add. One or more of *--arch*, *--enabled*,
	*--role*, or *--state* can optionally be specified to specify details of the
	component.

	In the second form of the command, raw data is passed as an argument to be
	the payload.

	In the third form of the command, a file containing the payload data is
	passed. This is convenient in cases of dealing with many components at once.

	In the fourth form of the command, the payload data is read from standard
	input.

	This command sends a POST request to SMD's /Components endpoint.

	This command accepts the following options:

	*--arch* _arch_
		Specify CPU architecture of component.

		Default: *X86*

	*-d, --data* (_data_ | @_path_ | @-)
		Specify raw _data_ to send, the _path_ to a file to read payload data
		from, or to read the data from standard input (@-). The format of data
		read in any of these forms is JSON by default unless *-f* is specified
		to change it.

	*--enabled*
		Specify if component is shows up as enabled in SMD.

		Default: *true*

	*-f, --format-input* _format_
		Format of raw data being used by *-d* as the payload. Supported formats
		are:

		- _json_ (default)
		- _yaml_

	*--role* _role_
		Specify the SMD role for the new component.

		Default: *Compute*

	*--state* _state_
		Specify the initial state of the new component.

		Default: *Ready*

*delete* --all++
*delete* _xname_...++
*delete* -d _data_ [-f _format_]++
*delete* -d @_file_ [-f _format_]++
*delete* -d @- [-f _format_]
	Delete one or more components in SMD. Unless *--no-confirm* is passed, the
	user is asked to confirm deletion.

	In the first form of the command, all components are deleted. *BE CAREFUL!*

	In the second form of the command, one or more xnames identifying the
	component(s) to delete is/are specified.

	In the third form of the command, raw data is passed as an argument to be
	the payload.

	In the fourth form of the command, a file containing the payload data (see
	the *Component* data structure above) is passed. This is convenient in cases
	of dealing with many components at once.

	In the fifth form of the command, the payload is read from standard input.

	This command sends one or more DELETE requests to SMD's /Components
	endpoint.

	This command accepts the following options:

	*-a, --all*
		Delete *all* components in SMD. *BE CAREFUL!*

	*-d, --data* (_data_ | @_path_ | @-)
		Specify raw _data_ to send, the _path_ to a file to read payload data
		from, or to read the data from standard input (@-). The format of data
		read in any of these forms is JSON by default unless *-f* is specified
		to change it.

	*--no-confirm*
		Do not ask the user to confirm deletion. Use with caution.

	*-f, --format-input* _format_
		Format of raw data being used by *-d* as the payload. Supported formats
		are:

		- _json_ (default)
		- _yaml_

*get* [-F _format_] [--nid _nid_] [--xname _xname_]
	Get all components or one identified by xname or node ID.

	If no filter flags are passed, all components are returned. Otherwise, the
	component specified by the passed filter flag(s) is returned.

	This command sends a GET request to SMD's /Components endpoint.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _yaml_

	*-n, --nid* _nid_,...
		One or more node IDs to filter results by. For multiple NIDs, either
		this flag can be specified multiple times or this flag can be specified
		once and multiple NIDs can be specified, separated by commas.

	*-x, --xname* _xname_,...
		One or more xnames to filter results by. For multiple xnames, either
		this flag can be specified multiple times or this flag can be specified
		once and multiple xnames, separated by commas.

## rfe

Manage Redfish endpoints. 

Subcommands for this command are as follows:

*add* [--domain _domain_] [--hostname _hostname_] [--username _user_] [--password _pass_] _xname_ _name_ _ip_addr_ _mac_addr_++
*add* [-f _format_] -d _data_++
*add* [-f _format_] -d @_path_++
*add* [-f _format_] -d @-++
	Add one or more new Redfish endpoints to SMD.

	In the first form of the command, an _xname_ (unique identifier), _name_
	(human-readable name), _ip_addr_ (IP address), and _mac_addr_ (MAC address)
	are required to identify and define the endpoint to add. Optional flags like
	*--domain*, *--hostname*, *--username*, and *--password* can provide
	additional details.

	In the second form of the command, raw data is passed as an argument to be 
	the payload.
	
	In the third form of the command, a file containing the payload data is
	passed. This is convenient in cases of dealing with many components at once.

	In the fourth form of the command, the payload data is read from standard
	input.

	This command sends a POST request to SMD's /RedfishEndpoints endpoint. An
	access token is required.

	This command accepts the following options:

	*-d, --data* (_data_ | @_path_ | @-)
		Specify raw _data_ to send, the _path_ to a file to read payload data
		from, or to read the data from standard input (@-). The format of data
		read in any of these forms is JSON by default unless *-f* is specified
		to change it.

	*--domain* _domain_
		Specify the domain part of the Redfish endpoint's FQDN.

	*-f, --format-input* _format_
		Format of input payload data used by *-d*. Supported formats are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

	*--hostname* _hostname_
		Specify the hostname part of the Redfish endpoint's FQDN.

	*--password* _password_
		Specify the password to use when interrogating the endpoint (stored in SMD).

	*--username* _username_
		Specify the username to use when interrogating the endpoint (stored in SMD).

*delete* [--no-confirm] --all++
*delete* [--no-confirm] _xname_...++
*delete* [--no-confirm] [-f _format_] -d _data_++
*delete* [--no-confirm] [-f _format_] -d @_path_++
*delete* [--no-confirm] [-f _format_] -d @-++
	Delete one or more Redfish endpoints in SMD. Unless *--no-confirm* is passed, the
	user may be asked to confirm deletion.

	In the first form of the command, all Redfish endpoints are deleted.

	In the second form of the command, one or more _xname_ arguments identifying the
	endpoint(s) to delete are specified.

	In the third form of the command, a file containing the payload data is
	passed. This is convenient in cases of dealing with many components at once.

	This command sends one or more DELETE requests to SMD's /RedfishEndpoints
	endpoint. An access token is required.

	This command accepts the following options:

	*-a, --all*
		Delete *all* Redfish endpoints in SMD.

	*-d, --data* (_data_ | @_path_ | @-)
		Specify raw _data_ to send, the _path_ to a file to read payload data
		from, or to read the data from standard input (@-). The format of data
		read in any of these forms is JSON by default unless *-f* is specified
		to change it.

	*-f, --format-input* _format_
		Format of input payload data used by *-d*. Supported formats are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

	*--no-confirm*
		Do not ask the user to confirm deletion.

*get* [-F _format_] [--fqdn _fqdn_,...] [-i _ip_,...] [-m _mac_,...] [--type _type_,...] [--uuid _uuid_,...] [-x _xname_,...]
	Get all Redfish endpoints or filter by various attributes.

	If no filter flags are passed, all Redfish endpoints are returned.
	Otherwise, only the endpoint(s) matching the specified filter criteria are
	returned. Multiple filters can be combined. For flags accepting multiple
	values (like *--xname*), values can be comma-separated or the flag can be
	repeated.

	This command sends a GET request to SMD's /RedfishEndpoints endpoint.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

	*--fqdn* _fqdn_,...
		Filter Redfish endpoints by one or more Fully Qualified Domain Names (FQDNs).

	*-i, --ip* _ip_,...
		Filter Redfish endpoints by one or more IP addresses.

	*-m, --mac* _mac_,...
		Filter Redfish endpoints by one or more MAC addresses.

	*--type* _type_,...
		Filter Redfish endpoints by one or more types (e.g., *NodeBMC*, *RouterBMC*).

	*--uuid* _uuid_,...
		Filter Redfish endpoints by one or more UUIDs.

	*-x, --xname* _xname_,...
		Filter Redfish endpoints by one or more xnames.

## group

Manage SMD groups. For managing group membership, see *group member* below.

Subcommands for this command are as follows:

*add* [--description _desc_] [--tag _tag_,...] [--member _xname_,...] [--exclusive-group _group_] _group_name_++
*add* -d _data_ [-f _format_]++
*add* -d @_file_ [-f _format_]++
*add* -d @- [-f _format_]
	Add a new group to SMD, optionally specifying members to add to the group.

	In the first form of the command, a _group_name_ is required to create the
	new group. An optional group description can be specified with
	*--description*. One or more components can be added to the new group by
	passing *--member* and one or more tags can be assigned to the group by
	passing *--tag*. Finally, the group can be set to be mutually exclusive with
	another group by passing *--exclusive-group*.

	In the second form of the command, raw data is passed as an argument to be
	the payload.

	In the third form of the command, a file containing the payload data is
	passed. This is convenient in cases of dealing with many groups at once.

	In the fourth form of the command, the payload data is read from standard
	input.

	This command sends one or more POST requests to SMD's /groups endpoint.

	This command accepts the following options:

	*-d, --data* (_data_ | @_path_ | @-)
		Specify raw _data_ to send, the _path_ to a file to read payload data
		from, or to read the data from standard input (@-). The format of data
		read in any of these forms is JSON by default unless *-f* is specified
		to change it.

	*-D, --description* _description_
		Specify a brief description of the group.

		Default: *The <group_name> group*

	*-e, --exclusive-group* _group_name_
		Specify a single group that the specified group will be mutually
		exclusive with. In other words, components in this group cannot also be
		a member of the specified exclusive group.

	*-f, --format-input* _format_
		Format of raw data being used by *-d* as the payload. Supported formats
		are:

		- _json_ (default)
		- _yaml_

	*-m, --member* _xname_,...
		One or more component IDs (xnames) to add to the group. For multiple
		components, either this flag can be specified multiple times or this
		flag can be specified once and multiple component IDs can be specified,
		separated by commas.

	*--tag* _tag_,...
		One or more tags to assign to the group. For multiple tags, either this
		flag can be specified multiple times or this flag can be specified once
		and multiple tags can be specified, separated by commas.

*delete* [--no-confirm] _group_name_...++
*delete* [--no-confirm] -d _data_ [-f _format_]++
*delete* [--no-confirm] -d @_file_ [-f _format_]++
*delete* [--no-confirm] -d @- [-f _format_]
	Delete one or more groups in SMD. Unless *--no-confirm* is passed, the user
	is asked to confirm deletion.

	In the first form of the command, one or more group labels can be specified
	to delete one or more groups.

	In the second form of the command, raw data is passed as an argument to be
	the payload.

	In the third form of the command, a file containing the payload data is
	passed. This is convenient in cases of dealing with many groups at once.

	In the fourth form of the command, the payload data is read from standard
	input.

	This command sends one or more DELETE requests to SMD's /groups endpoint.

	This command accepts the following options:

	*-d, --data* (_data_ | @_path_ | @-)
		Specify raw _data_ to send, the _path_ to a file to read payload data
		from, or to read the data from standard input (@-). The format of data
		read in any of these forms is JSON by default unless *-f* is specified
		to change it.

	*--no-confirm*
		Do not ask the user to confirm deletion. Use with caution.

	*-f, --format-input* _format_
		Format of raw data being used by *-d* as the payload. Supported formats
		are:

		- _json_ (default)
		- _yaml_

*get* [-F _format_] [--name _name_,...] [--tag _tag_,...]
	Get group information for all groups in SMD or for a subset, specified by
	filters.

	This command sends a GET to SMD's /groups endpoint.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _yaml_

	*--name* _group_name_,...
		One or more group names to filter groups by. For multiple groups names,
		either this flag can be specified multiple times or this flag can be
		specified once and multiple group names can be specified, separated by
		commas.

	*--tag* _tag_,...
		One or more tags to filter groups by. For multiple tags, either this
		flag can be specified multiple times or this flag can be specified once
		and multiple tags can be specified, separated by commas.

*update* [--description _description_] [--tag _tag_,...] _group_name_++
*update* -d _data_ [-f _format_]++
*update* -d @_file_ [-f _format_]++
*update* -d @- [-f _format_] < _file_
	Update one or more existing groups in SMD. If the group does not already
	exist, this command will fail.

	In the first form of the command, a _group_name_ is required as well as at
	least one of *--description* or *--tag*.

	In the second form of the command, raw data is passed as an argument to be
	the payload.

	In the third form of the command, a file containing the payload data is
	passed. This is convenient in cases of dealing with many groups at once.

	In the fourth form of the command, the payload data is read from standard
	input.

	This command sends a PATCH  request to SMD's /groups endpoint.

	This command accepts the following options:

	*-d, --data* (_data_ | @_path_ | @-)
		Specify raw _data_ to send, the _path_ to a file to read payload data
		from, or to read the data from standard input (@-). The format of data
		read in any of these forms is JSON by default unless *-f* is specified
		to change it.

	*-D, --description* _description_
		Specify a brief description of the group.

	*-f, --format-input* _format_
		Format of raw data being used by *-d* as the payload. Supported formats
		are:

		- _json_ (default)
		- _yaml_

	*--tag* _tag_,...
		One or more tags to assign to the group. For multiple tags, either this
		flag can be specified multiple times or this flag can be specified once
		and multiple tags can be specified, separated by commas. Passing this
		flag will *replace* any existing tags, so be sure any existing tags that
		need to be kept are passed to this flag.

## group member

Manage SMD group membership. For general group management, see *group*.

Subcommands for this command are as follows:

*add* _group_name_ _xname_...
	Add one or more components to an existing SMD group.

	This command sends one or more POST requests to the members subendpoint
	under SMD's /groups endpoint.

*delete* _group_name_ _xname_...
	Delete one or more components from an existing SMD group.

	This command sends one or more DELETE requests to the members subendpoint
	under SMD's /groups endpoint.

*get* [-F _format_] _group_name_
	Get members of an SMD group.

	This command sends a GET request to the members subendpoint under SMD's
	/groups endpoint.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _yaml_

*set* _group_name_ _xname_...
	Set the membership list of _group_name_ to _xname_.... Xnames specified that
	are not already in the group are added to it, xnames specified that are
	already in the group remain in the group, and xnames not specified that are
	already in the group are removed from the group.

# AUTHOR

Written by Devon T. Bautista and maintained by the OpenCHAMI developers.

# SEE ALSO

*ochami*(1)

; Vim modeline settings
; vim: set tw=80 noet sts=4 ts=4 sw=4 syntax=scdoc:
