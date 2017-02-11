package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"internal/linescount"
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
	substringPtr := flag.String("substring", "", "keep only lines containing this substring")
	invertPtr := flag.Bool("invert", false, "invert meaning of -substring (keep only lines *not* containing the substring)")
	periodPtr := flag.Duration("interval", 1e9, "poll interval")                           // defaults to 1e9ns = 1s
	durationPtr := flag.Duration("duration", 0, "monitoring duration (unlimited if zero)") // defaults to unlimited
	cumulPtr := flag.Bool("cumul", false, "log cumulative counters instead of delta")
	timePtr := flag.Bool("time", true, "add timestamp prefix")
	flag.Parse()
	if usage {
		flag.PrintDefaults()
		return
	}
	cout := make(chan linescount.Record)
	go linescount.Poll(*substringPtr, *invertPtr, *periodPtr, *durationPtr, *cumulPtr, cout)
	if *timePtr {
		fmt.Print("time", linescount.Separator)
	}
	printLine(linescount.Header)
	for dat := range cout {
		if *timePtr {
			fmt.Print(dat.Time.Format(RFC3339Millis), linescount.Separator)
		}
		printLine(dat)
	}
}
