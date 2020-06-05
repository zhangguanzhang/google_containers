module imgsync

go 1.14

replace github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.6.0

require (
	github.com/containers/image/v5 v5.4.4
	github.com/docker/docker v1.4.2-0.20191219165747-a9416c67da9f
	github.com/json-iterator/go v1.1.9
	github.com/opencontainers/image-spec v1.0.2-0.20190823105129-775207bd45b6
	github.com/panjf2000/ants/v2 v2.4.1
	github.com/parnurzeal/gorequest v0.2.16
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	go.etcd.io/bbolt v1.3.4
	golang.org/x/sys v0.0.0-20200602225109-6fdc65e7d980 // indirect
	golang.org/x/tools v0.0.0-20190524140312-2c0ae7006135
	moul.io/http2curl v1.0.0 // indirect
)
