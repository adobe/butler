#Butler Configuration
## Overview
The butler configuration file is a TOML based configuration file. It was felt that TOML is a simpler format to syntactically support, versus YAML or JSON. It is similar to the windows INI format, and you can read more about it at the [TOML](https://github.com/toml-lang/toml) website.

The butler configuaration is broken down into two main sections sections:

1. Globals
1. Managers

For a detailed example of a butler.toml configuration file, please refer to the `butler.toml.sample` file located within this directory.

### Globals
The globals section defines a few global configuration options for handling the managers.

### Managers
The managers section defines the configuration options for how you want to handle downloading and parsing of configuration files from remote locations. The managers are broken into four components

1. Manager Globals
1. Manager Repository Handler
1. Repository Handler Retrieval Options
1. Manager Reloader

Let's dive into each section in more detail.

## Globals

All the globals variables must go under the section which is labeled `[globals]` at the top level of the configuration file.

There are three options that must be configured in the globals section. These options are:

1. config-managers
1. scheduler-interval
1. exit-on-config-failure

### config-manager
The `config-manager` option is an array of managers for butler to handle configuration for. The manager name can be an arbitrary name, but you have to maintain consistency in the name while configuring the manager sub sections. What is more important is how you configure the the Handler and Reloader options of hte manager.

If you do not set the `config-manager` as an array, then it'll take the string value, and turn it into an array, which may have unexpected outcomes.

#### Default Value
None

#### Example
`config-manager = ["prometheus", "alertmanager"]`

### scheduler-interval
The `scheduler-interval` option is how often you want butler to process the configuration files which it has been configured to grab and handle for this manager. The `scheduler-interval` is an integer value in seconds.

#### Default Value
300 (5 minutes)

#### Example
`scheduler-interval = 300`

### exit-on-config-failure
The `exit-on-config-failure` option is a boolean option specifying whether or not you want butler to quit completely, on butler configuration errors.

If this option is not specified, butler will continue to try and retrieve the configuration and try to reload it.

#### Default Value
false

#### Example
`exit-on-config-failure = true`


## Managers
Each manager should go into it's own `[<managers>]` section at the top level of the configuration file. For each manager defined under the `config-manager` global setting, there must be a top level manager configuration of the same name.

That means if the following was set for `config-manager`:
`config-manager = ["a", "b"]`

Then there must be the following managers configured:
```
[a]
... options ...

[b]
... options ...
```
There are seven options that can be configured within the manager configuration section. Not all of them have to have any values associated with them.

1. clean-files
1. mustache-subs
1. enable-cache
1. cache-path
1. dest-path
1. primary-config-name

### clean-files
The `clean-files` configuration option either enables or disables butler from deleting files within the `dest-path` defined directory. From butler's perspective it should be the sole authority of what files it should manage. In the event that certain configuration files were inadvertently placed in the directory, and the tool gets reloaded, which then loads up the configuration file that shouldn't be there, then there could be unanticipated consequences. If you enable this option, butler will remove all files that it does not currently manage.
##### Default Value
false

##### Example
`clean-files = true`
### mustache-subs
### enable-cache
### cache-path
### dest-path
### primary-config-name
### Manager Global
### Manager Repository Handler
### Repository Handler Retrieval Options
### Manager Reloader


