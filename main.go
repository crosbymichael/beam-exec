package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/dotcloud/docker/engine"
	"github.com/dotcloud/docker/pkg/beam"
)

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func main() {
	var (
		eng = engine.New()
	)
	eng.Logging = true

	eng.Register("exec", func(job *engine.Job) engine.Status {
		cmd := exec.Command(job.Args[0], job.Args[1:]...)

		inpipe, err := cmd.StdinPipe()
		if err != nil {
			return job.Error(err)
		}
		errpipe, err := cmd.StderrPipe()
		if err != nil {
			return job.Error(err)
		}
		outpipe, err := cmd.StdoutPipe()
		if err != nil {
			return job.Error(err)
		}

		go func() {
			defer inpipe.Close()
			io.Copy(inpipe, job.Stdin)
		}()

		go io.Copy(job.Stdout, outpipe)
		go io.Copy(job.Stderr, errpipe)

		if err := cmd.Run(); err != nil {
			return job.Error(err)
		}
		return engine.StatusOK
	})

	l, err := net.Listen("unix", "beam.sock")
	if err != nil {
		fatal(err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		for _ = range sig {
			l.Close()
			os.Remove("beam.sock")
			os.Exit(0)
		}
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}

		go func() {
			defer conn.Close()

			u := conn.(*net.UnixConn)
			beamConn := &beam.UnixConn{UnixConn: u}

			r := engine.NewReceiver(beamConn)
			r.Engine = eng

			if err := r.Run(); err != nil {
				fatal(err)
			}
		}()
	}
}
