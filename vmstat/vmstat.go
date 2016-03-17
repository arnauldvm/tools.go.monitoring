package vmstat // import "sic.smals.be/tools/monitoring/vmstat"

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
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

const cpuPrefix = "cpu"

type Cpu []uint

func ParseCpu(line string) (cpu Cpu, err error) {
	fields := strings.Fields(line)
	newcpu := make([]uint, len(fields)-1)
	for i, f := range fields {
		if i == 0 {
			if f != cpuPrefix {
				err = errors.New("Not a '" + cpuPrefix + "' line")
				return
			}
			continue
		}
		uintfield, err := strconv.ParseUint(f, 10, 0)
		check(err)
		newcpu[i-1] = uint(uintfield)
	}
	return newcpu, nil
}

type Interrupts uint

type ContextSwitches uint

type Procs struct {
	running uint
	blocked uint
	delta   int
}

type VmstatRecord struct {
	cpu   Cpu
	intr  Interrupts
	ctxt  ContextSwitches
	procs Procs
}

// Poll sends a VmstatLine in the channel every period until duration
func Poll(period time.Duration, duration time.Duration, cout chan VmstatRecord) {
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
				line := scanner.Text()
				switch {
				case strings.HasPrefix(line, cpuPrefix+" "):
					cpu, err := ParseCpu(line)
					check(err)
					fmt.Println(cpu)

				default:
					//fmt.Printf("Unsupported: %s\n", line)
				}
			}
			check(scanner.Err())
		}()
	}
	close(cout)
}
