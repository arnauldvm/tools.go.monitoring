package main

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

func check(e error) {
	if e != nil {
		panic(e) // TODO: really?
	}
}

const (
	procStat = "/proc/stat"
)

// Vmstat returns an array of 10 uint every period until duration
func Vmstat(period time.Duration, duration time.Duration, cout chan [10]uint) {
	startTime := time.Now()
	for i := 0; time.Since(startTime) <= duration; i++ {
		if i > 0 {
			time.Sleep(period)
		}
		func() {
			inFile, err := os.Open(procStat)
			check(err)
			defer inFile.Close()
			scanner := bufio.NewScanner(inFile)
			for j := 0; scanner.Scan(); j++ {
				if j < 1 {
					fmt.Println(scanner.Text())
				}
			}
			check(scanner.Err())
		}()
	}
	close(cout)
}

func main() {
	period, err := time.ParseDuration("500ms")
	check(err)
	duration, err := time.ParseDuration("5s")
	check(err)
	cout := make(chan [10]uint)
	go Vmstat(period, duration, cout)
	for dat := range cout {
		fmt.Println(dat)
	}
}
