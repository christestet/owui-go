#!/bin/bash
set -eo pipefail

# configuration of the version - currently we are using 0.8.5
VERSION="0.8.5"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version | -v)
      VERSION="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

PORT=8088
IMAGE="ghcr.io/open-webui/open-webui:v${VERSION}"

# filename including version for better diffing
OUTPUT_FILE="openapi-reference/${VERSION}-openapi.json"

echo "starting openwebui version $VERSION in docker"

CONTAINER_ID=$(docker run -d --rm -p ${PORT}:8080 \
  -e ENV=dev \
  -e HF_HUB_OFFLINE=1 \
  -e ENABLE_CODE_INTERPRETER=False \
  -e ENABLE_IMAGE_GENERATION=False \
  -e WEBUI_AUTH=False \
  "$IMAGE")

echo "waiting for openwebui to start"

# polling for openwebui to start
RETRIES=30
until curl -s http://localhost:${PORT}/openapi.json > /dev/null; do
    sleep 2
    RETRIES=$((RETRIES - 1))
    if [ "$RETRIES" -le 0 ]; then
        echo "Error timeout waiting for openwebui to start"
        echo "last logs:"
        docker logs "$CONTAINER_ID" --tail 20
        docker stop "$CONTAINER_ID" > /dev/null
        exit 1
    fi
    echo -n "."
done

echo "downloading openwebui openapi.json"

curl -s http://localhost:${PORT}/openapi.json | jq . > "$OUTPUT_FILE"

echo "stopping openwebui container"
docker stop "$CONTAINER_ID" > /dev/null

echo "done! openwebui openapi.json saved to $OUTPUT_FILE"