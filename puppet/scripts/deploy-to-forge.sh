#!/bin/bash
set -x
export PATH=$PATH:/usr/local/bin:/usr/bin:/usr/sbin:/bin

start_time=`date +%s`
MODULEFULLPATH="$WORKSPACE/puppet/ft-brightcove_notifier"

/usr/local/bin/forge-admin.py --publish --source "$MODULEFULLPATH"
ERROR_CODE=$?
if [[ $ERROR_CODE -ne 0 ]]; then
    echo -e "Attempt to publish $MODULEFULLPATH failed with code $ERROR_CODE.\n"
    exit 255
fi
echo ""
exit 0
