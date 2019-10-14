# SSRF Sheriff

This is an SSRF testing sheriff written in Go. It was originally created for the [Uber H1-4420 2019 London Live Hacking Event](https://www.hackerone.com/blog/london-called-hackers-answered-recapping-h1-4420), but it is now being open-sourced for other organizations to implement and contribute back to.


## Features

- Repsond to any HTTP method (`GET`, `POST`, `PUT`, `DELETE`, etc.)
- Configurable secret token (see [base.example.yaml](config/base.example.yaml))
- Content-specific responses
  - With secret token in response body
    - JSON
    - XML
    - HTML
    - CSV
    - TXT
  - Without token in response body
    - GIF
    - PNG
    - JPEG
    - MP3
    - MP4

## Usage

```bash
go get -v github.com/teknogeek/ssrf-sheriff
cd $GOPATH/src/github.com/teknogeek/ssrf-sheriff
cp config/base.example.yaml config/base.yaml

# ... configure ...

go run main.go
```

### Example Requests:

**Plaintext**
```
$ curl -sSD- http://127.0.0.1:8000/foobar
HTTP/1.1 200 OK
Content-Type: text/plain
X-Secret-Token: SUP3R_S3cret_1337_K3y
Date: Mon, 14 Oct 2019 16:37:36 GMT
Content-Length: 21

SUP3R_S3cret_1337_K3y
```

**XML**
```
$ curl -sSD- http://127.0.0.1:8000/foobar.xml
HTTP/1.1 200 OK
Content-Type: application/xml
X-Secret-Token: SUP3R_S3cret_1337_K3y
Date: Mon, 14 Oct 2019 16:37:41 GMT
Content-Length: 81

<SerializableResponse><token>SUP3R_S3cret_1337_K3y</token></SerializableResponse>
```

## TODO

- Dynamically generate valid responses with the secret token visible for
  - GIF
  - PNG
  - JPEG
  - MP3
  - MP4
- Secrets in HTTP response generated/created/signed per-request, instead of returning a single secret for all requests
- TLS support

## Credit

Inspired (and requested) by [Frans Rosen](https://twitter.com/fransrosen) during his [talk at BountyCon '19 Singapore](https://speakerdeck.com/fransrosen/live-hacking-like-a-mvh-a-walkthrough-on-methodology-and-strategies-to-win-big?slide=49)


-----

Released under the [MIT License](LICENSE.txt).


