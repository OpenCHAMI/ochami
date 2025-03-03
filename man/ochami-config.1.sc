OCHAMI-CONFIG(1) "OpenCHAMI" "Manual Page for ochami-config"

# NAME

ochami-config - Manage configuration for ochami CLI

# SYNOPSIS

ochami config [--user | --system | --config _path_] cluster delete _cluster_name_++
ochami config [--user | --system | --config _path_] cluster set [-d] _cluster_name_ _key_ _value_++
ochami config [--user | --system | --config _path_] cluster show _cluster_name_ [_key_]++
ochami config [--user | --system | --config _path_] cluster unset _cluster_name_ _key_ _value_++
ochami config [--user | --system | --config _path_] set _key_ _value_++
ochami config [--user | --system | --config _path_] show [-f _format_] [_key_]++
ochami config [--user | --system | --config _path_] unset _key_

# COMMANDS

## cluster

Manage cluster configurations.

Subcommands for this command are as follows:

*delete* _cluster_name_
	Delete _cluster_name_ configuration from config file.

*set* [--user | --system | --config _path_] [-u _base_uri_] [-d] _cluster_name_ _key_ _value_
	Add or set configuration for a cluster.

	This command accepts the following options:

	*-u, --base-uri* _base_uri_
		Specify the base URI of OpenCHAMI services for the cluster.

		*ochami* will use this to concatenate endpoint information to when
		communicating with this cluster's OpenCHAMI services.

	*--config* _path_
		Modify the config file at _path_. The *--config* flag is the same one
		that is global to all commands and is not unique to this command.

	*-d, --default*
		Set this cluster as the default cluster. This means that if *--cluster*
		is not specified on the command line, this cluster's configuration is
		used.

	*--system*
		Modify the system config file.

	*--user*
		Modify the user config file (the default).

## set

Set configuration option for ochami CLI.

The format of this command is:

*set* [--user | --system | --config _path_] _key_ _value_

This command sets configuration values for configuration files for the ochami
CLI. It sets the _key_ in the file to _value_. By default, or if *--user* is
specified, the user configuration file is modified. If *--system* is specified,
the system configuration file is modified. Otherwise, if *--config* is specified
(this is the same flag that is available to all commands), _path_ is modified.
See the *FILES* section in *ochami*(1) for details on the user and system files.

This command accepts the following options:

*--config* _path_
	Modify the config file at _path_. The *--config* flag is the same one that
	is global to all commands and is not unique to this command.

*--system*
	Modify the system config file.

*--user*
	Modify the user config file (the default).

## show

Show the current configuration. This command can be used to generate a
configuration file populated with the default values.

This command accepts the following options:

*-f, --format* _format_
	Format of config output.

	Default: *json*
	Supported:
	- _json_
	- _yaml_

## unset

Unset configuration option for ochami CLI.

The format of this command is:

*unset* [--user | --system | --config _path_] _key_

This commands unsets configuration key _key_ for configuration files for the
ochami CLI, in effect deleting it. By default, or if *--user* is specified, the
user configuration file is modified. If *--system* is specified, the system
configuration file is modified. Otherwise, if *--config* is specified (this is
the same flag that is available to all command), _path_ is modified. See the
*FILES* section in *ochami*(1) for details on the user and system files.

*--config* _path_
	Modify the config file at _path_. The *--config* flag is the same one that
	is global to all commands and is not unique to this command.

*--system*
	Modify the system config file.

*--user*
	Modify the user config file (the default).

# AUTHOR

Written by Devon T. Bautista and maintained by the OpenCHAMI developers.

# SEE ALSO

*ochami-config*(5)

; Vim modeline settings
; vim: set tw=80 noet sts=4 ts=4 sw=4 syntax=scdoc:
