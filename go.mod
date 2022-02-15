module github.com/wetrycode/tegenaria

go 1.16

require (
	github.com/bits-and-blooms/bitset v1.2.1 // indirect
	github.com/bits-and-blooms/bloom/v3 v3.1.0
	github.com/gin-gonic/gin v1.7.7
	github.com/go-kiss/monkey v0.0.0-20210912230757-40cda447d0e3
	github.com/google/uuid v1.3.0
	github.com/json-iterator/go v1.1.12
	github.com/sirupsen/logrus v1.8.1
	github.com/spaolacci/murmur3 v1.1.0
	github.com/spf13/viper v1.10.1
	github.com/wxnacy/wgo v1.0.4
	github.com/yireyun/go-queue v0.0.0-20210520035143-72b190eafcba
	golang.org/x/net v0.0.0-20211216030914-fe4d6282115f
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
)

retract (
	v0.2.6
	v0.2.5
	v0.2.4
	v0.2.3
	v0.2.2
	v0.2.1
	v0.2.0
	v0.1.3
)
