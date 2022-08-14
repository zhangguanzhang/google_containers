#!/bin/bash

[ -n "$DEBUG" ] && set -x

#脚本要存放在项目根目录
readonly PRO_ROOT=$(cd $(dirname ${BASH_SOURCE:-$0})/../; pwd -P)
source "${PRO_ROOT}/build/lib/var.sh"

read TAG_NUM LDFLAGS < <(GONELIST::SetVersion)

echo go build -o ${PRO_ROOT}/imgsync -ldflags "${LDFLAGS}" ${PRO_ROOT}/main.go


case "$1" in
  "release") # checkout到tag构建完再checkout回来
    bash ${PRO_ROOT}/build/lib/all-release.sh
    ;;
  "build") #使用master构建测试版本
    if [ -z `command -v go ` ];then
      echo go is not in PATH
      exit 1
    fi
    go build -o ${PRO_ROOT}/imgsync -tags=containers_image_openpgp -ldflags "${LDFLAGS}" ${PRO_ROOT}/main.go
    ;;
  "docker-local") #使用本地编译二进制文件打包docker和dist
    Dockerfile=Dockerfile.local
    go build -o ${PRO_ROOT}/imgsync -tags=containers_image_openpgp -ldflags "${LDFLAGS}" ${PRO_ROOT}/main.go
    ;&
  "docker") #使用容器编译和打包dist
    docker build -t zhangguanzhang/google_containers_sync:$TAG_NUM $build_arg \
      --build-arg LDFLAGS="${LDFLAGS}" -f ${Dockerfile:=Dockerfile} .
    [ -n "${DockerUser}" ] && {
      docker login -u "${DockerUser}" "${DockerPass}"
      docker push zhangguanzhang/google_containers_sync:$TAG_NUM
    }
    ;;
  "clean")

    ;;
  *)
    echo -e "\t\033[1;31m must choose one to run \033[0m"
    exit 1
    ;;
esac
