# Copyright 2017-2026 Adobe. All rights reserved.
# This file is licensed to you under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License. You may obtain a copy
# of the License at http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software distributed under
# the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
# OF ANY KIND, either express or implied. See the License for the specific language
# governing permissions and limitations under the License.

########################################################################################################################
#
# Variables
#
########################################################################################################################

variable "VERSION" {
    default = "dev"
}

variable "SHA" {
    default = "unknown"
}

variable "CODECOV_TOKEN" {
    default = ""
}

# Registry configuration
variable "REGISTRY" {
    default = ""
}

variable "REGISTRY_PREFIX" {
    default = ""
}

########################################################################################################################
#
# Helper functions
#
########################################################################################################################

# Returns first param if non-blank, otherwise returns the default value
function "with_default" {
    params = [value, default]
    result = notequal(trimspace(value), "") ? trimspace(value) : default
}

# Constructs full image name with optional registry prefix
function "image_name" {
    params = [name]
    result = notequal(REGISTRY, "") ? "${REGISTRY}/${REGISTRY_PREFIX}${name}" : name
}

########################################################################################################################
#
# Groups - logical groupings of targets
#
########################################################################################################################

group "default" {
    targets = ["butler"]
}

group "all" {
    targets = ["butler", "butler-debug"]
}

group "test" {
    targets = ["test-unit", "test-accept"]
}

group "ci" {
    targets = ["butler", "test-unit"]
}

########################################################################################################################
#
# Base target - common settings inherited by all targets
#
########################################################################################################################

target "_base" {
    dockerfile = "docker/Dockerfile"
    context    = "."
    args = {
        VERSION = VERSION
        CODECOV_TOKEN = CODECOV_TOKEN
    }
    labels = {
        "org.opencontainers.image.version"     = VERSION
        "org.opencontainers.image.revision"    = SHA
        "org.opencontainers.image.source"      = "https://github.com/adobe/butler"
        "org.opencontainers.image.title"       = "Butler"
        "org.opencontainers.image.description" = "Configuration management utility"
        "org.opencontainers.image.authors"     = "matthsmi@adobe.com"
        "org.opencontainers.image.vendor"      = "Adobe Inc"
    }
    platforms = ["linux/amd64"]
}

########################################################################################################################
#
# Production targets
#
########################################################################################################################

target "butler" {
    inherits = ["_base"]
    target   = "butler"
    tags     = [
        image_name("butler:${VERSION}"),
        image_name("butler:latest"),
    ]
}

target "butler-debug" {
    inherits = ["_base"]
    target   = "butler-debug"
    tags     = [
        image_name("butler-debug:${VERSION}"),
        image_name("butler-debug:latest"),
    ]
}

########################################################################################################################
#
# Test targets
#
########################################################################################################################

target "test-unit" {
    inherits = ["_base"]
    target   = "test-unit-output"
    output   = ["type=local,dest=out/coverage"]
    # No tags needed - this outputs test results, not an image
}

target "test-accept" {
    inherits = ["_base"]
    target   = "test-accept"
    # No tags or output - tests run during build
    output   = ["type=cacheonly"]
}

########################################################################################################################
#
# Development targets
#
########################################################################################################################

# Interactive test container for unit tests
target "test-unit-interactive" {
    inherits = ["_base"]
    target   = "go-base"
    tags     = ["butler-test-unit:dev"]
    output   = ["type=docker"]
}

# Interactive test container for acceptance tests
target "test-accept-interactive" {
    inherits = ["_base"]
    target   = "test-accept-base"
    tags     = ["butler-test-accept:dev"]
    output   = ["type=docker"]
}

########################################################################################################################
#
# Multi-platform builds (optional)
#
########################################################################################################################

target "butler-multiplatform" {
    inherits  = ["butler"]
    platforms = [
        "linux/amd64",
        "linux/arm64",
    ]
}
