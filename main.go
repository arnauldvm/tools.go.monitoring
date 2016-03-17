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
	var periodStr, durationStr string
	if len(os.Args) > 1 {
		periodStr = os.Args[1]
	} else {
		periodStr = "500ms"
	}
	if len(os.Args) > 2 {
		durationStr = os.Args[2]
	} else {
		durationStr = "5s"
	}
	period, err := time.ParseDuration(periodStr)
	check(err)
	duration, err := time.ParseDuration(durationStr)
	check(err)
	cout := make(chan vmstat.VmstatRecord)
	go vmstat.Poll(period, duration, cout)
	printLine(vmstat.VmstatHeader)
	for dat := range cout {
		printLine(dat)
	}
}
