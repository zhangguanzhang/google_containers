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
    local img tag
    while read img tag;do
        docker push $img:$tag;docker rmi $img:$tag;
    done < <(docker images --format {{.Repository}}' '{{.Tag}})
}

google_tag(){
    curl -ks -XGET https://gcr.io/v2/${@#*/}/tags/list | jq -r .tags[]
}


image_pull(){
    SYNC_IMAGE_NAME=$1
    echo 'Sync the '$SYNC_IMAGE_NAME
    shift
    read domain namespace img_name < <(tr / ' '<<<$SYNC_IMAGE_NAME)
    hub_img_name=$MY_REPO/$(tr / $interval <<<$SYNC_IMAGE_NAME)
    # REPOSITORY is the name of the dir,convert the '/' to '.',and cut the last '.'

    while read tag;do
    echo $tag
        [ "$(docker images|wc -l)" -ge 2 ] && img_clean
        [[ "$(hub_tag_exist $hub_img_name $tag)" != 'null' ]] && continue
        [ -n "$tag" ] && image_tag $SYNC_IMAGE_NAME $tag $hub_img_name
    done < <(shuf tag)
    wait
    img_clean $domain $namespace $image_name 

}

# img_name tag
hub_tag_exist(){
    curl -s https://hub.docker.com/v2/repositories/$1/tags/$2/ | jq -r .name
}


main(){

    Multi_process_init $max_process

    image_pull gcr.io/cloud-datalab/datalab-gateway

    exec 5>&-;exec 5<&-
}

main 
