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


add_yum_repo() {
cat > /etc/yum.repos.d/google-cloud-sdk.repo <<EOF
[google-cloud-sdk]
name=Google Cloud SDK
baseurl=https://packages.cloud.google.com/yum/repos/cloud-sdk-el7-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
       https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOF
}

add_apt_source(){
    export CLOUD_SDK_REPO="cloud-sdk-$(lsb_release -c -s)"
    echo "deb http://packages.cloud.google.com/apt $CLOUD_SDK_REPO main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list
    curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
}

install_sdk() {
    local OS_VERSION=$(grep -Po '(?<=^ID=")\w+' /etc/os-release)
    local OS_VERSION=${OS_VERSION:-ubuntu}
    if [[ $OS_VERSION =~ "centos" ]];then
        if ! [ -f /etc/yum.repos.d/google-cloud-sdk.repo ];then
            add_yum_repo
            yum -y install google-cloud-sdk
        else
            echo "gcloud is installed"
        fi
    elif [[ $OS_VERSION =~ "ubuntu" ]];then
        if ! [ -f /etc/apt/sources.list.d/google-cloud-sdk.list ];then
            add_apt_source
            sudo apt-get -y update && sudo apt-get -y install google-cloud-sdk
        else
             echo "gcloud is installed"
        fi
    fi
}

auth_sdk(){
    local AUTH_COUNT=$(gcloud auth list --format="get(account)"|wc -l)
    [ "$AUTH_COUNT" -eq 0 ] && gcloud auth activate-service-account --key-file=$HOME/gcloud.config.json ||
        echo "gcloud service account is exsits"
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

google_name(){
    gcloud container images list --repository=$@ --format="value(NAME)"
}
google_tag(){
    gcloud container images list-tags $@  --format="get(TAGS)" --filter='tags:*' | sed 's#;#\n#g'
}
google_latest_digest(){
    gcloud container images list-tags --format='get(DIGEST)' $@ --filter="tags=latest"
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
        [[ $(df -h| awk  '$NF=="/"{print +$5}') -ge "$max_per" || -n $(sync_commit_check) ]] && { wait;img_clean $domain $namespace $image_name ; }
        [ "$(hub_tag_exist $MY_REPO_IMAGE_NAME $tag)" == 'null' ]] && continue
        read -u5
        {
            [ -n "$tag" ] && image_tag $SYNC_IMAGE_NAME $tag $MY_REPO/$MY_REPO_IMAGE_NAME
            echo >&5
        }&
    done < <($@_tag $SYNC_IMAGE_NAME | shuf)
    wait
    img_clean $domain $namespace $image_name 

}

sync_commit_check(){
    [[ $(( (`date +%s` - start_time)/60 )) -gt 40 || -n "$(docker images | awk '$NF~"GB"')" ]] &&
        echo ture || false
}

# img_name tag
hub_tag_exist(){
    curl -s https://hub.docker.com/v2/repositories/${MY_REPO}/$1/tags/$2/ | jq -r .name
}


trvis_live(){
    [ $(( (`date +%s` - live_start_time)/60 )) -ge 8 ] && { live_start_time=$(date +%s);echo 'for live in the travis!'; }
}


main(){
    install_sdk
    auth_sdk
    Multi_process_init $(( max_process * 4 ))
    live_start_time=$(date +%s)

    exec 5>&-;exec 5<&-

    Multi_process_init $max_process

    image_pull gcr.io/cloud-datalab google


    exec 5>&-;exec 5<&-
}

main
