package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

var startChannel chan string
var stopChannel chan bool
var done chan bool
var started bool
var filePath *string

func main() {
	filePath = flag.String("f", "", "file path")
	flag.Parse()

	fmt.Println(*filePath)
	if len(*filePath) == 0 {
		fmt.Fprintf(os.Stderr, "usage: %s [file path]\n", os.Args[0])
		flag.PrintDefaults()
		return
	}

	startChannel = make(chan string)
	stopChannel = make(chan bool)
	done = make(chan bool)
	started = false

	watch()
	start()

	startChannel <- *filePath
	<-done
}

func start() {
	go func() {
		for {
			log.Println("Watching...", <-startChannel)
			time.Sleep(500 * time.Millisecond)

			if started {
				stopChannel <- true
			}
			buildAndRun(*filePath)
			started = true

			log.Printf(strings.Repeat("-", 20))
		}
	}()
	waitInterrupt()
}

func watch() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					startChannel <- event.Name
				}
			case err := <-watcher.Errors:
				if err != nil {
					log.Println("error: ", err)
				}
			}
		}
	}()

	err = watcher.Add(*filePath)

	if err != nil {
		log.Fatal(err)
	}
}

func buildPath() string {
	return "/tmp/executable.cmd"
}

func buildAndRun(f string) {
	build(f)
	run(f)
}

func build(f string) {
	log.Println("Building...")
	cmd := exec.Command("go", "build", "-o", buildPath(), f)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	go func() {
		in := bufio.NewScanner(stdout)
		for in.Scan() {
			log.Printf(in.Text()) // write each line to your log, or anything you need
		}
		if err := in.Err(); err != nil {
			log.Printf("error: %s", err)
		}
	}()

	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}
}

func run(f string) {
	log.Println("Running...")
	cmd := exec.Command(buildPath())

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	go func() {
		in := bufio.NewScanner(stdout)
		for in.Scan() {
			log.Printf(in.Text()) // write each line to your log, or anything you need
		}
		if err := in.Err(); err != nil {
			log.Printf("error: %s", err)
		}
	}()

	go func() {
		<-stopChannel
		pid := cmd.Process.Pid
		log.Println("Killing PID ", pid)
		cmd.Process.Kill()
	}()
}

func waitInterrupt() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		<-signalChan
		fmt.Println("Received an interrupt, stopping services...")
		if _, err := os.Stat(buildPath()); err == nil {
			os.Remove(buildPath())
		}
		done <- true
	}()
}
