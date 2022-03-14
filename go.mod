module github.com/skatteetaten/architect/v2

go 1.16

require (
	github.com/docker/distribution v2.7.1+incompatible
	github.com/hashicorp/go-version v1.1.0
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1.0.20200206005728-dd78d7521eee // indirect
	github.com/openshift/api v3.9.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.3
	github.com/stretchr/testify v1.7.0
	k8s.io/api v0.23.4 // indirect
)

replace (
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2
	golang.org/x/text => golang.org/x/text v0.3.4
)
