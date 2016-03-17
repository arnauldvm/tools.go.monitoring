package main

import (
	"io"
	"os"
	"time"

	"sic.smals.be/tools/monitoring/vmstat"
)

func check(e error) {
	if e != nil {
		panic(e) // TODO: really?
	}
}

func printLine(wt io.WriterTo) {
	wt.WriteTo(os.Stdout)
	os.Stdout.Write([]byte{'\n'})
}

func main() {
	period, err := time.ParseDuration("500ms")
	check(err)
	duration, err := time.ParseDuration("5s")
	check(err)
	cout := make(chan vmstat.VmstatRecord)
	go vmstat.Poll(period, duration, cout)
	printLine(vmstat.VmstatHeader)
	for dat := range cout {
		printLine(dat)
	}
}
