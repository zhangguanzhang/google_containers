#!/bin/bash
set -e
docker login -u zhangguanzhang -p ${DOCKER_PASS}

cd $HOME
mkdir -p temp
cd temp

cp $HOME/sync/bolt.db .
ls -lh

cat>Dockerfile<<EOF
FROM zhangguanzhang/alpine
COPY bolt.db /
EOF
docker build -t zhangguanzhang/google_containers_db .
docker push zhangguanzhang/google_containers_db