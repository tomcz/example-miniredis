# example-miniredis

Playing around with using an in-process redis using [miniredis](https://github.com/alicebob/miniredis) as the backing store for a service that relies on [go-workers2](https://github.com/digitalocean/go-workers2) for background processing.

While miniredis was created as something to be only used in unit tests, this may be useful in running a service that normally requires a redis in a totally standalone mode.

## Notes

1. [main.go#49](https://github.com/tomcz/example-miniredis/blob/master/cmd/example/main.go#L49): The `Add` value for a parameter named `class` feels a little strange to me. Changing it to an arbitrary value seems to work just as well. My suspicion is that it relates to the [Sidekiq](https://github.com/mperham/sidekiq/wiki/The-Basics#client) ruby implementation needing the name of the worker class that it should invoke when the enqueued message gets picked up for processing. I guess we have it available in case we are mixing it up with Sidekiq's ruby workers and want them to be able to pick up any jobs that we enqueue from a go producer.

2. [main.go#119](https://github.com/tomcz/example-miniredis/blob/master/cmd/example/main.go#L119): `ProcessID` is supposed to uniquely identify this instance. In an implementation that uses a real redis, and in a multi-node environment like k8s where these things can go up & down, how do we set that up?

    Maybe we can't and therefore we will lose all the in-process jobs during a restart, redeploy or a pod move. It may mean that we should keep track of the progress of the jobs ourselves, for example with checkpoints or status codes in a database, and a periodic reconciliation to make sure any abandoned jobs are restarted.

    We can also decide that we will not be running more than one worker node at a time, and then we don't need to worry about the process ID. We may still need to ensure that there is truly only one node, for example when an old one is being shut down, the new one should not be trying to process anything. This could be done with leadership election or a lock, and that is something we can also use redis for. For example: https://redis.io/topics/distlock.
