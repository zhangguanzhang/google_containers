package cmd

import (
	"encoding/binary"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	bolt "go.etcd.io/bbolt"
	"go/types"
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
	db, err := bolt.Open(args[0], 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatalf("open the boltdb file %s error: %v", args[0], err)
	}
	defer db.Close()
	if err := db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			c := b.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				if len(v) != int(types.Uint32) {
					fmt.Printf("wrong: bucket:%s key=%s\n", name, k)
					continue
				}

				fmt.Printf("bucket:%-35s key=%-65s, value=%v\n", name, k, binary.LittleEndian.Uint32(v))
			}
			return nil
		})
	}); err != nil {
		log.Fatal(err)
	}
}
