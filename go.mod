module github.com/tomcz/example-miniredis

go 1.13

require (
	github.com/alicebob/miniredis/v2 v2.14.1
	github.com/digitalocean/go-workers2 v0.9.3
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/go-multierror v1.1.0
)

replace github.com/digitalocean/go-workers2 v0.9.3 => github.com/tomcz/go-workers2 v0.9.4-0.20201230054137-d1e18e2ded32
