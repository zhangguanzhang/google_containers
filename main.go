package main

import (
	log "github.com/sirupsen/logrus"
	"imgsync/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
