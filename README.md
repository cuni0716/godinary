# godinary
Image proxy with live resize &amp; tranformations


Install
```
git clone https://github.com/hundredrooms/godinary
```


### Docker flow:
- make build
- docker run --rm -p 3002:3002 --env-file .env -ti godinary:latest

### Development flow:
Without SSl and with gin live reloading in port 3000:
- mkdir data && cp .env.example .env
- make build-dev
- make run 

### Run tests
- make build-test

### Configuration
Variables can be passed as arguments or as env vars (uppercase and with GODINARY_ prefix)
```
$ godinary -h
Usage of godinary:
      --allow_hosts string       Domains authorized to ask godinary separated by commas (A comma at the end allows empty referers)
      --cdn_ttl string           Number of seconds images wil be cached in CDN (default "604800")
      --domain string            Domain to validate with Host header, it will deny any other request (if port is not standard must be passed as host:port)
      --fs_base string           FS option: Base dir for filesystem storage
      --gce_project string       GS option: Sentry DSN for error tracking
      --gs_bucket string         GS option: Bucket name
      --gs_credentials string    GS option: Path to service account file with Google Storage credentials
      --max_request int          Maximum number of simultaneous downloads (default 100)
      --max_request_domain int   Maximum number of simultaneous downloads per domain (default 10)
      --port string              Port where the https server listen (default "3002")
      --release string           Release hash to notify sentry
      --sentry_url string        Sentry DSN for error tracking
      --ssl_dir string           Path to directory with server.key and server.pem SSL files (default "/app/")
      --storage string           Storage type: 'gs' for google storage or 'fs' for filesystem (default "fs")
```


### Use it
```
http://localhost:3002/image/fetch/w_500/https://www.drupal.org/files/project-images/simplemeta2.png
```

Parameters:
- type fetch -> last param is target URL
- w: max width
- h: max height
- c: crop type (scale, fit and limit allowed)
- f: format (jpg, jpeg, png, gif, webp and auto allowed)
- q: quality (75 by default)

#Rabbit Worker:
To run rabbit Worker first is needed to exec "make up-dev" in one terminal.
When the enverionment and rabbit is running you can run the following commands:

1 To run the rabbit cache consumer rabbit-worker with "run-rabbit-consumer-dev"

$ rabbit.go -h
    Parameters (ENV values):
        --async_storage string      Storage Option, if 'true' ,storage will be asynchronous (default "true")
        --fs_base string            FS option: Base dir for filesystem storage
        --gce_project string        GS option: Sentry DSN for error tracking
        --gs_bucket string          GS option: Bucket name
        --gs_credentials string     GS option: Path to service account file with Google Storage credentials
        --max_rabbit_requests int   Maximum number of simultaneous downloads (default 100)
        --rabbitmq_queue string     Name of RabbitMQ queue to get images (default "core_godinary")
        --rabbitmq_url string       RabbitMQ DSN (default "amqp://guest:guest@godinary.rabbitmq:5672//")
        --release string            Release hash to notify sentry
        --sentry_url string         Sentry DSN for error tracking
        --storage string            Storage type: 'gs' for google storage or 'fs' for filesystem (default "fs")

    Al enverionment values are uppercase and with GODINARY in the beggining
    Example : GODINARY_FS_BASE="Path"

2 To enqueue elements in rabbit cache queue you can use "make run-rabbit-publisher-dev image_url='Your Image Url'"

3 If you prefer to execute a new go program inside docker image use "make sh-dev"

Finally to stop the development enverionment working use "make down-dev"