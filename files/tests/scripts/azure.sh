#!/bin/bash

# Convert strings to arrays
BUTLER_BLOB_TEST_CONFIGS=($BUTLER_BLOB_TEST_CONFIGS)
BUTLER_BLOB_TEST_RESPONSES=($BUTLER_BLOB_TEST_RESPONSES)

if [ -z "$BUTLER_STORAGE_ACCOUNT" -o -z "$BUTLER_STORAGE_TOKEN" ]; then
    echo "WARNING: must set your BUTLER_STORAGE_ACCOUNT and BUTLER_STORAGE_TOKEN environment variables"
fi

if [ -z "$BUTLER_BLOB_TEST_CONFIGS" -o -z "$BUTLER_BLOB_TEST_RESPONSES" ]; then
    echo "WARNING: must set your BUTLER_BLOB_TEST_CONFIGS and BUTLER_BLOB_TEST_RESPONSES environment variables"
fi

ITER=0
for config in "${BUTLER_BLOB_TEST_CONFIGS[@]}"
do
    cmd="/butler -config.path $config -config.retrieve-interval 10 -log.level debug"
    echo "[ testing $cmd"]
    $cmd -test
    res=$?
    if [ $res -ne ${BUTLER_BLOB_TEST_RESPONSES[$((ITER))]} ]; then
        echo "unexpected result for $cmd. res=$res expecting res to be ${BUTLER_BLOB_TEST_RESPONSES[$((ITER))]} at array place $ITER (starting at 0)"
        exit 1
    fi
    ITER=$((ITER+1))
    echo "[ done testing $cmd"]
    echo
done
