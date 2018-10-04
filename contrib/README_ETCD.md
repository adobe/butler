# Butler Configuration for supporting Etcd
## Overview
The butler configuration file is similar to that explained in the README.md file. But there are a few changes that need to made for it to fetch config files from Etcd

Only the Butler configuration fields that will be different for etcd are explained here. Everything else remains the same.

## Managers / Manager Globals

### repos
The `repos` configuration option defines an array of repositories where butler is going to attempt to gather configuration files. The repositories will be folder names within the same etcd server where the config files may be stored.

#### Default Value
Empty Array

#### Example
`repos = ["Folder1", "Folder2"]`

## Repository Handler
Each Repository Handler configuration must be under the config Manager section, and must be one of the options which are defined under the `repos` option within the Manager definition.

For example, look at the following (incomplete) definition:
```
[globals]
  config-managers = ["a", "b"]
...
[a]
^^^ Manager
  repos = ["Folder1", "Folder2"]
  ...
  [a.Folder1]
  ^^^^^^^^^^^^^^^^^^^^ This is where the Repository Handler configurationn option should reside.
```

There are 4 options that can be configured under the Repository Handler configuration section.
1. method
1. repo-path
1. primary-config
1. additional-config

### method
The `method` option defines what method to use for the retrieval of configuration files. It must be `etcd`.

#### Default Value
None

#### Example
1. `method = "etcd"`

### repo-path
The `repo-path` option is the URI path to the configuration file on the local or remote filesystem. It should not be a relative path, and should not include any host information. In case of etcd this will be relative the folder names defined under `repos` and can be left blank.

#### Default Value
None

#### Example
`repo-path = "some/path"`

### primary-config
The `primary-config` option is an array of strings, that are configuration files which will get merged into the single configuration file referenced by `primary-config-name` under the Manager Globals section. You can include paths in the configuration file name, and the paths will be retrieved relative to the `repo` and `repo-path` that was defined previously. If the file is `additional/config2.yml`, then it will be retrieved from `Folder1/some/path/additional/config2.yml`

#### Default Value
[]

#### Example
`primary-config = ["config1.yml", "additional/config2.yml"]`

### additional-config
The `additional-config` option is an array strings, which are additional configuration files which will be put on the filesystem under `dest-path` as they are defined within the option. They will be retrieved relative to the `repo` and `repo-path`. If the file is called `additional/config2.yml`, then it will be retrieved from `Folder1/some/path/additional/config2.yml` and placed on the filesystem as `<dest-path>/additional/config2.yml`

#### Default Value
[]

#### Example
`additional-config = ["alerts/alerts1.yml", "extras/alertmanager.yml"]`

## Repository Handler Retrieval Options
The Repository Handler Retrieval Options must be defined under the Repository Handler using the name of the defined method.

For example, look at the following (incomplete) definition:
```
[globals]
  config-managers = ["a", "b"]
...
[a]
  repos = "Folder1", "Folder2"]
  ...
  [a.Folder1]
    method = "etcd"
    ...
    [a.Folder1.etcd]
    ^^^^^^^^^^^^^^^^^^^^^^^^^ This is where the Repository Handler Retrieval Options should reside.
```
### endpoints
Endpoints contain the etcd urls to connect to

#### Examples
endpoints = ["http://node1.example.com:2379","http://node2.example.com:2379"]
endpoints = "https://127.0.0.1:2379"

