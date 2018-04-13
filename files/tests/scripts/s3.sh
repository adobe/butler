#!/bin/bash
# Copyright 2017 Adobe. All rights reserved.
# This file is licensed to you under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License. You may obtain a copy
# of the License at http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software distributed under
# the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
# OF ANY KIND, either express or implied. See the License for the specific language
# governing permissions and limitations under the License.

# Convert strings to arrays
BUTLER_S3_TEST_CONFIGS=($BUTLER_S3_TEST_CONFIGS)
BUTLER_S3_TEST_RESPONSES=($BUTLER_S3_TEST_RESPONSES)

if [ -z "$AWS_ACCESS_KEY_ID" -o -z "$AWS_SECRET_ACCESS_KEY" ]; then
    echo "WARNING: must set your AWS_ACCCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables"
fi

if [ -z "$BUTLER_S3_TEST_CONFIGS" -o -z "$BUTLER_S3_TEST_RESPONSES" ]; then
    echo "WARNING: must set your BUTLER_S3_TEST_CONFIGS and BUTLER_S3_TEST_RESPONSES environment variables"
fi

ITER=0
for config in "${BUTLER_S3_TEST_CONFIGS[@]}"
do
    cmd="/butler -config.path $config -config.retrieve-interval 10 -log.level debug -s3.region $BUTLER_S3_TEST_REGION"
    echo " [testing $cmd]"
    $cmd -test
    res=$?
    if [ $res -ne ${BUTLER_S3_TEST_RESPONSES[$((ITER))]} ]; then
        echo "unexpected result for $cmd. res=$res expecting res to be ${BUTLER_S3_TEST_RESPONSES[$((ITER))]} at array place $ITER (starting at 0)"
        exit 1
    fi
    ITER=$((ITER+1))
    echo " [done testing $cmd]"
    echo
done
