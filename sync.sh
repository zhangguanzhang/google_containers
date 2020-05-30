#!/bin/bash
images=(
kube-proxy:v1.16.0
kube-apiserver:v1.16.0
kube-controller-manager:v1.16.0
kube-scheduler:v1.16.0
)
for image in ${images[@]} ; do
  docker pull registry.aliyuncs.com/k8sxio/$image
  docker tag registry.aliyuncs.com/k8sxio/$image gcr.io/google_containers/$image
  docker rmi registry.aliyuncs.com/k8sxio/$image
done 
