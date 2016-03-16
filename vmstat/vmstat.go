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

type Cpu []uint

type Interrupts uint

type ContextSwitches uint

type Procs struct {
	running uint
	blocked uint
	delta   int
}

type VmstatLine struct {
	cpu   Cpu
	intr  Interrupts
	ctxt  ContextSwitches
	procs Procs
}

// Poll sends a VmstatLine in the channel every period until duration
func Poll(period time.Duration, duration time.Duration, cout chan VmstatLine) {
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
