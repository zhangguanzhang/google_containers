package core

import (
	"time"
)

const (
	DefaultLimit              = 20
	DefaultSyncTimeout        = 10 * time.Minute
	DefaultCtxTimeout         = 5 * time.Minute
	DefaultHTTPTimeout        = 30 * time.Second
	DefaultGoRequestRetry     = 3
	DefaultGoRequestRetryTime = 5 * time.Second

	defaultGcrRepo = "gcr.io"

	gcrStandardImagesTpl = "https://gcr.io/v2/%s/tags/list"
)

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
