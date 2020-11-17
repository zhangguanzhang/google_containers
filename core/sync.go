package core

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
	"github.com/panjf2000/ants/v2"
	log "github.com/sirupsen/logrus"
)

func Run(opt *SyncOption) {

	opt = opt.setDefault()

	defer func() {
		_ = opt.Closer()
	}()
	Sigs := make(chan os.Signal)

	var cancel context.CancelFunc
	opt.Ctx, cancel = context.WithCancel(context.Background())
	if opt.CmdTimeout > 0 {
		opt.Ctx, cancel = context.WithTimeout(opt.Ctx, opt.CmdTimeout)
	}

	var cancelOnce sync.Once
	defer cancel()
	go func() {
		for range Sigs {
			cancelOnce.Do(func() {
				log.Info("Receiving a termination signal, gracefully shutdown!")
				cancel()
			})
			log.Info("The goroutines pool has stopped, please wait for the remaining tasks to complete.")
		}
	}()
	signal.Notify(Sigs, syscall.SIGINT, syscall.SIGTERM)

	if err := opt.CheckSumer.CreatBucket("k8s.gcr.io"); err != nil {
		log.Error(err)
	}

	if opt.LiveInterval > 0 {
		if opt.LiveInterval >= 10*time.Minute { //travis-ci 10分钟没任何输出就会被强制关闭
			opt.LiveInterval = 9 * time.Minute
		}
		go func() {
			for {
				select {
				case <-opt.Ctx.Done():
					return
				case <-time.After(opt.LiveInterval):
					log.Info("Live output for in travis-ci")
				}
			}
		}()
	}
	//
	//for _, ns := range namespace {
	//	g.Sync(ns)
	//}
	if err := Sync(opt); err != nil {
		log.Fatal(err)
	}
}

func Sync(opt *SyncOption) error {
	allImages, err := ImageNames(opt)
	if err != nil {
		return err
	}

	imgs := SyncImages(allImages, opt)
	log.Info("sync done")
	report(imgs)
	return nil
}

func SyncImages(imgs Images, opt *SyncOption) Images {

	processWg := new(sync.WaitGroup)
	processWg.Add(len(imgs))

	if opt.Limit == 0 {
		opt.Limit = DefaultLimit
	}

	pool, err := ants.NewPool(opt.Limit, ants.WithPreAlloc(true), ants.WithPanicHandler(func(i interface{}) {
		log.Error(i)
	}))

	if err != nil {
		log.Fatalf("failed to create goroutines pool: %s", err)
	}
	sort.Sort(imgs)
	for i := 0; i < len(imgs); i++ {
		k := i
		err = pool.Submit(func() {
			defer processWg.Done()

			select {
			case <-opt.Ctx.Done():
			default:
				log.Debug("process image: ", imgs[k].String())
				newSum, needSync := checkSync(imgs[k], opt)
				if !needSync {
					return
				}

				rerr := retry(opt.Retry, opt.RetryInterval, func() error {
					return sync2DockerHub(imgs[k], opt)
				})
				if rerr != nil {
					imgs[k].Err = rerr
					log.Errorf("failed to process image %s, error: %s", imgs[k].String(), rerr)
					return
				}
				imgs[k].Success = true

				//写入校验值
				if sErr := opt.CheckSumer.Save(imgs[k].Key(), newSum); sErr != nil {
					log.Errorf("failed to save image [%s] checksum: %s", imgs[k].String(), sErr)
				}
				log.Debugf("save image [%s] checksum: %d", imgs[k].String(), newSum)
			}
		})
		if err != nil {
			log.Fatalf("failed to submit task: %s", err)
		}
	}
	processWg.Wait()
	pool.Release()
	return imgs
}

func sync2DockerHub(image *Image, opt *SyncOption) error {

	srcImg := image.String()
	destImg := fmt.Sprintf("%s/%s/%s:%s", opt.PushRepo, opt.PushNS, image.Name, image.Tag)

	log.Infof("syncing %s => %s", srcImg, destImg)

	ctx, cancel := context.WithTimeout(opt.Ctx, opt.SingleTimeout)
	defer cancel()

	policyContext, err := signature.NewPolicyContext(
		&signature.Policy{
			Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()},
		},
	)

	if err != nil {
		return err
	}
	defer func() { _ = policyContext.Destroy() }()

	srcRef, err := docker.ParseReference("//" + srcImg)
	if err != nil {
		return err
	}
	destRef, err := docker.ParseReference("//" + destImg)
	if err != nil {
		return err
	}

	sourceCtx := &types.SystemContext{DockerAuthConfig: &types.DockerAuthConfig{}}
	destinationCtx := &types.SystemContext{DockerAuthConfig: &types.DockerAuthConfig{
		Username: opt.Auth.User,
		Password: opt.Auth.Pass,
	}}

	log.Debugf("copy %s to %s ...", image.String(), opt.PushRepo)
	_, err = copy.Image(ctx, policyContext, destRef, srcRef, &copy.Options{
		SourceCtx:          sourceCtx,
		DestinationCtx:     destinationCtx,
		ImageListSelection: copy.CopyAllImages,
	})
	log.Debugf("%s copy done, error is %v.", srcImg, err)
	return err
}

//已经同步过了没
func checkSync(image *Image, opt *SyncOption) (uint32, bool) {
	var (
		bodySum uint32
		diff    bool
	)
	imgFullName := image.String()
	err := retry(opt.Retry, opt.RetryInterval, func() error {
		var mErr error
		bodySum, mErr = GetManifestBodyCheckSum(imgFullName)
		if mErr != nil {
			return mErr
		}
		if bodySum == 0 {
			return errors.New("checkSum is 0, maybe resp body is nil")
		}
		return nil
	})

	if err != nil {
		image.Err = err
		log.Errorf("failed to get image [%s] manifest, error: %s", imgFullName, err)
		return 0, false
	}

	// db查询校验值是否相等，只同步一个ns下镜像，所以bucket的key只用baseName:tag
	diff, err = opt.CheckSumer.Diff(image.Key(), bodySum)
	if err != nil {
		image.Err = err
		log.Errorf("failed to get image [%s] checkSum, error: %s", imgFullName, err)
		return 0, false
	}

	log.Debugf("%s diff:%v", imgFullName, diff)

	if !diff { //相同
		image.Success = true
		image.CacheHit = true
		log.Debugf("image [%s] not changed, skip sync...", imgFullName)
		return 0, false
	}

	return bodySum, true
}

func report(images Images) {

	var successCount, failedCount, cacheHitCount int
	var report string

	for _, img := range images {
		if img.Success {
			successCount++
			if img.CacheHit {
				cacheHitCount++
			}
		} else {
			failedCount++
		}
	}
	report = fmt.Sprintf(`========================================
>> Sync Repo: k8s.gcr.io
>> Sync Total: %d
>> Sync Failed: %d
>> Sync Success: %d
>> CacheHit: %d`, len(images), failedCount, successCount, cacheHitCount)
	fmt.Println(report)
}
