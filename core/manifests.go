package core

import (
	"context"
	"fmt"
	"hash/crc32"

	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/types"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// crc32 hash and err
func GetManifestBodyCheckSum(imageName string) (uint32, error) {
	srcRef, err := docker.ParseReference("//" + imageName)
	if err != nil {
		return 0, err
	}

	sourceCtx := &types.SystemContext{DockerAuthConfig: &types.DockerAuthConfig{}}
	imageSrcCtx, imageSrcCancel := context.WithTimeout(context.Background(), DefaultCtxTimeout)
	defer imageSrcCancel()
	src, err := srcRef.NewImageSource(imageSrcCtx, sourceCtx)
	if err != nil {
		return 0, err
	}

	getManifestCtx, getManifestCancel := context.WithTimeout(context.Background(), DefaultCtxTimeout)
	defer getManifestCancel()
	mbs, _, err := src.GetManifest(getManifestCtx, nil)
	if err != nil {
		return 0, err
	}

	mType := manifest.GuessMIMEType(mbs)
	if mType == "" {
		return 0, fmt.Errorf("faile to parse image [%s] manifest type", imageName)
	}

	if mType != manifest.DockerV2ListMediaType && mType != imgspecv1.MediaTypeImageIndex {
		_, err = manifest.FromBlob(mbs, mType)
		if err != nil {
			return 0, err
		}
	}

	return crc32.ChecksumIEEE(mbs), nil

}
