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
	Separator       = " "
)

/* Header is a list of field names. By convention names ending with "/a"
   are accumulators (ever growing), while names ending with "/i" are instant values. */

type Header []string

func (h Header) String() string { // implements fmt.Stringer
	buf := new(bytes.Buffer)
	h.WriteTo(buf)
	return buf.String()
}
func (h Header) WriteTo(w io.Writer) (n int64, err error) { // implements io.WriterTo
	return writeTo(w, strings.Join(h, Separator), 0)
}
func (h Header) append(k Header) Header {
	return Header(append(h, k...))
}

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
			n, err = writeTo(w, Separator, n)
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

/* << The amount of time, measured in units of USER_HZ
   (1/100ths of a second on most architectures, use
   sysconf(_SC_CLK_TCK) to obtain the right value), that
   the system spent in various states >> */

var cpuHeader = Header([]string{
	"cpu:user/a",
	"cpu:nice/a",
	"cpu:system/a",
	"cpu:idle/a",
	"cpu:iowait/a",
	"cpu:irq/a",
	"cpu:softirq/a",
	"cpu:steal/a",
	"cpu:guest/a",
	"cpu:guest_nice/a",
})

// implement recordPart

func (cpu Cpu) String() string { // implements fmt.Stringer
	buf := new(bytes.Buffer)
	cpu.WriteTo(buf)
	return buf.String()
}
func (cpu Cpu) WriteTo(w io.Writer) (n int64, err error) { // implements io.WriterTo
	for i, val := range cpu {
		if i > 0 {
			n, err = writeTo(w, Separator, n)
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
func (cpu Cpu) diff(prevCpu Cpu) (diffCpu Cpu) {
	diffCpu = Cpu(make([]uint, len(cpu)))
	for i, val := range cpu {
		diffCpu[i] = val - prevCpu[i]
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

var intrHeader = Header([]string{"intr:total/a"})

// implement recordPart

func (intr Interrupts) String() string { // implements fmt.Stringer
	return fmt.Sprint(uint(intr))
}
func (intr Interrupts) WriteTo(w io.Writer) (int64, error) { // implements io.WriterTo
	return writeTo(w, intr, 0)
}
func (intr Interrupts) diff(prevIntr Interrupts) Interrupts {
	return Interrupts(intr - prevIntr)
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

var ctxtHeader = Header([]string{"ctxt:total/a"})

// implement recordPart

func (ctxt ContextSwitches) String() string { // implements fmt.Stringer
	return fmt.Sprint(uint(ctxt))
}
func (ctxt ContextSwitches) WriteTo(w io.Writer) (int64, error) { // implements io.WriterTo
	return writeTo(w, ctxt, 0)
}
func (ctxt ContextSwitches) diff(prevCtxt ContextSwitches) ContextSwitches {
	return ContextSwitches(ctxt - prevCtxt)
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

var procsHeader = Header([]string{"proc:forks/a", "proc:running/i", "proc:blocked/i"})

// implement recordPart

func (procs Procs) String() string { // implements fmt.Stringer
	buf := new(bytes.Buffer)
	procs.WriteTo(buf)
	return buf.String()
}
func (procs Procs) WriteTo(w io.Writer) (n int64, err error) { // implements io.WriterTo
	return writeManyTo(w, 0, procs.forks, procs.running, procs.blocked)
}
func (procs Procs) diff(prevProcs Procs) (diffProcs Procs) {
	diffProcs.forks = procs.forks - prevProcs.forks
	diffProcs.running = procs.running // not an accumulator, but an instant value
	diffProcs.blocked = procs.blocked // not an accumulator, but an instant value
	return
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

var VmstatHeader = Header([]string{"a/d"}).append(procsHeader).append(intrHeader).append(ctxtHeader).append(cpuHeader)

type cumulFlag bool

func (c cumulFlag) String() string {
	if c {
		return "a"
	} else {
		return "d"
	}
}

type VmstatRecord struct {
	isCumul cumulFlag
	procs   Procs
	intr    Interrupts
	ctxt    ContextSwitches
	cpu     Cpu
}

func (record VmstatRecord) String() string { // implements fmt.Stringer
	buf := new(bytes.Buffer)
	record.WriteTo(buf)
	return buf.String()
}
func (record VmstatRecord) WriteTo(w io.Writer) (n int64, err error) { // implements io.WriterTo
	return writeManyTo(w, 0, record.isCumul, record.procs, record.intr, record.ctxt, record.cpu)
}
func (record VmstatRecord) diff(prevRecord VmstatRecord) (diffRecord VmstatRecord) {
	diffRecord.isCumul = cumulFlag(false)
	diffRecord.procs = record.procs.diff(prevRecord.procs)
	diffRecord.intr = record.intr.diff(prevRecord.intr)
	diffRecord.ctxt = record.ctxt.diff(prevRecord.ctxt)
	diffRecord.cpu = record.cpu.diff(prevRecord.cpu)
	return
}

var parsers = map[string]parserFunction{
	cpuPrefix:  ParseCpu,
	intrPrefix: ParseInterrupts,
	ctxtPrefix: ParseContextSwitches,
}

func parseVmstat() (record VmstatRecord, err error) {
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
	record.isCumul = true
	return
}

/* Polling */

// Poll sends a VmstatLine in the channel every period until duration.
// If cumul is false, it prints the diff of the accumulators, instead of the accumulators themselves
func Poll(period time.Duration, duration time.Duration, cumul bool, cout chan VmstatRecord) {
	startTime := time.Now()
	var oldRecord VmstatRecord
	for i := 0; (0 == duration) || (time.Since(startTime) <= duration); i++ {
		if i > 0 {
			time.Sleep(period)
		}
		record, err := parseVmstat()
		if err != nil {
			log.Println(err)
			continue
		}
		if cumul {
			cout <- record
		} else {
			if i < 1 {
				cout <- record
			} else {
				cout <- record.diff(oldRecord)
			}
			oldRecord = record
		}
	}
	close(cout)
}
