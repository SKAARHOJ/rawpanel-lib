module github.com/SKAARHOJ/rawpanel-lib/gorwp

go 1.18

require (
	github.com/SKAARHOJ/ibeam-lib-utils v0.0.0-20210106155940-a19b7739e1b7
	github.com/SKAARHOJ/rawpanel-lib v1.1.0
	github.com/SKAARHOJ/rawpanel-lib/topology v0.0.0-20220621113050-8b9f3c65d90e
	github.com/s00500/env_logger v0.1.24
	google.golang.org/protobuf v1.28.0
)

require (
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	golang.org/x/sys v0.0.0-20220422013727-9388b58f7150 // indirect
)

replace github.com/SKAARHOJ/rawpanel-lib => ../../rawpanel-lib

replace github.com/SKAARHOJ/rawpanel-lib/topology => ../../rawpanel-lib/topology

replace github.com/SKAARHOJ/rawpanel-lib/gorwp => ../../rawpanel-lib/gorwp
