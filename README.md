# cmstore
Redis server and client setup that's easily deployable to examine the output from
OpenCensus after instrumenting a Go redis client https://github.com/orijtech/redigo

It features a "proxy" server as a company would have on premises, or for website crawling
and analysis. It caches already seen URLs and has the ability to purge them too

## Requirements

Name|Information
---|---
Stackdriver Tracing and Monitoring enabled accounts|Please take a look at the "Requirements" part of this article https://medium.com/@orijtech/cloud-spanner-instrumented-by-opencensus-and-exported-to-stackdriver-6ed61ed6ab4e
Redis-server instance/URL|Provide the URL to it via environment variable `REDIS_SERVER_ADDR`. By default it is `:6379`
Go|Optional if you already have the pre-built binaries that are accessible on the downloads page
Make|Optional if you are using pre-built binaries:w


## Building from source
```shell
go get -u -v github.com/orijtech/cmstore && make
```

### Running the server
```shell
GOOGLE_APPLICATION_CREDENTIALS=<yourCredentialsIfUnsetInEnvironment> ./bin/server
```

### Running the client
```shell
GOOGLE_APPLICATION_CREDENTIALS=<yourCredentialsIfUnsetInEnvironment> ./bin/client
```
