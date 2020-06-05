#!/bin/bash
set -e


mkdir ${HOME}/sync

[[ -z "$USE_DOCKER_BIN" ]] && {
  cp imgsync ${HOME}/sync/
} || {
  # 出现bug的时候直接使用docker的二进制文件，而不用提交代码触发ci
  docker run --rm -d --name tool zhangguanzhang/google_containers_imgsync sleep 20
  docker cp tool:/imgsync ${HOME}/sync/
  docker kill tool
}

docker run --rm -d --name db zhangguanzhang/google_containers_db sleep 20
docker cp db:/bolt.db ${HOME}/sync/

docker kill db
