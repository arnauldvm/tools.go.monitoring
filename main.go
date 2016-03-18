package main

import (
	"flag"
	"fmt"
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

const RFC3339Millis = "2006-01-02T15:04:05.000-0700"

func main() {
	var usage bool
	flag.BoolVar(&usage, "usage", false, "prints this usage description")
	// -h, -help, --help also automatically recognised
	periodPtr := flag.Duration("interval", 500e6, "poll interval")
	durationPtr := flag.Duration("duration", 0, "monitoring duration")
	cumulPtr := flag.Bool("cumul", false, "log cumulative counters instead of delta")
	timePtr := flag.Bool("time", true, "add timestamp prefix")
	flag.Parse()
	if usage {
		flag.PrintDefaults()
		return
	}
	cout := make(chan vmstat.VmstatRecord)
	go vmstat.Poll(*periodPtr, *durationPtr, *cumulPtr, cout)
	if *timePtr {
		fmt.Print("time", vmstat.Separator)
	}
	printLine(vmstat.VmstatHeader)
	for dat := range cout {
		if *timePtr {
			fmt.Print(time.Now().Format(RFC3339Millis), vmstat.Separator)
		}
		printLine(dat)
	}
}
