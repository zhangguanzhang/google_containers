package core

import (
	"fmt"
)

type Image struct {
	NameSpaces string // username
	Name       string // basename
	Tag        string

	Success  bool
	CacheHit bool
	Err      error
}

func (img *Image) String() string {
	return fmt.Sprintf("%s/%s:%s", img.NameSpaces, img.Name, img.Tag)
}

type Images []*Image

func (imgs Images) Len() int           { return len(imgs) }
func (imgs Images) Less(i, j int) bool { return imgs[i].String() < imgs[j].String() }
func (imgs Images) Swap(i, j int)      { imgs[i], imgs[j] = imgs[j], imgs[i] }
