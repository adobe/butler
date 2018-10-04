# Butler Configuration
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
1. enable-http-log

### config-manager
The `config-manager` option is an array of managers for butler to handle configuration for. The manager name can be an arbitrary name, but you have to maintain consistency in the name while configuring the manager sub sections. What is more important is how you configure the the Handler and Reloader options of hte manager.

If you do not set the `config-manager` as an array, then it'll take the string value, and turn it into an array, which may have unexpected outcomes.

#### Default Value
None

#### Example
`config-manager = ["prometheus", "alertmanager"]`

### scheduler-interval
The `scheduler-interval` option is how often you want butler to process the configuration files which it has been configured to grab and handle for this manager. The `scheduler-interval` is an string based integer value in seconds.

#### Default Value
"300" (5 minutes)

#### Example
`scheduler-interval = "300"`

### exit-on-config-failure
The `exit-on-config-failure` option is a stringed boolean option (eg: "true" or "false")specifying whether or not you want butler to quit completely, on butler configuration errors.

If this option is not specified, butler will continue to try and retrieve the configuration and try to reload it.

#### Default Value
"false"

#### Example
`exit-on-config-failure = "true"`

### status-file
The `status-file` option is a string path to the location where butler should store some internal status information to.
It should be readable and writable by the user that butler runs as.

#### Default Value
/var/tmp/butler.status

#### Example
`status-file = "/var/tmp/butler.status"`

### enable-http-log
The `enable-http-log` option is a string boolean value which configures whether or not butler will log http requests to its stderr output, on top of all the other logs that
it prints. It logs in the standard Apache log format.

#### Default Value
"true"

#### Example
`status-file = "/var/tmp/butler.status"`

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
'false"

##### Example
`clean-files = "true"`

### mustache-subs
The `mustache-subs` configuration option defines an array of mustache substitutions that should be attempted on EVERY configuration file that butler managages. The mustache substitutions should be in the form of mustache=substitution format.

#### Default Value
An empty array

#### Example
`mustache-subs = ["mustache-sub-entry=Some text to replace mustache-sub-entry with"]`

### enable-cache
The `enable-cache` configuration option either enables or disables butler from caching the currently known good configuration files. This is helpful in the event that a bad configuration file gets downloaded, then butler can put the last known good configuration back into place.

#### Default Value
"false"

#### Example
`enable-cache = "true"`

### cache-path
The `cache-path` configuration option tells butler where to store the cached configuration files to. If enable-cache is set to "true", then cache-path must exist.

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
Each Repository Handler configuration must be under the config Manager section, and must be one of the options which are defined under the `repos` option within the Manager definition.

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
The `method` option defines what method to use for the retrieval of configuration files. Currently this option is only blob, file, http/https, and S3.

#### Default Value
None

#### Example
1. `method = "blob"`
1. `method = "file"`
1. `method = "http"`
1. `method = "https"`
1. `method = "S3"`

### repo-path
The `repo-path` option is the URI path to the configuration file on the local or remote filesystem. It should not be a relative path, and should not include any host information. In case of S3 this will be relative the folder names defined under `repos` and can be left blank. In the case of blob, the repo-path can be set to the storage account name.

#### Default Value
None

#### Example
`repo-path = "/butler/configs/prometheus"`

### primary-config
The `primary-config` option is an array of strings, that are configuration files which will get merged into the single configuration file referenced by `primary-config-name` under the Manager Globals section. You can include paths in the configuration file name, and the paths will be retrieved relative to the `repo-path` that was defined previously. If the file is `additional/config2.yml`, then it will be retrieved from `<repo url>/butler/configs/prometheus/additional/config2.yml`

#### Default Value
[]

#### Example
`primary-config = ["config1.yml", "additional/config2.yml"]`

### additional-config
The `additional-config` option is an array strings, which are additional configuration files which will be put on the filesystem under `dest-path` as they are defined within the option. They will be retrieved relative to the `repo-path`. If the file is called `additional/config2.yml`, then it will be retrieverd from `<repo url>/butler/configs/prometheus/additional/config2.yml` and placed on the filesystem as `<dest-path>/additional/config2.yml`

#### Default Value
[]

#### Example
`additional-config = ["alerts/alerts1.yml", "extras/alertmanager.yml"]`

## Repository Handler Retrieval Options (HTTP)
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
    ^^^^^^^^^^^^^^^^^^^^^^^^^ This is where the Repository Handler Retrieval Options should reside.
```
## Repository Handler Retrieval Options (FILE)
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
    method = "file"
    ...
    [a.repo1.domain.com.file]
    ^^^^^^^^^^^^^^^^^^^^^^^^^ This is where the Repository Handler Retrieval Options should reside.
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
The `method` option defines what method to use to handle the reloading of the manager which butler is managing configuration files for. Currently this option is only http or https. This means that the application which butler is managing configurations for must have the ability to be reloaded by HTTP. In the future, there will be added the mechanism to reload via a command line method.

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
1. auth-type
1. auth-user
1. auth-token

#### host
The `host` option is the host that the http connection will utilise.

#### port
The `port` option is what port you want the http connection to use. This is a required option.

#### uri
The `uri`
#### method
The `method` option is the HTTP method to use when handling the reload operation.

#### payload
The `payload` option is what you will be PUT/POSTed to the server in the reload operation.

#### content-type
The `content-type` option is the http content type header to use if your method is PUT/POST.

#### retries
The `retries` option defines how many retries the reloader should take when attempting to reload the service. There is no default value, and this must be set.

#### retry-wait-min
The `retry-wait-min` option is the minimum amount of time, in seconds, to HOLD OFF in seconds before attempting the retry.

#### retry-wait-max
The `retry-wait-max` option is the maximum amount of time, in seconds, to HOLD OFF in seconds before attempting the retry.

#### timeout
The `timeout` option is the amount of time, in seconds, until the http connection times out.

#### auth-type
The `auth-type` option is where you can define the authentication type to attempt when butler tries to retrieve configs from a repo. The valid auth-type options are `basic` and `digest` `token-key`.
Refer to the main Butler CMS [README](README.md) for details on the differences and usage of the fields.

#### auth-user
The `auth-user` option defines what the user is that should be used when trying to authenticate to the repository.
For `token-key` authentication, use this field for the token section.

#### auth-token
The `auth-token` option defines what password/token should be used when trying to authenticate to the repository.
For `token-key` authentication, use this field for the key section.


### FILE Retrieval Options
The file retrieval option only has one option that can be used. If you use this option, then you are not going to use the `repo-path` option under the Repository Handler configuration section. Just set `repo-path=""`. Alternatively, you do not have to set this option, and use `repo-path` instead.

1. path

#### path
The `path` option is the path on the filesystem where butler should be looking for files.

Here is an example:

```
[globals]
  config-managers = ["a", "b"]
...
[a]
  repos = "repo1.domain.com", "repo2.domain.com"]
  ...
  [a.repo1.domain.com]
    method = "file"
    repo-path = ""
    ...
    [a.repo1.domain.com.file]
      path = "/our/path/to/configs"
```

### Blob Retrieval Options
The blob retrieval option only has one option that can be used. If you use this option, then you are do not have to set the BUTLER_STORAGE_ACCOUNT environment variable.

1. storage-account-name

#### storage-account-name
The `storage-account-name` option is name of the Azure Storage Account

Here is an example:

```
[globals]
  config-managers = ["a", "b"]
...
[a]
  repos = "repo1.domain.com", "repo2.domain.com"]
  ...
  [a.repo1.domain.com]
    method = "blob"
    repo-path = "azurestoragecontainer"
    ...
    [a.repo1.domain.com.blob]
      # Alternatively to this you can et 
      storage-account-name = "blobstorageaccountname"
```
