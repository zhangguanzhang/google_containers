package cmd

import (
	"encoding/binary"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	bolt "go.etcd.io/bbolt"
	"go/types"
	"imgsync/core"
	"strings"
	"time"
)

func NewCheckComamnd() *cobra.Command {
	var dbFile string
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check if the image needs to be synchronized",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			db, err := bolt.Open(dbFile, 0600, &bolt.Options{Timeout: 3 * time.Second})
			if err != nil {
				log.Fatalf("open the boltdb file %s error: %v", dbFile, err)
			}
			defer db.Close()

			if err := db.View(func(tx *bolt.Tx) error {
				return tx.ForEach(func(bName []byte, b *bolt.Bucket) error {
					c := b.Cursor()
					for k, v := c.First(); k != nil; k, v = c.Next() {
						if len(v) != int(types.Uint32) {
							log.Errorf("wrong: bucket:%s key=%s\n", bName, k)
							continue
						}

						if strings.Compare(fmt.Sprintf("%s/%s", bName, k), args[0]) == 0 {
							lValue := binary.LittleEndian.Uint32(v)
							rValue, err := core.GetManifestBodyCheckSum(args[0])
							if err != nil {
								log.Fatal(err)
							}
							fmt.Printf("%s/%s local:%d remote:%d\n", bName, k, lValue, rValue)
							break
						}

					}
					return nil
				})
			}); err != nil {
				log.Fatal(err)
			}
		},
	}

	cmd.Flags().StringVar(&dbFile, "db", "bolt.db", "the bold db file.")

	return cmd
}
