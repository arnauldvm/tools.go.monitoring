package vmstat // import "sic.smals.be/tools/monitoring/vmstat"

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"time"
)

func check(e error) {
	if e != nil {
		panic(e) // TODO: really?
	}
}

const (
	defaultProcStat = "/proc/stat"
)

var procStat string = defaultProcStat

func init() {
	fsRoot := os.Getenv("FS_ROOT")
	if fsRoot != "" {
		procStat = path.Join(fsRoot, defaultProcStat)
	}
}

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
