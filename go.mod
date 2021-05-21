module github.com/tomcz/example-miniredis

go 1.13

require (
	github.com/alicebob/miniredis/v2 v2.14.1
	github.com/digitalocean/go-workers2 v0.9.3
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/go-multierror v1.1.0
	github.com/sirupsen/logrus v1.7.0
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
)

replace github.com/digitalocean/go-workers2 v0.9.3 => github.com/tomcz/go-workers2 v0.9.4-0.20201230054137-d1e18e2ded32
