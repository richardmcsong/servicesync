FROM golang:alpine AS build

# Build env
ARG CGO_ENABLED=0
ARG GOOS=linux
ARG GOARCH=amd64
ARG VERSION

ENV GO111MODULE=on

WORKDIR $GOPATH/src/github.com/richardmcsong/servicesync

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download
RUN go mod verify

COPY cmd cmd
COPY pkg pkg

RUN CGO_ENABLED=${CGO_ENABLED} GOOS=${GOOS} GOARCH=${GOARCH} go build -o /bin/servicesync -ldflags "-X github.com/richardmcsong/servicesync/pkg/config.Version="${VERSION} ${GOPATH}/src/github.com/richardmcsong/servicesync/cmd/servicesync

FROM scratch

COPY --from=build /bin/servicesync /bin/servicesync

# TODO: Lock down the user
# RUN useradd -ms /bin/bash gial
# USER gial
# WORKDIR /home/gial

ENTRYPOINT ["/bin/servicesync"]
CMD [ "--config", "/etc/config/config.yaml" ]


