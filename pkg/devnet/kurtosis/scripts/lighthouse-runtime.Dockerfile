# The pinned ethereum-package passes the explicit lighthouse executable to VCs.
# Observer services supply their own entrypoint. Keep this image command-neutral.
FROM ubuntu:24.04
COPY lighthouse /usr/local/bin/lighthouse
ARG SOURCE_COMMIT
LABEL source_commit="$SOURCE_COMMIT"
