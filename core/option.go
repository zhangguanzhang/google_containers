package core

import (
	"context"
	types2 "github.com/docker/docker/api/types"
	"github.com/docker/docker/registry"
	bolt "github.com/etcd-io/bbolt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"strings"
	"time"
)

type auth struct {
	User string `json:"user" yaml:"user"`
	Pass string `json:"pass" yaml:"pass"`
}

type SyncOption struct {
	Auth          auth
	CmdTimeout    time.Duration // command timeout
	SingleTimeout time.Duration // Sync single image timeout
	LiveInterval  time.Duration
	LoginRetry    uint8
	Limit         int // Images sync process limit
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

func (s *SyncOption) PreRun(cmd *cobra.Command, args []string) error {

	if s.Auth.User == "" && s.Auth.Pass == "" {
		log.Fatal("--user or --password must not be empty")
	}
	if s.PushNS == "" {
		s.PushNS = s.Auth.User
	}


	if err := s.Verify(); err != nil {
		return err
	}

	log.Infof("Login Succeeded for %s", s.PushRepo)

	db, err := bolt.Open(s.DbFile, 0660, &bolt.Options{Timeout: 10 * time.Second})
	if err != nil {
		return errors.Wrapf(err, "open the boltdb file %s", s.DbFile)
	}
	s.Closer = db.Close
	s.CheckSumer = NewBolt(db)
	return nil
}

func (s *SyncOption) setDefault() *SyncOption {
	if s.QueryLimit < 2 {
		s.QueryLimit = 2
	}
	if s.Limit < 2 {
		s.Limit = 2
	}
	return s
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

	var (
		status      string
		errContains = []string{"imeout", "dead"}
	)

	for count := s.LoginRetry; count > 0; count-- {
		status, _, err = RegistryService.Auth(s.Ctx, authConf, "")
		if err != nil && contains(errContains, err.Error()) {
			<-time.After(time.Second * 1)
		} else {
			break
		}
	}

	if err != nil {
		return err
	}

	if !strings.Contains(status, "Succeeded") {
		return errors.New("cannot get status")
	}
	return nil
}


func contains(s []string, searchterm string) bool {
	for _, v := range s {
		if strings.Contains(searchterm, v) {
			return true
		}
	}
	return false
}