package core

import (
	"encoding/binary"
	"fmt"
	"go/types"
	"os"

	bolt "github.com/etcd-io/bbolt"
	log "github.com/sirupsen/logrus"
)

type CheckSumer interface {
	CreatBucket(string) error
	Diff(string, uint32) (bool, error)
	Save(string, uint32) error
}

type boltdb struct {
	db         *bolt.DB
	bucketName string // current bucket name
}

func NewBolt(db *bolt.DB) CheckSumer {
	return &boltdb{db: db}
}

func (b *boltdb) Bucket(tx *bolt.Tx) *bolt.Bucket {
	return tx.Bucket([]byte(b.bucketName))
}

func (b *boltdb) CreatBucket(domain string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		var err error
		_, err = tx.CreateBucketIfNotExists([]byte(domain))
		if err != nil {
			return fmt.Errorf("create bucket failed: %s", err)
		}
		b.bucketName = domain
		return nil
	})
}

// imageName是镜像名去掉域名部分带tag remoteSum
func (b *boltdb) Diff(imageName string, remoteSum uint32) (bool, error) {
	var (
		err      error
		SumBytes []byte
	)

	err = b.db.View(func(tx *bolt.Tx) error {
		SumBytes = b.Bucket(tx).Get([]byte(imageName))
		return nil
	})

	if err != nil {
		return false, err
	}

	if len(SumBytes) != int(types.Uint32) { //没读到数据或者，长度不对下不能使用binary的方法转uint32，大于4字节会out of range
		log.Debugf("imageName:%s, len:%d", imageName, len(SumBytes))
		return true, nil
	}

	lsum := binary.LittleEndian.Uint32(SumBytes) //和下面的Save同时使用小端或者大端
	if remoteSum != lsum {
		if os.Getenv("HASH_DIS") != "" { //环境变量来debug输出
			log.Infof("imageName:%s local:%d remote:%d", imageName, remoteSum, lsum)
		}
		return true, nil
	}

	return false, err
}

func (b *boltdb) Save(imageName string, checkSum uint32) error {
	dstBytesBuf := make([]byte, types.Uint32)
	binary.LittleEndian.PutUint32(dstBytesBuf, checkSum)
	return b.db.Update(func(tx *bolt.Tx) error {
		return b.Bucket(tx).Put([]byte(imageName), dstBytesBuf)
	})
}
