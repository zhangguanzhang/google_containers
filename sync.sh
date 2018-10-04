#!/bin/bash
max_process=$1
MY_REPO=zhangguanzhang
interval=.
max_per=70
#--------------------------

Multi_process_init() {
    trap 'exec 5>&-;exec 5<&-;exit 0' 2
    pipe=`mktemp -u tmp.XXXX`
    mkfifo $pipe
    exec 5<>$pipe
    rm -f $pipe
    seq $1 >&5
}


#  GCR_IMAGE_NAME  tag  REPO_IMAGE_NAME
image_tag(){
    docker pull $1:$2
    docker tag $1:$2 $3:$2
    docker rmi $1:$2
}

img_clean(){
    while read img tag;do
        docker push $img:$tag;docker rmi $img:$tag;
    done < <(docker images --format {{.Repository}}' '{{.Tag}})
}

google_tag(){
#    gcloud container images list-tags $@  --format="get(TAGS)" --filter='tags:*' | sed 's#;#\n#g'
    curl -ks -XGET https://gcr.io/v2/${@#*/}/tags/list | jq -r .tags[]
}


image_pull(){
    REPOSITORY=$1
    echo 'Sync the '$REPOSITORY
    shift
    domain=${REPOSITORY%%/*}
    namespace=${REPOSITORY##*/}
    Prefix=$domain$interval$namespace$interval
    # REPOSITORY is the name of the dir,convert the '/' to '.',and cut the last '.'

    SYNC_IMAGE_NAME=gcr.io/cloud-datalab/datalab-gateway
    image_name=${SYNC_IMAGE_NAME##*/}
    MY_REPO_IMAGE_NAME=${Prefix}${image_name}
    while read tag;do
    #处理latest标签
    echo $tag
        [ "$(docker images|wc -l)" -ge 2 ] && img_clean $domain $namespace $image_name
        [[ "$(hub_tag_exist $MY_REPO_IMAGE_NAME $tag)" == 'null' ]] && continue
        [ -n "$tag" ] && image_tag $SYNC_IMAGE_NAME $tag $MY_REPO/$MY_REPO_IMAGE_NAME
    done < <(shuf tag)
    wait
    img_clean $domain $namespace $image_name 

}

# img_name tag
hub_tag_exist(){
    curl -s https://hub.docker.com/v2/repositories/${MY_REPO}/$1/tags/$2/ | jq -r .name
}


main(){

    Multi_process_init $max_process

    image_pull gcr.io/cloud-datalab google

    exec 5>&-;exec 5<&-
}

main





hub_tag_exist(){
    curl -s https://hub.docker.com/v2/repositories/zhangguanzhang/gcr.io.cloud-datalab.datalab/tags/$1/ | jq -r .name
}

ArrayOS-Rel_AG-1 dmz 192.168.2.248 inside 192.168.0.248
ArrayOS-Rel_AG-2 dmz 192.168.2.249 inside 192.168.0.249
ArrayOS-Rel_APV outside 192.168.1.250 dmz 192.168.2.250


google::name(){
    gcloud container images list --repository=$@ --format="value(NAME)"
}


pr(){
    echo $dir:$1 > $dir/$1

}
dir=gcr.io/gcr.io/cloud-datalab/datalab-gateway

Multi_process_init() {
    trap 'exec 5>&-;exec 5<&-;exit 0' 2
    pipe=`mktemp -u tmp.XXXX`
    mkfifo $pipe
    exec 5<>$pipe
    rm -f $pipe
    seq $1 >&5
}

pr(){
    [[ "$(curl -s https://hub.docker.com/v2/repositories/zhangguanzhang/gcr.io.cloud-datalab.datalab-gateway/tags/$1/ | jq -r .name)" == null ]] && echo $1 >>newtag
}






Multi_process_init 40

while read tag;do
    [ -f $dir/$tag ] && continue
    read -u5
    {
        [[ -n "$tag" ]] && pr $tag
        echo >&5
    }&
done < tag
wait

