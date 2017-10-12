#Butler Configuration
## Overview
The butler configuration file is a TOML based configuration file. It was felt that TOML is a simpler format to syntactically support, versus YAML or JSON. It is similar to the windows INI format, and you can read more about it at the [TOML](https://github.com/toml-lang/toml) website.

The butler configuaration is broken down into two main sections sections:

1. Globals
1. Managers

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
### Manager Global
### Manager Repository Handler
### Repository Handler Retrieval Options
### Manager Reloader


