package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/types"
	jsoniter "github.com/json-iterator/go"
	"github.com/panjf2000/ants/v2"
	"github.com/parnurzeal/gorequest"
	log "github.com/sirupsen/logrus"
)

const (
	imgList = "https://k8s.gcr.io/v2/tags/list"
	DefaultHTTPTimeout        = 15 * time.Second
	repo = "k8s.gcr.io/"
)


// baseName，不是full name
func NSImages(op *SyncOption) ([]string, error) {
	log.Info("get k8s.gcr.io public images...")
	resp, body, errs := gorequest.New().
		Timeout(DefaultHTTPTimeout).
		Retry(op.Retry, op.RetryInterval).
		Get(imgList).
		EndBytes()
	if errs != nil {
		return nil, fmt.Errorf("%s", errs)
	}

	defer func() { _ = resp.Body.Close() }()

	var imageNames []string
	err := jsoniter.UnmarshalFromString(jsoniter.Get(body, "child").ToString(), &imageNames)
	if err != nil {
		return nil, err
	}

	if len(op.AdditionNS) > 0 {
		log.Debugf("AdditionNS: %v", op.AdditionNS)
	}

	for _, v := range op.AdditionNS {
		resp, body, errs := gorequest.New().
			Timeout(DefaultHTTPTimeout).
			Retry(op.Retry, op.RetryInterval).
			Get(fmt.Sprintf("https://k8s.gcr.io/v2/%s/tags/list", v)).
			EndBytes()
		if errs != nil {
			log.Errorf("%s", errs)
			continue
		}

		defer func() { _ = resp.Body.Close() }()

		nsImageNames := []string{}
		err := jsoniter.UnmarshalFromString(jsoniter.Get(body, "child").ToString(), &nsImageNames)
		if err != nil {
			log.Error(errs)
			continue
		}
		for k := range nsImageNames {
			nsImageNames[k] = v + "/" + nsImageNames[k]
		}
		imageNames = append(imageNames, nsImageNames...)
	}

	return imageNames, nil
}

//并发获取k8s.gcr.io/$imgName:tag写入chan
func ImageNames(opt *SyncOption) (Images, error) {

	publicImageNames, err := NSImages(opt)
	if err != nil {
		return nil, err
	}

	log.Infof("sync ns count: %d in k8s.gcr.io", len(publicImageNames))

	pool, err := ants.NewPool(opt.QueryLimit, ants.WithPreAlloc(true), ants.WithPanicHandler(func(i interface{}) {
		log.Error(i)
	}))
	if err != nil {
		log.Fatalf("failed to create goroutines pool: %s", err)
	}

	//并发写入镜像信息结构体
	var images Images
	imgCh := make(chan Image, opt.QueryLimit)
	err = pool.Submit(func() {
		for image := range imgCh {
			img := image
			images = append(images, &img)
		}
	})
	if err != nil {
		log.Fatalf("failed to submit task: %s", err)
	}


	imgGetWg := new(sync.WaitGroup)
	imgGetWg.Add(len(publicImageNames))

	for _, tmpImageName := range publicImageNames {
		imageBaseName := tmpImageName
		var iName string
		iName = repo + imageBaseName

		//协程池提交生产者写入channel
		err = pool.Submit(func() {
			defer imgGetWg.Done()
			select {
			case <-opt.Ctx.Done():
				log.Debugf("context done exit while %s", iName)
			default:
				log.Debugf("query image [%s] tags...", iName)
				tags, err := getImageTags(iName, TagsOption{
					ctx:     opt.Ctx,
					Timeout: time.Second * 25,
				})
				if err != nil {
					log.Errorf("failed to get image [%s] tags, error: %s", iName, err)
					return
				}
				log.Debugf("image [%s] tags count: %d", iName, len(tags))
				//构建带tag的镜像名

				for _, tag := range tags {
					imgCh <- Image{
						Name:       imageBaseName,
						Tag:        tag,
					}
				}
			}
		})
		if err != nil {
			log.Fatalf("failed to submit task: %s while %s", err, iName)
		}
	}
	imgGetWg.Wait()
	log.Infof("Complete the tag of all images, total:%d", len(images))
	pool.Release()
	close(imgCh)
	return images, nil
}


func getImageTags(imageName string, opt TagsOption) ([]string, error) {
	srcRef, err := docker.ParseReference("//" + imageName)
	if err != nil {
		return nil, err
	}
	sourceCtx := &types.SystemContext{DockerAuthConfig: &types.DockerAuthConfig{}}
	tagsCtx, tagsCancel := context.WithTimeout(opt.ctx, opt.Timeout)
	defer tagsCancel()
	return docker.GetRepositoryTags(tagsCtx, sourceCtx, srcRef)
}


func retry(count int, interval time.Duration, f func() error) error {
	var err error
	for ; count > 0; count-- {
		if err = f(); err != nil {
			if interval > 0 {
				<-time.After(interval)
			}
		} else {
			break
		}
	}
	return err
}
