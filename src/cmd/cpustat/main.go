package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"internal/cpustat"
)

func printLine(wt io.WriterTo) {
	wt.WriteTo(os.Stdout)
	os.Stdout.Write([]byte{'\n'})
}

const RFC3339Millis = "2006-01-02T15:04:05.000-0700"

func main() {
	var usage bool
	flag.BoolVar(&usage, "usage", false, "prints this usage description")
	// -h, -help, --help also automatically recognised
	periodPtr := flag.Duration("interval", 1e9, "poll interval")                           // defaults to 1e9ns = 1s
	durationPtr := flag.Duration("duration", 0, "monitoring duration (unlimited if zero)") // defaults to unlimited
	cumulPtr := flag.Bool("cumul", false, "log cumulative counters instead of delta")
	relPtr := flag.Bool("rel", true, "relative cpu usage (in pct), ignored if cumul is true")
	timePtr := flag.Bool("time", true, "add timestamp prefix")
	flag.Parse()
	if usage {
		flag.PrintDefaults()
		return
	}
	cout := make(chan cpustat.Record)
	go cpustat.Poll(*periodPtr, *durationPtr, *cumulPtr, *relPtr, cout)
	if *timePtr {
		fmt.Print("time", cpustat.Separator)
	}
	printLine(cpustat.Header)
	for dat := range cout {
		if *timePtr {
			fmt.Print(dat.Time.Format(RFC3339Millis), cpustat.Separator)
		}
		printLine(dat)
	}
}
