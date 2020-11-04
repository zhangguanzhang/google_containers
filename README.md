## 关于

可以去[阿里云镜像仓库市场](https://cr.console.aliyun.com/images/cn-hangzhou/google_containers/kube-apiserver/detail)看看`registry.aliyuncs.com/google_containers`实际上这个ns是用户创建的。

由于这个ns的所有者同步太慢了，我便写了这个工具来时刻同步

本仓库只同步`k8s.gcr.io` ==> `registry.aliyuncs.com/k8sxio`，目前`gcr.io/google_containers`已经不和`k8s.gcr.io`一致了，所以目前只同步`k8s.gcr.io`

目前已经同步完了，可以看[travis-ci的运行状态](https://travis-ci.com/github/zhangguanzhang/google_containers)


查看镜像列表的话去[阿里云镜像仓库市场](https://cr.console.aliyun.com/cn-hangzhou/instances/images) 

登录后在搜索框那输入`k8sxio/`搜索即可，也可以看单独的镜像例如`k8sxio/kube-apiserver`

## 特性

- **不依赖 Docker 运行**
- **同步期间不占用本地磁盘空间(直接通过标准库转发镜像)**
- **可控的并发同步(优雅关闭/可调节并发数量)**


- 核心拷贝方法引用的[containers/image](https://github.com/containers/image)，部分代码借鉴了[mritd](https://github.com/mritd/imgsync)

- 利用 boltdb 存储每个镜像 manifest 信息的 crc32 校验值，通过比对判断是否需要同步，而不是每次请求目标仓库
- 把 boltdb 文件放 docker镜像里存在 dockerhub 上，利用 travis 的api来重启 travis-ci 的 runner 来同步


## 用法

编译的话记得带上tag，关闭CGO
```shell
go build -tags=containers_image_openpgp main.go
```

sync命令是同步的命令，sum是查看boltdb文件的信息
```cassandraql
imgsync sync --help
            
    Sync docker images.
    
    Usage:
      imgsync sync [flags]
    
    Flags:
          --addition-ns stringArray    addition ns to sync (default [])
          --command-timeout duration   timeout for the command execution.
          --db string                  the boltdb file (default "bolt.db")
      -h, --help                       help for sync
          --img-timeout duration       sync single image timeout. (default 15m0s)
          --live-interval duration     live output for travis-ci.
          --login-retry uint8          login retry when timeout. (default 2)
      -p, --password string            The password to push.
          --process-limit int          sync process limit. (default 2)
          --push-ns string             the ns push to
          --push-to string             the repo push to (default "docker.io")
          --query-limit int            http query limit. (default 10)
          --retry int                  retry count while err. (default 4)
          --retry-interval duration    retry interval while err. (default 4s)
      -u, --user string                The username to push.
    
    Global Flags:
          --debug   debug mode

```

### 示例

在travis-ci的设置那里设置好环境变量来控制运行的一些属性，也可以把镜像同步到自己的内网仓库上
```cassandraql
  ${HOME}/sync/imgsync sync 
  --db ${HOME}/sync/bolt.db 
  --push-to registry.aliyuncs.com 
  --password ${PASS} 
  --push-ns=k8sxio 
  --user zhangguanzhang@qq.com 
  --command-timeout ${TMOUT} 
  --process-limit ${PROCESS:=2}
  --img-timeout ${IMG_TMOUT:=10m} 
  --live-interval ${LIVE:=9m20s}
  --login-retry ${LOGIN_RETRY:=2}
  --debug=${DEBUG:=false} 
```