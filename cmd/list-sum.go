package cmd

import (
	"encoding/binary"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	bolt "go.etcd.io/bbolt"
	"go/types"
	"imgsync/core"
	"time"
)

func NewSumCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sum",
		Short: "list all check sum",
		Args:  cobra.ExactArgs(1),
		Run:   listCheckSum,
	}
}

func listCheckSum(cmd *cobra.Command, args []string) {
	db, err := bolt.Open(args[0], 0600, &bolt.Options{
		Timeout: 1 * time.Second,
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("open the boltdb file %s error: %v", args[0], err)
	}
	defer db.Close()
	if err := db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(bName []byte, b *bolt.Bucket) error {
			c := b.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				if len(v) != int(types.Uint32) {
					fmt.Printf("wrong: bucket:%s key=%s\n", bName, k)
					continue
				}

				fmt.Printf("bucket:%-35s key=%-65s, value=%v\n", bName, k, binary.LittleEndian.Uint32(v))
			}
			return nil
		})
	}); err != nil {
		log.Fatal(err)
	}
}

func NewGetSumCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "gsum",
		Short: "get Sum",
		Args:  cobra.ExactArgs(1),
		Run:   getSum,
	}
}

func getSum(cmd *cobra.Command, args []string) {
	for _, image := range args {
		crc32Value, err := core.GetManifestBodyCheckSum(image)
		if err != nil {
			log.Errorf("%s|%v", image, err)
		}
		fmt.Printf("%s | %d\n", image, crc32Value)
	}
}
