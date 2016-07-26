# tlsproxy

`tlsproxy` will proxy an incoming TLS session to a server and pass the
responses along. This is handy for doing version bumping, for instance a RHEL5
client that needs to connect to a TLS1.1+ server can connect via this proxy
running on a RHEL6+ server.

## Usage

Before running, you need to have a valid certificate. Use OpenSSL for this.

**Interactively**

```
tlsproxy -h
```

**Daemon**

A systemd unit file has been provided which can be used to run the service as
a daemon. Edit this file to pass the arguments that you require to be passed.

## Building

```
go build .
```

## License

This project is licensed under the MIT License. See the `LICENSE` file for
details.
