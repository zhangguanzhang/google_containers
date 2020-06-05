#!/bin/bash
set -e


mkdir ${HOME}/sync

[[ -z "$USE_DOCKER_BIN" ]] && {
  cp imgsync ${HOME}/sync/
} || {
  docker run --rm -d --name tool zhangguanzhang/google_containers_imgsync sleep 20
  docker cp tool:/imgsync ${HOME}/sync/
}

docker run --rm -d --name db zhangguanzhang/google_containers_db sleep 20
docker cp db:/bolt.db ${HOME}/sync

docker kill db
