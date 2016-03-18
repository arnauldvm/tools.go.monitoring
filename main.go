package main

import (
	"flag"
	"io"
	"os"

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
	var usage bool
	flag.BoolVar(&usage, "usage", false, "prints this usage description")
	// -h, -help, --help also automatically recognised
	periodPtr := flag.Duration("interval", 500e6, "poll interval")
	durationPtr := flag.Duration("duration", 0, "monitoring duration")
	flag.Parse()
	if usage {
		flag.PrintDefaults()
		return
	}
	cout := make(chan vmstat.VmstatRecord)
	go vmstat.Poll(*periodPtr, *durationPtr, cout)
	printLine(vmstat.VmstatHeader)
	for dat := range cout {
		printLine(dat)
	}
}
