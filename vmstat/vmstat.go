package vmstat // import "sic.smals.be/tools/monitoring/vmstat"

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	defaultProcStat = "/proc/stat"
	separator       = " "
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
	if err != nil {
		return
	}
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

type recordPart interface {
	fmt.Stringer
	io.WriterTo
}
type parserFunction func(string) (recordPart, error)

func writeTo(w io.Writer, v interface{}, p int64) (int64, error) {
	m, err := w.Write([]byte(fmt.Sprint(v)))
	return p + int64(m), err
}
func writeManyTo(w io.Writer, p int64, vals ...interface{}) (n int64, err error) {
	n = p
	for i, val := range vals {
		if i > 0 {
			n, err = writeTo(w, separator, n)
			if err != nil {
				return
			}
		}
		n, err = writeTo(w, val, n)
		if err != nil {
			return
		}
	}
	return
}

/* CPU */

const cpuPrefix = "cpu"

type Cpu []uint

// implement recordPart

func (cpu Cpu) String() string { // implements fmt.Stringer
	buf := new(bytes.Buffer)
	cpu.WriteTo(buf)
	return buf.String()
}
func (cpu Cpu) WriteTo(w io.Writer) (n int64, err error) { // implements io.WriterTo
	for i, val := range cpu {
		if i > 0 {
			n, err = writeTo(w, separator, n)
			if err != nil {
				return
			}
		}
		n, err = writeTo(w, val, n)
		if err != nil {
			return
		}
	}
	return
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
		if err != nil {
			return nil, err
		}
		newcpu[i-1] = uint(uint64field)
	}
	return Cpu(newcpu), nil
}

/* Interrupts */

const intrPrefix = "intr"

type Interrupts uint

// implement recordPart

func (intr Interrupts) String() string { // implements fmt.Stringer
	return fmt.Sprint(uint(intr))
}
func (intr Interrupts) WriteTo(w io.Writer) (int64, error) { // implements io.WriterTo
	return writeTo(w, intr, 0)
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

// implement recordPart

func (ctxt ContextSwitches) String() string { // implements fmt.Stringer
	return fmt.Sprint(uint(ctxt))
}
func (ctxt ContextSwitches) WriteTo(w io.Writer) (int64, error) { // implements io.WriterTo
	return writeTo(w, ctxt, 0)
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

// implement recordPart

func (procs Procs) String() string { // implements fmt.Stringer
	buf := new(bytes.Buffer)
	procs.WriteTo(buf)
	return buf.String()
}
func (procs Procs) WriteTo(w io.Writer) (n int64, err error) { // implements io.WriterTo
	return writeManyTo(w, 0, procs.forks, procs.running, procs.blocked)
}

func (procs *Procs) parse(line string) error {
	fields := strings.Fields(line)
	uint64field, err := strconv.ParseUint(fields[1], 10, 0)
	if err != nil {
		return err
	}
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
	procs Procs
	intr  Interrupts
	ctxt  ContextSwitches
	cpu   Cpu
}

func (record VmstatRecord) String() string { // implements fmt.Stringer
	buf := new(bytes.Buffer)
	record.WriteTo(buf)
	return buf.String()
}
func (record VmstatRecord) WriteTo(w io.Writer) (n int64, err error) { // implements io.WriterTo
	return writeManyTo(w, 0, record.procs, record.intr, record.ctxt, record.cpu)
}

func parseVmstat() (record VmstatRecord, err error) {
	record = *new(VmstatRecord)
	inFile, err := os.Open(procStat)
	if err != nil {
		return
	}
	defer inFile.Close()
	scanner := bufio.NewScanner(inFile)
	for j := 0; scanner.Scan(); j++ {
		line := scanner.Text()
		linePrefix := strings.SplitN(line, " ", 2)[0]
		parserFn, ok := parsers[linePrefix]
		if ok {
			var part recordPart
			part, err = parserFn(line)
			if err != nil {
				return
			}
			switch val := part.(type) {
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
				if err != nil {
					return
				}
			} else {
				// ignore other records
			}
		}
	}
	err = scanner.Err()
	return
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
		record, err := parseVmstat()
		if err != nil {
			log.Println(err)
			continue
		}
		cout <- record
	}
	close(cout)
}
