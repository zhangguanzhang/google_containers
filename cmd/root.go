package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"imgsync/core"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

func NewImgSyncCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "imgsync",
		Short: "Docker image sync tool",
		Long: `
Docker image sync tool.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	return rootCmd
}

func Execute() error {
	var debug bool
	rootCmd := NewImgSyncCommand()
	initLog := func() {
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
		if debug {
			log.SetLevel(log.DebugLevel)

		}
	}
	cobra.OnInitialize(initLog)
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "debug mode")
	rootCmd.SetVersionTemplate(versionTpl())
	rootCmd.AddCommand(NewSyncComamnd(nil))
	rootCmd.AddCommand(NewSumCommand())

	return rootCmd.Execute()
}

var (
	Version      string
	gitCommit    string
	gitTreeState = ""                     // state of git tree, either "clean" or "dirty"
	buildDate    = "1970-01-01T00:00:00Z" // build date, output of $(date +'%Y-%m-%dT%H:%M:%S')
)

func versionTpl() string {
	return fmt.Sprintf(`Name: imgsync
Version: %s
CommitID: %s
GitTreeState: %s
BuildDate: %s
GoVersion: %s
Compiler: %s
Platform: %s/%s
`, Version, gitCommit, gitTreeState, buildDate, runtime.Version(), runtime.Compiler, runtime.GOOS, runtime.GOARCH)
}

func boot(opt *core.SyncOption, namespace string) {
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

	g := &core.Gcr{Option: opt}

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

	g.Sync(namespace)
}
