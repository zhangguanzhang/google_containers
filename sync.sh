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

    SYNC_IMAGE_NAME=gcr.io/cloud-datalab/datalab
    image_name=${SYNC_IMAGE_NAME##*/}
    MY_REPO_IMAGE_NAME=${Prefix}${image_name}
    while read tag;do
    #处理latest标签
    echo $tag
        [ "$(docker images|wc -l)" -ge 5 ] && { wait;img_clean $domain $namespace $image_name ; }
        [[ "$(hub_tag_exist $MY_REPO_IMAGE_NAME $tag)" == 'null' ]] && continue
        read -u5
        {
            [ -n "$tag" ] && image_tag $SYNC_IMAGE_NAME $tag $MY_REPO/$MY_REPO_IMAGE_NAME
            echo >&5
        }&
    done < <($@_tag $SYNC_IMAGE_NAME | shuf)
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
