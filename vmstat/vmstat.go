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

func stringInSlice(str string, slice []string) bool {
	for _, val := range slice {
		if str == val {
			return true
		}
	}
	return false
}

type recordPart fmt.Stringer
type parserFunction func(string) (recordPart, error)

/* CPU */

const cpuPrefix = "cpu"

type Cpu []uint

func (cpu Cpu) String() string { // implements recordPart >> fmt.Stringer
	return fmt.Sprint([]uint(cpu))
}

func ParseCpu(line string) (cpu recordPart, err error) {
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

func ParseInterrupts(line string) (intr recordPart, err error) {
	field, err := parseFirstField(line, intrPrefix)
	if err != nil {
		return
	}
	intr = Interrupts(field)
	return
}

/* Context switches */

const ctxtPrefix = "ctxt"

type ContextSwitches uint

func (ctxt ContextSwitches) String() string { // implements recordPart >> fmt.Stringer
	return fmt.Sprint(uint(ctxt))
}

func ParseContextSwitches(line string) (ctxt recordPart, err error) {
	field, err := parseFirstField(line, ctxtPrefix)
	if err != nil {
		return
	}
	ctxt = ContextSwitches(field)
	return
}

/* Process/Threads */

const (
	forksPrefix       = "processes"
	runningProcPrefix = "procs_running"
	blockedProcPrefix = "procs_blocked"
)

var procPrefixes = []string{forksPrefix, runningProcPrefix, blockedProcPrefix}

type Procs struct {
	forks   int
	running uint
	blocked uint
}

func (procs Procs) String() string { // implements recordPart >> fmt.Stringer
	return fmt.Sprintf("%d %d %d", procs.forks, procs.running, procs.blocked)
}

func (procs *Procs) parse(line string) error {
	fields := strings.Fields(line)
	uint64field, err := strconv.ParseUint(fields[1], 10, 0)
	check(err)
	switch fields[0] {
	case forksPrefix:
		procs.forks = int(uint64field)
	case runningProcPrefix:
		procs.running = uint(uint64field)
	case blockedProcPrefix:
		procs.blocked = uint(uint64field)
	default:
		return fmt.Errorf("Not a '%s' line (found '%s')", strings.Join(procPrefixes, "' or '"), fields[0])
	}
	return nil
}

/* Vmstat record */

type VmstatRecord struct {
	cpu   Cpu
	intr  Interrupts
	ctxt  ContextSwitches
	procs Procs
}

func (record VmstatRecord) String() string {
	return fmt.Sprintf("%s %s %s %s", record.procs, record.intr, record.ctxt, record.cpu)
}

func parseVmstat() {
	inFile, err := os.Open(procStat)
	check(err)
	defer inFile.Close()
	scanner := bufio.NewScanner(inFile)
	record := new(VmstatRecord)
	for j := 0; scanner.Scan(); j++ {
		line := scanner.Text()
		linePrefix := strings.SplitN(line, " ", 2)[0]
		parserFn, ok := parsers[linePrefix]
		if ok {
			recordPart, err := parserFn(line)
			check(err)
			switch val := recordPart.(type) {
			case Cpu:
				record.cpu = val
			case Interrupts:
				record.intr = val
			case ContextSwitches:
				record.ctxt = val
			}
		} else {
			if stringInSlice(linePrefix, procPrefixes) {
				err = record.procs.parse(line)
				check(err)
			} else {
				//fmt.Printf("Unsupported: %s\n", line)
			}
		}
	}
	fmt.Println(record)
	check(scanner.Err())
}

/* Polling */

var parsers = map[string]parserFunction{
	cpuPrefix:  ParseCpu,
	intrPrefix: ParseInterrupts,
	ctxtPrefix: ParseContextSwitches,
}

// Poll sends a VmstatLine in the channel every period until duration
func Poll(period time.Duration, duration time.Duration, cout chan VmstatRecord) {
	startTime := time.Now()
	for i := 0; time.Since(startTime) <= duration; i++ {
		if i > 0 {
			time.Sleep(period)
		}
		parseVmstat()
	}
	close(cout)
}
