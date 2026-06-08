OCHAMI-BOOT(1) "OpenCHAMI" "Manual Page for ochami-boot"

# NAME

ochami-boot - Communicate with the Boot Service

# SYNOPSIS

*ochami boot* [_global-options_] _command_ [_command-options_] [_arguments_]

*ochami boot* (*bmc* | *config* | *node*) *add* [-f _format_] [-d (_data_ | @_path_ | @-)]++
*ochami boot* (*bmc* | *config* | *node*) *delete* [--no-confirm] _uid_...++
*ochami boot* (*bmc* | *config* | *node*) *get* [-F _format_] _uid_++
*ochami boot* (*bmc* | *config* | *node*) *list* [-F _format_]++
*ochami boot* (*bmc* | *config* | *node*) *patch* [-f _format_] [-p _patch_method_] [-d (_data_ | @_path_ | @-)] _uid_++
*ochami boot* (*bmc* | *config* | *node*) *patch* (--add _key_=_val_ | --remove _key_=_index_ | --set _key_=_val_ | --unset _key_)... _uid_++
*ochami boot* (*bmc* | *config* | *node*) *set* [-f _format_] [-d (_data_ | @_path_ | @-)] _uid_++
*ochami boot service status* [-F _format_]

# DATA STRUCTURE

## BOOT CONFIGURATION

The data structure for sending and receiving boot configuration is detailed in
JSON form below:

```
{
  "hosts":[
    "item1",
    "item2"
  ],
  "macs":[
    "item1",
    "item2"
  ],
  "nids":[
    1,
    2
  ],
  "groups":[
    "item1",
    "item2"
  ],
  "kernel":"http://s3.openchami.cluster/kernels/vmlinuz1",
  "initrd":"http://s3.openchami.cluster/initrds/initramfs1.img",
  "params":"console=tty0,115200n8 console=ttyS0,115200n8",
  "priority": 42
}
```

## BMC SPECIFICATION

The data structure for sending and receiving BMC specifications is detailed in
JSON form below:

```
{
  "xname": "x1000c0s0b0",
  "description": "This node's BMC",
  "interface": {
    "type": "management",
    "mac": "de:ca:fc:0f:fe:e1",
    "ip": "172.16.0.254"
  }
}
```

## NODE CONFIGURATION

The data structure for sending and receiving node specifications is detailed in
JSON form below:

```
{
  "xname": "x1000c0s0b0n0",
  "nid": 42,
  "bootMac": "de:ca:fc:0f:fe:e1",
  "role": "example-role",
  "subRole": "example-subrole",
  "hostname": "ex01.example.org",
  "interfaces": [
    {
      "mac": "de:ca:fc:0f:fe:e1",
      "ip": "172.16.0.1",
      "type": "management"
    }
  ],
  "groups": [
    "group1",
    "group2"
  ]
}
```

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
	_https://foobar.openchami.cluster:8443/boot-service_) or a relative base path
	for the service (e.g. _/boot-service_). If an absolute URI is specified, this
	completely overrides any value set with the *--cluster-uri* flag or
	*cluster.uri* in the config file for the cluster. If using an absolute URI,
	it should contain the desired service's base path. If a relative path is
	specified (with or without the leading forward slash), then this value
	overrides the service's default base path and is appended to the cluster's
	base URI (set with the *--cluster-uri* flag or the *cluster.uri* cluster
	config option), which is required to be set if a relative path is used here.

	The boot service has a base path of */boot-service* by default.

	See *ochami*(1) for *--cluster-uri* and *ochami-config*(5) for details on
	cluster configuration options.

# COMMANDS

The *bmc*, *config*, and *node* commands share a common set of subcommands for
creating, deleting, reading, listing, patching, and replacing boot-service
resources. The *service* command provides operations for boot-service itself.

[[ *Resource*
:< *Subcommands*
:< *Description*
|  *bmc*
:  *add*, *delete*, *get*, *list*, *patch*, *set*
:  Manage BMC specifications
|  *config*
:  *add*, *delete*, *get*, *list*, *patch*, *set*
:  Manage boot configurations
|  *node*
:  *add*, *delete*, *get*, *list*, *patch*, *set*
:  Manage node specifications
|  *service*
:  *status*
:  Check boot-service status

## bmc

Manage BMCs stored in boot-service.

Subcommands for this command are as follows:

*add* [-f _format_] < _file_++
*add* [-f _format_] -d @_file_++
*add* [-f _format_] -d @- < _file_++
*add* [-f _format_] -d _data_
	Add one or more BMC specifications to boot-service.

	In the first and third forms of the command, data is read from standard
	input.

	In the second form of the command, a file containing the payload data is
	passed.

	In the fourth form of the command, the payload is passed raw on the command
	line.

	This command sends a POST request to boot-service's BMC endpoint.

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

*delete* [--no-confirm] _uid_...
	Delete one or more BMCs identified by _uid_. Unless *--no-confirm* is
	passed, the user is asked to confirm deletion.

	This command sends one or more DELETE requests to boot-service's BMC
	endpoint.

	This command accepts the following options:

	*--no-confirm*
		Do not ask the user to confirm deletion. Use with caution.

*get* [-F _format_] _uid_
	Get BMC details for BMC identified by _uid_.

	This command sends a GET to boot-service's BMC endpoint.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

*list* [-F _format_]
	List BMCs stored in boot-service.

	This command sends a GET to boot-service's BMC endpoint.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

*patch* ([--add _key_=_val_]... | [--remove _key_=_index_]... | [--set _key_=_val_]... | [--unset _key_]...) _uid_++
*patch* [-f _format_] [-p _patch_method_] -d _data_ _uid_++
*patch* [-f _format_] [-p _patch_method_] -d @_file_ _uid_++
*patch* [-f _format_] [-p _patch_method_] -d @- _uid_ < _file_++
*patch* [-f _format_] [-p _patch_method_] _uid_ < _file_
	Using various patch methods, patch the specification for an existing BMC
	identified by _uid_.

	*IMPORTANT:* Only the spec portion of the resource can be patched.  Metadata
	(name, labels, annotations) and status are managed by the API.  Attempts to
	patch metadata or status fields will be ignored.

	In the first form of the command, at least one of *--add*, *--remove*,
	*--set*, or *--unset* is passed. Each of these flags can be specified more
	than once, but at least one of them must be passed in this form. This method
	uses add/remove/set/unset flags to perform the patch. For _key_, dot
	notation is used for subkeys (e.g. _key.subkey_).

	In the second through fifth forms of the command, patch data is supplied
	along with an optional *--patch-method* flag to specify the patch method.

	This command sends a PATCH request to boot-service's BMC endpoint.

	This command accepts the following options:

	*--add* _key_[[._subkey_]...]=_val_
		Append value to an existing array field using an RFC 6902 add operation.
		Only can be used with key/value patch flags (automatic if any of
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

	*--remove* _key_[[._subkey_]...]=_index_
		Remove value at _index_ from array field using an RFC 6902 remove
		operation. Only can be used with key/value patch flags (automatic if any of
		*--add*/*--remove*/*--set*/*--unset* are specified).

	*--set* _key_[[._subkey_]...]=_val_
		Set key with its value, overwriting any previous value and creating if the
		key doesn't exist. Only can be used with key/value patch flags (automatic
		if any of *--add*/*--remove*/*--set*/*--unset* are specified).

	*--unset* _key_[[._subkey_]...]
		Unset key (and its value). Only can be used with key/value patch flags
		(automatic if any of
		*--add*/*--remove*/*--set*/*--unset* are specified).

*set* [-f _format_] _uid_ < _file_++
*set* [-f _format_] -d @_file_ _uid_++
*set* [-f _format_] -d @- _uid_ < _file_++
*set* [-f _format_] -d _data_ _uid_
	Set the specification of a BMC identified by _uid_. The entire
	specification for the BMC is replaced with the specification that is passed.

	In the first and third forms of the command, data is read from standard
	input.

	In the second form of the command, a file containing the payload data is
	passed.

	In the fourth form of the command, the payload is passed raw on the command
	line.

	This command sends a PUT request to boot-service's BMC endpoint.

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

## config

Manage boot configurations stored in the boot service.

Subcommands for this command are as follows:

*add* [-f _format_] < _file_++
*add* [-f _format_] -d @_file_++
*add* [-f _format_] -d @- < _file_++
*add* [-f _format_] -d _data_
	Add new boot configuration to be able to be used by nodes. If boot
	configuration already exists for the specified components, this command will
	fail.

	In the first and third forms of the command, data is read from standard
	input.

	In the second form of the command, a file containing the payload data is
	passed.

	In the fourth form of the command, the payload is passed raw on the command
	line.

	This command sends a POST request to boot-service's /bootconfiguration
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

*delete* [--no-confirm] _uid_...
	Delete one or more boot configurations identified by _uid_. Unless
	*--no-confirm* is passed, the user is asked to confirm deletion.

	This command sends one or more DELETE requests to boot-service's
	/bootconfiguration endpoint.

	This command accepts the following options:

	*--no-confirm*
		Do not ask the user to confirm deletion. Use with caution.

*get* [-F _format_] _uid_
	Get boot configuration details for configuration identified by _uid_.

	This command sends a GET to boot-service's /bootconfiguration/_uid_
	endpoints.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

*list* [-F _format_]
	List boot configurations stored in boot-service.

	This command sends a GET to boot-service's /bootconfiguration endpoint.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

*patch* ([--add _key_=_val_]... | [--remove _key_=_index_]... | [--set _key_=_val_]... | [--unset _key_]...) _uid_++
*patch* [-f _format_] [-p _patch_method_] -d _data_ _uid_++
*patch* [-f _format_] [-p _patch_method_] -d @_file_ _uid_++
*patch* [-f _format_] [-p _patch_method_] -d @- _uid_ < _file_++
*patch* [-f _format_] [-p _patch_method_] _uid_ < _file_
	Using various patch methods, patch specification for an existing boot
	configuration identified by _uid_.

	*IMPORTANT:* Only the spec portion of the resource can be patched.  Metadata
	(name, labels, annotations) and status are managed by the API.  Attempts to
	patch metadata or status fields will be ignored.

	In the first form of the command, at least one of *--add*, *--remove*,
	*--set*, or *--unset* is passed. Each of these flags can be specified more
	than once, but at least one of them must be passed in this form. This method
	uses add/remove/set/unset flags to perform the patch. For _key_, dot
	notation is used for subkeys (e.g. _key.subkey_).

	In the second through fifth forms of the command, patch data is supplied
	along with an optional *--patch-method* flag to specify the patch method.

	This command sends a PATCH request to boot-service's
	/bootconfiguration/_uid_ endpoint.

	This command accepts the following options:

	*--add* _key_[[._subkey_]...]=_val_
		Append value to an existing array field using an RFC 6902 add operation.
		Only can be used with key/value patch flags (automatic if any of
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

	*--remove* _key_[[._subkey_]...]=_index_
		Remove value at _index_ from array field using an RFC 6902 remove
		operation. Only can be used with key/value patch flags (automatic if any of
		*--add*/*--remove*/*--set*/*--unset* are specified).

	*--set* _key_[[._subkey_]...]=_val_
		Set key with its value, overwriting any previous value and creating if the
		key doesn't exist. Only can be used with key/value patch flags (automatic
		if any of *--add*/*--remove*/*--set*/*--unset* are specified).

	*--unset* _key_[[._subkey_]...]
		Unset key (and its value). Only can be used with key/value patch flags
		(automatic if any of
		*--add*/*--remove*/*--set*/*--unset* are specified).

*set* [-f _format_] _uid_ < _file_++
*set* [-f _format_] -d @_file_ _uid_++
*set* [-f _format_] -d @- _uid_ < _file_++
*set* [-f _format_] -d _data_ _uid_
	Set the specification of a boot configuration identified by _uid_. The
	entire specification for the boot configuration gets replaced with the
	specification that is passed.

	In the first and third forms of the command, data is read from standard
	input.

	In the second form of the command, a file containing the payload data is
	passed.

	In the fourth form of the command, the payload is passed raw on the command
	line.

	This command sends a PUT request to boot-service's /bootconfiguration/_uid_
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

## node

Manage nodes stored in boot-service.

Subcommands for this command are as follows:

*add* [-f _format_] < _file_++
*add* [-f _format_] -d @_file_++
*add* [-f _format_] -d @- < _file_++
*add* [-f _format_] -d _data_
	Add one or more nodes to boot-service.

	In the first and third forms of the command, data is read from standard
	input.

	In the second form of the command, a file containing the payload data is
	passed.

	In the fourth form of the command, the payload is passed raw on the command
	line.

	This command sends a POST request to boot-service's node endpoint.

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

*delete* [--no-confirm] _uid_...
	Delete one or more nodes identified by _uid_. Unless *--no-confirm* is
	passed, the user is asked to confirm deletion.

	This command sends one or more DELETE requests to boot-service's node
	endpoint.

	This command accepts the following options:

	*--no-confirm*
		Do not ask the user to confirm deletion. Use with caution.

*get* [-F _format_] _uid_
	Get node details for node identified by _uid_.

	This command sends a GET to boot-service's node endpoint.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

*list* [-F _format_]
	List nodes stored in boot-service.

	This command sends a GET to boot-service's node endpoint.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

*patch* ([--add _key_=_val_]... | [--remove _key_=_index_]... | [--set _key_=_val_]... | [--unset _key_]...) _uid_++
*patch* [-f _format_] [-p _patch_method_] -d _data_ _uid_++
*patch* [-f _format_] [-p _patch_method_] -d @_file_ _uid_++
*patch* [-f _format_] [-p _patch_method_] -d @- _uid_ < _file_++
*patch* [-f _format_] [-p _patch_method_] _uid_ < _file_
	Using various patch methods, patch the specification for an existing node
	identified by _uid_.

	*IMPORTANT:* Only the spec portion of the resource can be patched.  Metadata
	(name, labels, annotations) and status are managed by the API.  Attempts to
	patch metadata or status fields will be ignored.

	In the first form of the command, at least one of *--add*, *--remove*,
	*--set*, or *--unset* is passed. Each of these flags can be specified more
	than once, but at least one of them must be passed in this form. This method
	uses add/remove/set/unset flags to perform the patch. For _key_, dot
	notation is used for subkeys (e.g. _key.subkey_).

	In the second through fifth forms of the command, patch data is supplied
	along with an optional *--patch-method* flag to specify the patch method.

	This command sends a PATCH request to boot-service's node endpoint.

	This command accepts the following options:

	*--add* _key_[[._subkey_]...]=_val_
		Append value to an existing array field using an RFC 6902 add operation.
		Only can be used with key/value patch flags (automatic if any of
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

	*--remove* _key_[[._subkey_]...]=_index_
		Remove value at _index_ from array field using an RFC 6902 remove
		operation. Only can be used with key/value patch flags (automatic if any of
		*--add*/*--remove*/*--set*/*--unset* are specified).

	*--set* _key_[[._subkey_]...]=_val_
		Set key with its value, overwriting any previous value and creating if the
		key doesn't exist. Only can be used with key/value patch flags (automatic
		if any of *--add*/*--remove*/*--set*/*--unset* are specified).

	*--unset* _key_[[._subkey_]...]
		Unset key (and its value). Only can be used with key/value patch flags
		(automatic if any of
		*--add*/*--remove*/*--set*/*--unset* are specified).

*set* [-f _format_] _uid_ < _file_++
*set* [-f _format_] -d @_file_ _uid_++
*set* [-f _format_] -d @- _uid_ < _file_++
*set* [-f _format_] -d _data_ _uid_
	Set the specification of a node identified by _uid_. The entire
	specification for the node is replaced with the specification that is
	passed.

	In the first and third forms of the command, data is read from standard
	input.

	In the second form of the command, a file containing the payload data is
	passed.

	In the fourth form of the command, the payload is passed raw on the command
	line.

	This command sends a PUT request to boot-service's node endpoint.

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

## service

Manage and check boot-service itself.

Subcommands for this command are as follows:

*status* [-F _format_]
	Display status of the boot service.

	This command sends a GET to boot-service's health endpoint.

	This command accepts the following options:

	*-F, --format-output* _format_
		Output response data in specified _format_. Supported values are:

		- _json_ (default)
		- _json-pretty_
		- _yaml_

# AUTHOR

Written by Devon T. Bautista and maintained by the OpenCHAMI developers.

# SEE ALSO

*ochami*(1)

; Vim modeline settings
; vim: set tw=80 noet sts=4 ts=4 sw=4 syntax=scdoc:
