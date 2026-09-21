OCHAMI-CONFIG(1) "OpenCHAMI" "Manual Page for ochami-config"

# NAME

ochami-config - Manage configuration for ochami CLI

# SYNOPSIS

ochami config [GLOBALOPTS] cluster delete _cluster_name_++
ochami config [GLOBALOPTS] cluster set [-d] _cluster_name_ _key_ _value_++
ochami config [GLOBALOPTS] cluster show [_cluster_name_] [_key_]++
ochami config [GLOBALOPTS] cluster unset _cluster_name_ _key_++
ochami config [GLOBALOPTS] set _key_ _value_++
ochami config [GLOBALOPTS] show [_key_]++
ochami config [GLOBALOPTS] unset _key_

# GLOBAL OPTIONS

*--config* _path_
	Use the configuration file at _path_.

*--system*
	Use the system-wide configuration file. (See *FILES* below.)

*--user*
	Use the user-level configuration file. (See *FILES* below.)

# DESCRIPTION

The *config* metacommand is used for printing and modifying configuration
options for *ochami* within its configuration files (see *FILES* below). If
neither *--config*, *--system*, nor *--user* (mutually exclusive) are specified,
*ochami* uses the user-level configuration file for modification commands and
uses the resulting config of merging the user-level config with the system-wide
config (the former preceding the latter) for printing commands.

Missing system-wide and user-level configuration files are skipped. If a file
exists but cannot be read, parsed, validated, or merged, *ochami* reports the
error and exits instead of silently ignoring the invalid configuration.

Modification commands replace the configuration file atomically: they write
the new contents to a temporary file in the same directory and rename it over
the original, so that directory must be writable. If the configuration file
is a symbolic link, the file it points to is replaced and the link is kept.

The format of _key_ uses a period (*.*) to delimit subkeys, following a
*<superkey>.<subkey>* syntax. For example, in order to reference the *format*
key under the *log* key, the key reference path would be *log.format*.

For global commands (e.g. any command not under *cluster*), only global keys are
allowed to be referenced. In other words, the key may not begin with *clusters*.
For example, *log.level* is allowed but *clusters[0].name* or
*clusters[0].cluster.uri* are not allowed. See *ochami-config*(5) for details on
available keys.

For *cluster* commands, only cluster-specific keys are allowed to be referenced.
For example, *cluster.uri* or *name* are allowed, but *log.format* is not. See
*ochami-config*(5) for details on available keys.

# COMMANDS

## cluster

Manage cluster configurations.

Subcommands for this command are as follows:

*delete* _cluster_name_
	Delete _cluster_name_ configuration from config file.

*set* [-d] _cluster_name_ _key_ _value_
	Add or set configuration for a cluster.

	If _cluster_name_ does not exist in the configuration file, it is created,
	unless _key_ is *name*: renaming a cluster requires the cluster to exist.
	_key_ can be a top-level cluster key (e.g. *name*) or a cluster config
	option (e.g. *cluster.uri*). When changing a cluster's name, if that cluster
	is the default cluster, then *default-cluster* will be changed to the
	cluster's new name. Changing a cluster's name to an existing cluster name is
	not allowed. A cluster name must be non-empty and must not contain a period
	(*.*).

	This command accepts the following options:

	*-d, --default*
		Set this cluster as the default cluster. This means that if *--cluster*
		is not specified on the command line, this cluster's configuration is
		used.

*show* [_cluster_name_] [_key_]
	Show the configuration for _cluster_name_ or all clusters if _cluster_name_
	is unspecified. If _key_ is not specified, show the whole configuration for
	_cluster_name_, otherwise show the value for _key_. Configuration is printed
	as YAML.

*unset* _cluster_name_ _key_
	Unset the _key_ configuration option from _cluster_name_

## set

Set configuration option for ochami CLI.

The format of this command is:

*set* _key_ _value_

This command sets global configuration values for *ochami*. It sets the _key_ in
the file to _value_.

## show

Show the *ochami* configuration.

The format of this command is:

*show* [_key_]

Print the known *ochami* configuration. An optional _key_ can be passed to print
a specific global config option, otherwise the whole configuration is printed.
Configuration is printed as YAML.
By default, the config that is used is that merged from the user-level config
file and the system-wide config file, with the former preceding the latter. This
is unless any of the config file options are passed. In that case, only the
config from the relevant file is read.

This command only deals with global configuration options, and not with
individual cluster configurations, though the cluster list can be shown. Use
*ochami config cluster show* to view individual cluster configuration.

## unset

Unset global configuration option.

The format of this command is:

*unset* _key_

# FILES

_/etc/ochami/config.yaml_
	The system-wide configuration file for *ochami*.

_$XDG_CONFIG_HOME/ochami/config.yaml_
	The user-level configuration file for *ochami* if *XDG_CONFIG_HOME* is set
	to an absolute path.

_~/.config/ochami/config.yaml_
	The user-level configuration file for *ochami* if *XDG_CONFIG_HOME* is unset
	or not an absolute path.

# AUTHOR

Written by Devon T. Bautista and maintained by the OpenCHAMI developers.

# SEE ALSO

*ochami*(1), *ochami-config*(5)

; Vim modeline settings
; vim: set tw=80 noet sts=4 ts=4 sw=4 syntax=scdoc:
