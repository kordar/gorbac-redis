module github.com/kordar/gorbac-redis

go 1.18

replace github.com/kordar/gorbac => ../../github.com/gorbac

require (
	github.com/kordar/gorbac v1.0.7
	github.com/redis/go-redis/v9 v9.6.1
	github.com/spf13/cast v1.6.0
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/kordar/gologger v0.0.8 // indirect
)
