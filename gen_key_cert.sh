#!/usr/bin/env bash

# openssl 1.1.1
read -p "Enter your ip or domain:" domain
openssl req -subj "/CN=${domain}" \
        -addext "subjectAltName=DNS:${domain},DNS:*.${domain},IP:${domain}" \
	-x509 -sha256 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 1024 -nodes
