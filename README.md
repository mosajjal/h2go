# h2go

> Note: this is heavily based on the awesome work of [ls0f](https://github.com/ls0f/cracker). Only copied here to remove some 3rd party libraries

proxy over http[s], support http,socks5 proxy.

```
+-----1------+             +-------2-------+          
| client app |  <=======>  |  local proxy  |  <#####
+------------+             +---------------+       #
                                                   #
                                                   #
                                                   # http[s]
                                                   #
                                                   #
+------4------+             +-------3------+       #
| target host |  <=======>  |http[s] server|  <#####
+-------------+             +--------------+         
```

# Install

Download the latest binaries from this [release page](https://github.com/mosajjal/h2go/releases).

# Usage

## Server side (Run on your vps or other application container platform)

```
./h2go server --addr :8080 --secret <password>
```

## Client side (Run on your local pc)

```
./h2go client --raddr http://example.com:8080 --secret <password>
```

## https

It is strongly recommended to open the https option on the server side.

### Notice

If you have a ssl certificate, It would be easy.

```
./h2go server --addr :443 --secret <password> --https --cert /etc/cert.pem --key /etc/key.pem
```

```
./h2go client --raddr https://example.com --secret <password>
```

you can also generate self-signed ssl certificate using the gencert command.

```
./h2go gencert --domain example.com # you can also use an IP instead of a domain, or provide multiple --domain flags
```

```
./h2go server --addr :443 --secret <password> --https --cert /etc/self-signed-cert.pem --key /etc/self-ca-key.pem
```

```
./h2go client --raddr https://example.com --secret <password> --cert /etc/self-signed-cert.pem
```


