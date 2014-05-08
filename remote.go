package main

import (
	"log"
	"net"
	"os"

	"github.com/dotcloud/docker/engine"
	"github.com/dotcloud/docker/pkg/beam"
)

func main() {
	eng := engine.New()
	eng.Logging = true

	c, err := net.Dial("unix", "beam.sock")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	f, err := c.(*net.UnixConn).File()
	if err != nil {
		log.Fatal(err)
	}

	child, err := beam.FileConn(f)
	if err != nil {
		log.Fatal(err)
	}
	defer child.Close()

	sender := engine.NewSender(child)
	sender.Install(eng)

	job := eng.Job(os.Args[1], os.Args[2:]...)
	job.Stdout.Add(os.Stdout)
	job.Stderr.Add(os.Stderr)
	job.Stdin.Add(os.Stdin)

	if err := job.Run(); err != nil {
		log.Fatal(err)
	}
}
