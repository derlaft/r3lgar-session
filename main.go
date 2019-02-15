// say hi or die! (=

package main

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var dones = []chan bool{}

func background(process string, finish chan error) *exec.Cmd {
	cmd := exec.Command("/bin/sh", process)
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		finish <- cmd.Wait()
	}()

	return cmd
}

// http://stackoverflow.com/questions/11886531/terminating-a-process-started-with-os-exec-in-golang
func start(process string, finish chan bool) {

	var (
		done     = make(chan error, 1)
		cmd      *exec.Cmd
		restarts = 0
	)

	for {
		cmd = background(process, done)
		select {
		case <-finish:
			if err := cmd.Process.Kill(); err != nil {
				log.Println("failed to kill: ", err)
			}
			log.Printf("process %v killed as shutting down", process)
			return
		case err := <-done:
			if err != nil {
				log.Printf("process %v done with error = %v; restarting", process, err)
			}
		}

		restarts++
		if restarts == 20 {
			time.Sleep(time.Second * 5)
			restarts = 0
		}
	}
}

func visit(path string, f os.FileInfo, err error) error {
	if !f.IsDir() && strings.HasSuffix(path, ".sh") {
		done := make(chan bool, 1)
		dones = append(dones, done)
		go start(path, done)
	}
	return nil
}

func main() {
	if len(os.Args) < 2 {
		log.Printf("Usage: %v directory_with_scripts", os.Args[0])
		return
	}
	err := filepath.Walk(os.Args[1], visit)
	if err != nil {
		log.Println(err)
	}

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		// terminate
		for _, done := range dones {
			done <- true
		}
		os.Exit(0)
	}()

	<-make(chan bool)
}
