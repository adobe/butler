# Butler Configuration for supporting S3 
## Overview
The butler configuration file is similar to that explained in the README.md file. But there are a few changes that need to made to the Repository Handler for it to fetch config files from AWS S3.

Only the Butler configuration fields that will be different for S3 are explained here. Everything else remains the same.

## Managers / Manager Globals

### repos
The `repos` configuration option defines an array of repositories where butler is going to attempt to gather configuration files. The repositories will be folder names within the same S3 bucket where the config files may be stored.

#### Default Value
Empty Array

#### Example 
`repos = ["Folder1", "Folder2"]`

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
    method = "S3"
    ...
    [a.Folder1.S3]
    ^^^^^^^^^^^^^^^^^^^^^^^^^ This is wehre the Repository Handler Retrieval Options should reside.
```
### bucket
The `bucket` is the AWS S3 bucket name where the files are hosted.

#### Example
`bucket = "bucket-name"`

### region
The 'region' is the AWS region of the bucket

#### Example
`region = "us-west-2"`