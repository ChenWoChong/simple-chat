#!/usr/bin/env bash
#
# Created by chenchong on 20/09/30.
#
set -e

# if command starts with an option, prepend client
if [[ "${1:0:1}" = '-' ]]; then
    set -- client "$@"
fi
# cd workspace

# if command client only, add use default args
if [[ "$1" = 'client' ]] && [[ "$#" -eq 1 ]]; then
    exec client -conf ${CONFIG_FILE} -v ${VERBOSE} -logtostderr true
fi

exec "$@"