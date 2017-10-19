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
1. Manager Reloader Options

Let's dive into each section in more detail.

## Globals

All the globals variables must go under the section which is labeled `[globals]` at the top level of the configuration file.

There are four options that must be configured in the globals section. These options are:

1. config-managers
1. scheduler-interval
1. exit-on-config-failure
1. status-file

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

### status-file
The `status-file` option is a string path to the location where butler should store some internal status information to.
It should be readable and writable by the user that butler runs as.

#### Default Value
/var/tmp/butler.status

#### Example
`status-file = "/var/tmp/butler.status`

## Managers / Manager Globals
Each manager should go into it's own `[<managers>]` section at the top level of the configuration file. For each manager defined under the `config-manager` global setting, there must be a top level manager configuration of the same name. The goal of the manager is to be what butler uses to manage a specific set of configuration files for a configured tool.

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

1. repos
1. clean-files
1. mustache-subs
1. enable-cache
1. cache-path
1. dest-path
1. primary-config-name

### repos
The `repos` configuration option defines an array of repositories where butler is going to attempt to gather configuration files from. This must be defined, and if it is not, butler will not continue, since it has nothing to work with.

#### Default Value
Empty Array

#### Example
`repos = ["repo1.domain.com", "repo2.domain.com"]`

### clean-files
The `clean-files` configuration option either enables or disables butler from deleting files within the `dest-path` defined directory. From butler's perspective it should be the sole authority of what files it should manage. In the event that certain configuration files were inadvertently placed in the directory, and the tool gets reloaded, which then loads up the configuration file that shouldn't be there, then there could be unanticipated consequences. If you enable this option, butler will remove all files that it does not currently manage.

##### Default Value
false

##### Example
`clean-files = true`

### mustache-subs
The `mustache-subs` configuration option defines an array of mustache substitutions that should be attempted on EVERY configuration file that butler managages. The mustache substitutions should be in the form of mustache=substitution format.

#### Default Value
An empty array

#### Example
`mustache-subs = ["mustache-sub-entry=Some text to replace mustache-sub-entry with"]`

### enable-cache
The `enable-cache` configuration option either enables or disables butler from caching the currently known good configuration files. This is helpful in the event that a bad configuration file gets downloaded, then butler can put the last known good configuration back into place.

#### Default Value
false

#### Example
`enable-cache = true`

### cache-path
The `cache-path` configuration option tells butler where to store the cached configuration files to.

##### Default Value
Empty String

#### Example
`cache-path = "/opt/butler/cache"`

### dest-path
The `dest-path` configuration option tells butler where it should put all of the configuration files that are managed by butler.

#### Default Value
Empty String

#### Example
`dest-path = "/opt/prometheus/etc"`

### primary-config-name
The `primary-config-name` configuration option tells butler where all the files defined under a manager configuration's `primary-config` configuration option should be stored. One of the initial goals of butler was to take a bunch of files from one a repo, and merge them into one primary configuration file. This option tells butler what that configuration file should be.

## Repository Handler
Each Repository Handler configuration must be under the config Manager section, and must be one of the options which are defined under the `repos` option within the Manager defintion.

For example, look at the following (incomplete) definition:
```
[globals]
  config-managers = ["a", "b"]
...
[a]
^^^ Manager
  repos = ["repo1.domain.com", "repo2.domain.com"]
  ...
  [a.repo1.domain.com]
  ^^^^^^^^^^^^^^^^^^^^ This is where the Repository Handler configurationn option should reside.
```

There are 4 options that can be configured under the Repository Handler configuration section.
1. method
1. repo-path
1. primary-config
1. additional-config

### method
### repo-path
### primary-config
### additional-config


## Repository Handler Retrieval Options
The Repository Handler Retrieval Options must be defined under the Repository Handler using the name of the defined method.

For example, look at the following (incomplete) definition:
```
[globals]
  config-managers = ["a", "b"]
...
[a]
  repos = "repo1.domain.com", "repo2.domain.com"]
  ...
  [a.repo1.domain.com]
    method = "http"
    ...
    [a.repo1.domain.com.http]
    ^^^^^^^^^^^^^^^^^^^^^^^^^ This is wehre the Repository Handler Retrieval Options should reside.
```
## Manager Reloader
The Manager Reloader Option defines how the manager is to be reloaded. Currently there is one methods of reloading a manager. That is either over http or https connections.

The Manager Reloader Option must be defined under the config Manager section. Let's look at the following (incomplete) configuration snippet:
```
[globals]
  config-managers = ["a", "b"]
  ...
[a]
  ...
  [a.reloader]
  ^^^^^^^^^^^^ This is where the Manager Reloader options should reside
```

There is only one option that can be configured under the Manager Reloader Options.
1. method

### method
The `method` option defines what method to use to handle the reloading of the manager which butler is managing configuration files for. Currently this option is only http or https. This means that the application which butler is managing configurations for must have the ability to be reloaded by HTTP. In the future, we'll be adding the mechanism to reload via a command line method.

## Manager Reloader Options
The Manager Reloader Options option defines which options need to be used in order to reload the manager successfully.

The Manager Reloader Options option must be defined under the Manager Reloader section. Let's look at the following (incomplete) configuration snippet:
```
[globals]
  config-managers = ["a", "b"]
  ...
[a]
  ...
  [a.reloader]
    method = "http"

    [a.reloader.http]
    ^^^^^^^^^^^^^^^^^ This is where the Manager Reloader Options options should reside.
```
### HTTP(S) Reloader Options
Currently http/https are the only reloader options that are supported. The options which must be configured for the http/https reloader are.

1. host
1. port
1. uri
1. method
1. payload
1. content-type
1. retries
1. retry-wait-min
1. retry-wait-max
1. timeout


