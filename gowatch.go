package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/go-fsnotify/fsnotify"
)

func runGo(f string) {
	cmd := exec.Command("go", "run", f)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		return
	}
	fmt.Println(out.String())
}

func main() {
	filePtr := flag.String("f", "", "file path")
	flag.Parse()

	if len(*filePtr) == 0 {
		fmt.Fprintf(os.Stderr, "usage: %s [file path]\n", os.Args[0])
		flag.PrintDefaults()
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		runGo(*filePtr)
		for {
			select {
			case event := <-watcher.Events:
				// log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					// log.Println("modified file:", event.Name)
					// clear screen
					cmd := exec.Command("clear")
					cmd.Stdout = os.Stdout
					cmd.Run()

					runGo(*filePtr)
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(*filePtr)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}
