#!/usr/bin/env bash
#
# Created by chenchong on 20/09/30.
#
set -e

# if command starts with an option, prepend server
if [[ "${1:0:1}" = '-' ]]; then
    set -- server "$@"
fi
# cd workspace

# if command server only, add use default args
if [[ "$1" = 'server' ]] && [[ "$#" -eq 1 ]]; then
    exec server -conf ${CONFIG_FILE} -v ${VERBOSE} -logtostderr true
fi

exec "$@"