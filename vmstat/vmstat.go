package vmstat // import "sic.smals.be/tools/monitoring/vmstat"

import (
	"bufio"
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

func checkPrefix(expected, actual string) error {
	if expected == actual {
		return nil
	}
	return fmt.Errorf("Not a '%s' line (found '%s')", expected, actual)
}

func parseFirstField(line, prefix string) (field uint, err error) {
	fields := strings.Fields(line)
	err = checkPrefix(prefix, fields[0])
	if err != nil {
		return
	}
	uint64field, err := strconv.ParseUint(fields[1], 10, 0)
	check(err)
	field = uint(uint64field)
	return
}

type recordPart fmt.Stringer
/* CPU */

const cpuPrefix = "cpu"

type Cpu []uint

func (cpu Cpu) String() string { // implements recordPart >> fmt.Stringer
	return fmt.Sprint([]uint(cpu))
}

func ParseCpu(line string) (cpu Cpu, err error) {
	fields := strings.Fields(line)
	newcpu := make([]uint, len(fields)-1)
	for i, f := range fields {
		if i == 0 {
			err = checkPrefix(cpuPrefix, f)
			if err != nil {
				return
			}
			continue
		}
		uint64field, err := strconv.ParseUint(f, 10, 0)
		check(err)
		newcpu[i-1] = uint(uint64field)
	}
	return Cpu(newcpu), nil
}

/* Interrupts */

const intrPrefix = "intr"

type Interrupts uint

func (intr Interrupts) String() string { // implements recordPart >> fmt.Stringer
	return fmt.Sprint(uint(intr))
}

func ParseInterrupts(line string) (intr Interrupts, err error) {
	field, err := parseFirstField(line, intrPrefix)
	if err != nil {
		return
	}
	intr = Interrupts(field)
	return
}

/* Context switches */

type ContextSwitches uint

func (ctxt ContextSwitches) String() string { // implements recordPart >> fmt.Stringer
	return fmt.Sprint(uint(ctxt))
}

/* Process/Threads */

type Procs struct {
	running uint
	blocked uint
	delta   int
}

func (procs Procs) String() string { // implements recordPart >> fmt.Stringer
	return fmt.Sprintf("%d %d %d", procs.running, procs.blocked, procs.delta)
}

/* Vmstat record */

type VmstatRecord struct {
	cpu   Cpu
	intr  Interrupts
	ctxt  ContextSwitches
	procs Procs
}

/* Polling */

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
				case strings.HasPrefix(line, intrPrefix+" "):
					intr, err := ParseInterrupts(line)
					check(err)
					fmt.Println(intr)

				default:
					//fmt.Printf("Unsupported: %s\n", line)
				}
			}
			check(scanner.Err())
		}()
	}
	close(cout)
}
