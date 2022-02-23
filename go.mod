module github.com/wetrycode/tegenaria

go 1.16

require (
	github.com/alicebob/miniredis v2.5.0+incompatible // indirect
	github.com/alicebob/miniredis/v2 v2.18.0
	github.com/bits-and-blooms/bitset v1.2.1 // indirect
	github.com/bits-and-blooms/bloom/v3 v3.1.0
	github.com/elliotchance/redismock v1.5.3 // indirect
	github.com/elliotchance/redismock/v8 v8.11.0
	github.com/gin-gonic/gin v1.7.7
	github.com/go-kiss/monkey v0.0.0-20210912230757-40cda447d0e3
	github.com/go-redis/redis v6.15.9+incompatible // indirect
	github.com/go-redis/redis/v8 v8.11.4
	github.com/go-redis/redismock/v8 v8.0.6 // indirect
	github.com/gomodule/redigo v1.8.8 // indirect
	github.com/google/uuid v1.3.0
	github.com/json-iterator/go v1.1.12
	github.com/rafaeljusto/redigomock/v3 v3.0.1 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/spaolacci/murmur3 v1.1.0
	github.com/spf13/viper v1.10.1
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/wxnacy/wgo v1.0.4
	github.com/yireyun/go-queue v0.0.0-20210520035143-72b190eafcba
	github.com/yuin/gopher-lua v0.0.0-20210529063254-f4c35e4016d9 // indirect
	golang.org/x/net v0.0.0-20211216030914-fe4d6282115f
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
)

// Undo the wrong version number
retract (
	v0.2.6
	v0.2.5
	v0.2.4
	v0.2.3
	v0.2.2
	v0.2.1
	v0.2.0
)
