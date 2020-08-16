package cmd

import (
	"fmt"
	"runtime"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)




func NewImgSyncCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "imgsync",
		Short: "Docker image sync tool",
		Long: `
Docker image sync tool for k8s.gcr.io.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	return rootCmd
}


func Execute() {
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
	rootCmd.AddCommand(NewSyncComamnd(nil),
		NewSumCommand(),
		NewGetSumCommand(),
		NewCheckComamnd(),
		NewReplaceComamnd(),
	)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
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