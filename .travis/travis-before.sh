#!/bin/bash
set -e


mkdir ${HOME}/sync

cp imgsync ${HOME}/sync/

docker run --rm -d --name db zhangguanzhang/google_containers_db sleep 20
docker cp db:/bolt.db ${HOME}/sync

docker kill db
