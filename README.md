# h2go

> Note: this is heavily based on the awesome work of [ls0f](https://github.com/ls0f/cracker). Only copied here to remove some 3rd party libraries

proxy over http[s], support http,socks5 proxy.

```
+------------+            +--------------+          
| client app |  <=======> |local proxy   | <#######
+------------+            +--------------+        #
                                                  #
                                                  #
                                                  # http[s]
                                                  #
                                                  #
+-------------+            +--------------+       #
| target host |  <=======> |http[s] server|  <#####
+-------------+            +--------------+         
```

# Install

Download the latest binaries from this [release page](https://github.com/mosajjal/h2go/releases).

# Usage

## Server side (Run on your vps or other application container platform)

```
./server -addr :8080 -secret <password>
```

## Client side (Run on your local pc)

```
./client -raddr http://example.com:8080 -secret <password>
```

## https

It is strongly recommended to open the https option on the server side.

### Notice

If you have a ssl certificate, It would be easy.

```
./server -addr :443 -secret <password> -https -cert /etc/cert.pem -key /etc/key.pem
```

```
./client -raddr https://example.com -secret <password>
```

Of Course, you can create a self-signed ssl certificate by openssl.

```
sh -c "$(curl https://raw.githubusercontent.com/mosajjal/h2go/master/gen_key_cert.sh)"
```

```
./server -addr :443 -secret <password> -https -cert /etc/self-signed-cert.pem -key /etc/self-ca-key.pem
```

```
./client -raddr https://example.com -secret <password> -cert /etc/self-signed-cert.pem
```


