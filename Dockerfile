FROM golang:1.20 as build
ARG CGO_ENABLED=0
ARG GOOS=linux
ARG GOARCH=amd64
WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
COPY . /workspace
RUN make build-only

FROM gcr.io/distroless/base-debian10 as runtime
COPY --from=build /workspace/dist/k8s-hostpath-device-plugin /bin/k8s-hostpath-device-plugin
ENTRYPOINT ["/bin/k8s-hostpath-device-plugin"]
