module github.com/tomcz/example-miniredis

go 1.23

toolchain go1.23.3

require (
	github.com/alicebob/miniredis/v2 v2.14.1
	github.com/digitalocean/go-workers2 v0.9.3
	github.com/gorilla/mux v1.8.0
	github.com/sirupsen/logrus v1.7.0
	golang.org/x/sync v0.10.0
)

require (
	github.com/alicebob/gopher-json v0.0.0-20200520072559-a9ecdc9d1d3a // indirect
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/go-redis/redis v6.15.9+incompatible // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/yuin/gopher-lua v0.0.0-20191220021717-ab39c6098bdb // indirect
	golang.org/x/net v0.1.0 // indirect
	golang.org/x/sys v0.1.0 // indirect
)

replace github.com/digitalocean/go-workers2 v0.9.3 => github.com/tomcz/go-workers2 v0.9.4-0.20201230054137-d1e18e2ded32
