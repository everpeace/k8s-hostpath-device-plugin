package main

import (
	"math/rand"
	"time"

	"github.com/everpeace/k8s-hostpath-device-plugin/cmd"
)

func main() {
	rand.NewSource(time.Now().UnixNano())
	cmd.Execute()
}
