package core

import (
	"context"
	"fmt"
	"time"
)

type Image struct {
	Name       string // basename
	Tag        string

	Success  bool
	CacheHit bool
	Err      error
}

func (img *Image) String() string {
	return fmt.Sprintf("k8s.gcr.io/%s:%s", img.Name, img.Tag)
}

func (img *Image) Key() string {
	return fmt.Sprintf("%s:%s", img.Name, img.Tag)
}

type Images []*Image

func (imgs Images) Len() int           { return len(imgs) }
func (imgs Images) Less(i, j int) bool { return imgs[i].String() < imgs[j].String() }
func (imgs Images) Swap(i, j int)      { imgs[i], imgs[j] = imgs[j], imgs[i] }


type TagsOption struct {
	ctx     context.Context
	Timeout time.Duration
}

const (
	DefaultCtxTimeout         = 5 * time.Minute
	DefaultLimit              = 20
)
