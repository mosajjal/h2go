FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.23.4-alpine3.21

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

LABEL maintainer="Ali Mosajjal <hi@n0p.me>"
RUN apk add --no-cache git
RUN mkdir /app
ADD . /app/
WORKDIR /app/cmd/h2go
ENV CGO_ENABLED=0
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOFLAGS=-buildvcs=false go build -ldflags "-s -w -X main.version=$(git describe --tags) -X main.commit=$(git rev-parse HEAD)" -o h2go .
CMD ["/app/cmd/h2go/h2go"]

FROM scratch
COPY --from=0 /app/cmd/h2go/h2go /h2go
ENTRYPOINT ["/h2go"]