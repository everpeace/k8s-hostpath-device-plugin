FROM golang:1.17 as build
ARG CGO_ENABLED=0
ARG GOOS=linux
ARG GOARCH=amd64
WORKDIR /go/src/app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go install -ldflags="-s -w"

FROM gcr.io/distroless/base-debian10
COPY --from=build /go/bin/k8s-hostpath-device-plugin /bin/k8s-hostpath-device-plugin
ENTRYPOINT ["/bin/k8s-hostpath-device-plugin"]
