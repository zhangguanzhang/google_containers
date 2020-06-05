package core

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"

	"github.com/parnurzeal/gorequest"

	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
)

type Gcr struct {
	Option *SyncOption
}

//返回ns下所有镜像名且带tag
func (gcr *Gcr) Images(ctx context.Context, namespace string) Images {
	var (
		publicImageNames []string
		ns string
	)
	//先获取ns下所有镜像名
	if strings.Contains(namespace, "/") {
		nameSlice := strings.Split(namespace, "/")
		publicImageNames = []string{nameSlice[1]}
		ns = nameSlice[0]
	} else {
		ns = namespace
		publicImageNames = gcr.NSImageNames(namespace)
	}

	logrus.Debugf("start to get gcr.io/%s tags...", namespace)

	pool, err := ants.NewPool(gcr.Option.QueryLimit, ants.WithPreAlloc(true), ants.WithPanicHandler(func(i interface{}) {
		logrus.Error(i)
	}))
	if err != nil {
		logrus.Fatalf("failed to create goroutines pool: %s", err)
	}

	var images Images
	imgCh := make(chan Image, gcr.Option.QueryLimit)
	err = pool.Submit(func() {
		for image := range imgCh {
			img := image
			images = append(images, &img)
		}
	})
	if err != nil {
		logrus.Fatalf("failed to submit task: %s", err)
	}

	imgGetWg := new(sync.WaitGroup)
	imgGetWg.Add(len(publicImageNames))
	for _, tmpImageName := range publicImageNames {
		imageBaseName := tmpImageName
		var iName string
		iName = fmt.Sprintf("%s/%s/%s", defaultGcrRepo, ns, imageBaseName)
		//	func() {
		//	gcr.ImageNames(namespace, imageBaseName, imgCh, ctx, imgGetWg)
		//}
		err = pool.Submit(func() {
			defer imgGetWg.Done()
			select {
			case <-ctx.Done():
				logrus.Debugf("context done exit while %s", imageBaseName)
			default:
				logrus.Debugf("query image [%s] tags...", iName)
				tags, terr := getImageTags(iName, TagsOption{
					ctx:     gcr.Option.Ctx,
					Timeout: time.Second * 10,
				})
				if terr != nil {
					logrus.Errorf("failed to get image [%s] tags, error: %s", iName, terr)
					return
				}
				logrus.Debugf("image [%s] tags count: %d", iName, len(tags))
				//构建带tag的镜像名
				for _, tag := range tags {
					imgCh <- Image{
						NameSpaces: ns,
						Name:       imageBaseName,
						Tag:        tag,
					}
				}
			}
		})
		if err != nil {
			logrus.Fatalf("failed to submit task: %s", err)
		}
		logrus.Debugf("Complete the tag of the image: %s", iName)
	}

	imgGetWg.Wait()
	logrus.Infof("Complete the tag of all images under ns: gcr.io/%s", namespace)
	pool.Release()
	close(imgCh)
	return images
}

//返回ns下镜像列表, gcr.io/$ns/*
func (gcr *Gcr) NSImageNames(ns string) []string {
	logrus.Infof("get gcr.io/%s public images...", ns)

	var addr string
	addr = fmt.Sprintf(gcrStandardImagesTpl, ns)

	resp, body, errs := gorequest.New().
		Timeout(DefaultHTTPTimeout).
		Retry(gcr.Option.Retry, gcr.Option.RetryInterval).
		Get(addr).
		EndBytes()
	if errs != nil {
		logrus.Fatalf("failed to get gcr/%s images, address: %s, error: %s", ns, addr, errs)
	}
	defer func() { _ = resp.Body.Close() }()

	var imageNames []string
	err := jsoniter.UnmarshalFromString(jsoniter.Get(body, "child").ToString(), &imageNames)
	if err != nil {
		logrus.Fatalf("failed to get gcr/%s images, address: %s, error: %s", ns, addr, err)
	}
	return imageNames
}

func (gcr *Gcr) ImageNames(ns, baseName string, imgCh chan<- Image, ctx context.Context, imgGetWg *sync.WaitGroup) {
	iName := fmt.Sprintf("%s/%s/%s", defaultGcrRepo, ns, baseName)
	defer imgGetWg.Done()
	select {
	case <-ctx.Done():
		logrus.Debugf("context done exit while %s", iName)
	default:
		logrus.Debugf("query image [%s] tags...", iName)
		tags, err := getImageTags(iName, TagsOption{
			ctx:     gcr.Option.Ctx,
			Timeout: time.Second * 5,
		})
		if err != nil {
			logrus.Errorf("failed to get image [%s] tags, error: %s", iName, err)
			return
		}
		logrus.Debugf("image [%s] tags count: %d", iName, len(tags))
		//构建带tag的镜像名

		for _, tag := range tags {
			imgCh <- Image{
				NameSpaces: ns,
				Name:       baseName,
				Tag:        tag,
			}
		}
	}
}

func (gcr *Gcr) Sync(namespace string) {

	gcrImages := gcr.setDefault().Images(gcr.Option.Ctx, namespace)
	logrus.Infof("sync images count: %d in gcr.io/%s", len(gcrImages), namespace)
	//logrus.Fatal(gcrImages)
	imgs := SyncImages(gcr.Option.Ctx, gcrImages, gcr.Option)
	report(imgs, namespace)
}

func (gcr *Gcr) setDefault() *Gcr {
	//命令行指定为0会阻塞goroutine的pool
	if gcr.Option.QueryLimit < 2 {
		gcr.Option.QueryLimit = 2
	}
	//gcr.namespace = opt.NameSpace
	return gcr
}

func report(images Images, ns string) {

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
>> Sync Repo: gcr.io/%s
>> Sync Total: %d
>> Sync Failed: %d
>> Sync Success: %d
>> CacheHit: %d`, ns, len(images), failedCount, successCount, cacheHitCount)
	fmt.Println(report)
}
