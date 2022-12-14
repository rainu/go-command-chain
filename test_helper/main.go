package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"time"
)

// This is a helper application which can made some stdout and stderr outputs.
// It will be used in the chain_test.go and is not part of the library. It exists
// only for test purposes.

func main() {
	toErr := flag.String("e", "", "write this value to stderr")
	toOut := flag.String("o", "", "write this value to stdout")
	tickOut := flag.Duration("to", 0, "write one line at out per interval (see -ti) for X time")
	tickErr := flag.Duration("te", 0, "write one line at err per interval (see -ti) for X time")
	tickInt := flag.Duration("ti", 1*time.Second, "in which interval should the lines be written")
	printEnv := flag.Bool("pe", false, "print environment variables to stdout")
	printWorkDir := flag.Bool("pwd", false, "print the current working directory to stdout")
	exitCode := flag.Int("x", 0, "the exit code")

	flag.Parse()

	if toErr != nil && *toErr != "" {
		println(*toErr)
	}
	if toOut != nil && *toOut != "" {
		fmt.Println(*toOut)
	}
	if *printEnv {
		env := os.Environ()
		for _, curEnv := range env {
			fmt.Println(curEnv)
		}
	}
	if *printWorkDir {
		wd, _ := os.Getwd()
		fmt.Println(wd)
	}

	wg := sync.WaitGroup{}

	handleOut(tickOut, tickInt, &wg)
	handleErr(tickErr, tickInt, &wg)

	wg.Wait()

	if exitCode != nil {
		os.Exit(*exitCode)
	}
}

func handleOut(tickOut *time.Duration, tickInt *time.Duration, wg *sync.WaitGroup) {
	if tickOut != nil && *tickOut != 0 {
		timer := time.NewTimer(*tickOut)
		ticker := time.NewTicker(*tickInt)

		wg.Add(1)
		go func() {
			defer wg.Done()

		outLoop:
			for {
				select {
				case <-ticker.C:
					fmt.Println("OUT")
				case <-timer.C:
					break outLoop
				}
			}
		}()
	}
}

func handleErr(tickErr *time.Duration, tickInt *time.Duration, wg *sync.WaitGroup) {
	if tickErr != nil && *tickErr != 0 {
		timer := time.NewTimer(*tickErr)
		ticker := time.NewTicker(*tickInt)

		wg.Add(1)
		go func() {
			defer wg.Done()

		errLoop:
			for {
				select {
				case <-ticker.C:
					println("ERR")
				case <-timer.C:
					break errLoop
				}
			}
		}()
	}
}
