package main

import (
	"fmt"
	"time"

	"sic.smals.be/tools/monitoring/vmstat"
)

func check(e error) {
	if e != nil {
		panic(e) // TODO: really?
	}
}

func main() {
	period, err := time.ParseDuration("500ms")
	check(err)
	duration, err := time.ParseDuration("5s")
	check(err)
	cout := make(chan vmstat.VmstatLine)
	go vmstat.Poll(period, duration, cout)
	for dat := range cout {
		fmt.Println(dat)
	}
}
