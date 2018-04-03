#!/bin/bash

RUNNING_IMAGE=`docker ps -f "name=${UNIT_TESTER_TAG}" -q`
if [ x${RUNNING_IMAGE} = "x" ]; then
    docker run -e VERSION=$VERSION --rm -it --name ${UNIT_TESTER_TAG} ${UNIT_TESTER_TAG} /bin/bash
    ret=$?
else
    docker exec -t -i $RUNNING_IMAGE /bin/bash
    ret=$?
fi
exit $ret
