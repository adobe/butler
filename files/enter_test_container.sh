#!/bin/bash

RUNNING_IMAGE=`docker ps -f "name=${TESTER_TAG}" -q`
if [ x${RUNNING_IMAGE} = "x" ]; then
    docker run --rm -i --name ${TESTER_TAG} ${TESTER_TAG} /bin/bash
    ret=$?
else
    docker exec -i $RUNNING_IMAGE /bin/bash
    ret=$?
fi
exit $ret
