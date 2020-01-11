# example-miniredis

Playing around with using an in-process redis using [miniredis](https://github.com/alicebob/miniredis) as the backing store for a service that relies on [go-workers2](https://github.com/digitalocean/go-workers2) for background processing.

While miniredis was created as something to be only used in unit tests, this may be useful in running a service that normally requires a redis in a totally standalone mode.
