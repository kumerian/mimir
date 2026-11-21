#!/bin/sh
set -eu

chown mimir:mimir /var/lib/mimir
chmod 0700 /var/lib/mimir
exec su-exec mimir:mimir /usr/local/bin/mimir "$@"
