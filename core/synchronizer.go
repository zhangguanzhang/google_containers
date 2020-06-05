package core

import (
	"context"
	"errors"
	"fmt"
	types2 "github.com/docker/docker/api/types"
	"github.com/docker/docker/registry"
	"github.com/spf13/cobra"
	bolt "go.etcd.io/bbolt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
	"github.com/panjf2000/ants/v2"

	log "github.com/sirupsen/logrus"
)

type auth struct {
	User string `json:"user" yaml:"user"`
	Pass string `json:"pass" yaml:"pass"`
}

type DomainIns struct {
	Auth auth `json:"auth,omitempty"`
}

type SyncOption struct {
	Auth          auth          `json:"auth" yaml:"auth,omitempty"`
	CmdTimeout    time.Duration // command timeout
	SingleTimeout time.Duration // Sync single image timeout
	LiveInterval  time.Duration
	LoginRetry    uint8
	Limit         int // Images sync process limit
	ConfPath      string
	PushRepo      string
	PushNS        string
	QueryLimit    int // Query Gcr images limit
	Retry         int
	RetryInterval time.Duration
	Report        bool // Report sync result
	Ctx           context.Context
	CheckSumer
	Closer func() error
	DbFile string
}

func (s *SyncOption) PreRun(cmd *cobra.Command, args []string) {

	if s.Auth.User == "" && s.Auth.Pass == "" {
		log.Fatal("username or pass must not be empty")
	}
	if s.PushNS == "" {
		s.PushNS = s.Auth.User
	}

	if err := s.Verify(); err != nil {
		log.Fatal(err)
	}

	log.Infof("Login Succeeded for %s", s.PushRepo)

	db, err := bolt.Open(s.DbFile, 0600, &bolt.Options{Timeout: 3 * time.Second})
	if err != nil {
		log.Fatalf("open the boltdb file %s error: %v", s.DbFile, err)
	}
	s.Closer = db.Close
	s.CheckSumer = NewBolt(db)
}

func (s *SyncOption) Verify() error {
	authConf := &types2.AuthConfig{
		Username: s.Auth.User,
		Password: s.Auth.Pass,
	}

	// https://github.com/moby/moby/blob/c3b3aedfa4ad51de0a2ebfd08a716f585390b512/daemon/daemon.go#L714
	// https://github.com/moby/moby/blob/master/daemon/auth.go

	if s.PushRepo == registry.IndexName {
		authConf.ServerAddress = registry.IndexServer
	} else {
		authConf.ServerAddress = s.PushRepo
	}
	if !strings.HasPrefix(authConf.ServerAddress, "https://") && !strings.HasPrefix(authConf.ServerAddress, "http://") {
		authConf.ServerAddress = "https://" + authConf.ServerAddress
	}

	RegistryService, err := registry.NewService(registry.ServiceOptions{})
	if err != nil {
		return err
	}

	var status string

	for count := s.LoginRetry; count > 0; count-- {
		if status, _, err = RegistryService.Auth(s.Ctx, authConf, ""); err != nil && strings.Contains(err.Error(), "timeout") {
			<-time.After(time.Second * 1)
		} else {
			break
		}
	}

	if err != nil {
		return err
	}

	if !strings.Contains(status, "Succeeded") {
		return fmt.Errorf("cannot get status")
	}
	return nil
}

func Run(opt *SyncOption, namespace []string) {

	defer opt.Closer()
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

	if err := opt.CheckSumer.CreatBucket("gcr.io"); err != nil {
		log.Error(err)
	}

	g := &Gcr{Option: opt}

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

	for _, ns := range namespace {
		g.Sync(ns)
	}

}

type TagsOption struct {
	ctx     context.Context
	Timeout time.Duration
}

func SyncImages(ctx context.Context, imgs Images, opt *SyncOption) Images {
	//imgs := batchProcess(images, opt)
	//logrus.Infof("starting sync images, image total: %d", len(imgs))

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
			case <-ctx.Done():
			default:
				log.Debugf("process image: gcr.io/%s", imgs[k].String())
				newSum, needSync := checkSync(imgs[k], opt)
				if !needSync {
					return
				}

				rerr := retry(opt.Retry, opt.RetryInterval, func() error {
					return sync2DockerHub(imgs[k], opt)
				})
				if rerr != nil {
					imgs[k].Err = rerr
					log.Errorf("failed to process image gcr.io/%s, error: %s", imgs[k].String(), rerr)
					return
				}
				imgs[k].Success = true

				//写入校验值
				if sErr := opt.CheckSumer.Save(imgs[k].String(), newSum); sErr != nil {
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

	srcImg := fmt.Sprintf("gcr.io/%s", image.String())
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

//已经同步过了没
func checkSync(image *Image, opt *SyncOption) (uint32, bool) {
	var (
		bodySum uint32
		diff    bool
	)
	imgFullName := fmt.Sprintf("gcr.io/%s", image.String())
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

	// db查询校验值是否相等
	diff, err = opt.CheckSumer.Diff(imgFullName, bodySum)
	if err != nil {
		image.Err = err
		log.Errorf("failed to get image [%s] checkSum, error: %s", imgFullName, err)
		return 0, false
	}

	if !diff { //相同
		image.Success = true
		image.CacheHit = true
		log.Debugf("image [%s] not changed, skip sync...", imgFullName)
		return 0, false
	}

	return bodySum, true
}
