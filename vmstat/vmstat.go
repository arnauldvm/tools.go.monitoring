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
type parserFunction func(def lineDef, line string, targetSlice []uint) error

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

/* Field Definition */

type fieldDef struct {
	header        string
	isAccumulator bool
}

func (fd fieldDef) String() string { // implements fmt.Stringer
	if fd.isAccumulator {
		return fd.header + "/a"
	} else {
		return fd.header + "/i"
	}
}

func defToHeader(def fieldDef) Header {
	return Header([]string{def.String()})
}
func defsToHeader(defs []fieldDef) Header {
	h := Header(make([]string, len(defs)))
	for i, d := range defs {
		h[i] = d.String()
	}
	return h
}

func diffField(fieldDef fieldDef, val, prevVal uint) uint {
	if fieldDef.isAccumulator {
		return val - prevVal
	} else {
		return val
	}
}
func diffFields(fieldsDefs []fieldDef, vals, prevVals []uint) []uint {
	diffVals := make([]uint, len(vals))
	for i, val := range vals {
		diffVals[i] = diffField(fieldsDefs[i], val, prevVals[i])
	}
	return diffVals
}

/* Line definition */

type lineDef struct {
	prefix string
	parser parserFunction
}

func (ld lineDef) String() string {
	return ld.prefix
}

/* CPU */

var cpuLineDef = lineDef{"cpu", ParseCpu}

type Cpu []uint

/* << The amount of time, measured in units of USER_HZ
   (1/100ths of a second on most architectures, use
   sysconf(_SC_CLK_TCK) to obtain the right value), that
   the system spent in various states >> */

var cpuFieldsDefs = []fieldDef{
	fieldDef{"cpu:total", true},
	fieldDef{"cpu:user", true},
	fieldDef{"cpu:nice", true},
	fieldDef{"cpu:system", true},
	fieldDef{"cpu:idle", true},
	fieldDef{"cpu:iowait", true},
	fieldDef{"cpu:irq", true},
	fieldDef{"cpu:softirq", true},
	fieldDef{"cpu:steal", true},
	fieldDef{"cpu:guest", true},
	fieldDef{"cpu:guest_nice", true},
}

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
func (cpu Cpu) diff(prevCpu Cpu) Cpu {
	return Cpu(diffFields(cpuFieldsDefs, cpu, prevCpu))
}

func makeEmptyCpu() Cpu {
	return Cpu(make([]uint, len(cpuFieldsDefs)))
}
func ParseCpu(def lineDef, line string, targetSlice []uint) (err error) {
	fields := strings.Fields(line)
	var val uint
	for i, f := range fields {
		if i == 0 {
			err = checkPrefix(def.prefix, f)
			if err != nil {
				return
			}
			continue
		}
		var uint64field uint64
		uint64field, err = strconv.ParseUint(f, 10, 0)
		if err != nil {
			return
		}
		val = uint(uint64field)
		targetSlice[i-1+1] = val
		targetSlice[0] += val // cpu:total field
	}
	return
}

/* Interrupts */

var intrLineDef = lineDef{"intr", ParseInterrupts}

type Interrupts uint

var intrFieldDef = fieldDef{"intr:total", true}

// implement recordPart

func (intr Interrupts) String() string { // implements fmt.Stringer
	return fmt.Sprint(uint(intr))
}
func (intr Interrupts) WriteTo(w io.Writer) (int64, error) { // implements io.WriterTo
	return writeTo(w, intr, 0)
}
func (intr Interrupts) diff(prevIntr Interrupts) Interrupts {
	return Interrupts(diffField(intrFieldDef, uint(intr), uint(prevIntr)))
}

func makeEmptyInterrupts() Interrupts {
	return *new(Interrupts)
}
func ParseInterrupts(def lineDef, line string, targetSlice []uint) (err error) {
	targetSlice[0], err = parseFirstField(line, def.prefix)
	return
}

/* Context switches */

var ctxtLineDef = lineDef{"ctxt", ParseContextSwitches}

type ContextSwitches uint

var ctxtFieldDef = fieldDef{"ctxt:total", true}

// implement recordPart

func (ctxt ContextSwitches) String() string { // implements fmt.Stringer
	return fmt.Sprint(uint(ctxt))
}
func (ctxt ContextSwitches) WriteTo(w io.Writer) (int64, error) { // implements io.WriterTo
	return writeTo(w, ctxt, 0)
}
func (ctxt ContextSwitches) diff(prevCtxt ContextSwitches) ContextSwitches {
	return ContextSwitches(diffField(ctxtFieldDef, uint(ctxt), uint(prevCtxt)))
}

func makeEmptyContextSwitches() ContextSwitches {
	return *new(ContextSwitches)
}
func ParseContextSwitches(def lineDef, line string, targetSlice []uint) (err error) {
	targetSlice[0], err = parseFirstField(line, def.prefix)
	return
}

/* Process/Threads */

var forksLineDef = lineDef{"processes", ParseProcsForks}
var runningProcsLineDef = lineDef{"procs_running", ParseRunningProcs}
var blockedProcsLineDef = lineDef{"procs_blocked", ParseBlockedProcs}

type Procs []uint

var procsFieldsDefs = []fieldDef{
	fieldDef{"procs:forks", true},
	fieldDef{"procs:running", false},
	fieldDef{"procs:blocked", false},
}

// implement recordPart

func (procs Procs) String() string { // implements fmt.Stringer
	buf := new(bytes.Buffer)
	procs.WriteTo(buf)
	return buf.String()
}
func (procs Procs) WriteTo(w io.Writer) (n int64, err error) { // implements io.WriterTo
	return writeManyTo(w, 0, procs[0], procs[1], procs[2])
}
func (procs Procs) diff(prevProcs Procs) (diffProcs Procs) {
	return Procs(diffFields(procsFieldsDefs, procs, prevProcs))
}

func makeEmptyProcs() Procs {
	return Procs(make([]uint, len(procsFieldsDefs)))
}
func ParseProcsForks(def lineDef, line string, targetSlice []uint) (err error) {
	targetSlice[0], err = parseFirstField(line, def.prefix)
	return
}

func ParseRunningProcs(def lineDef, line string, targetSlice []uint) (err error) {
	targetSlice[1], err = parseFirstField(line, def.prefix)
	return
}

func ParseBlockedProcs(def lineDef, line string, targetSlice []uint) (err error) {
	targetSlice[2], err = parseFirstField(line, def.prefix)
	return
}

/* Vmstat record */

var VmstatHeader = Header([]string{"a/d"}).
	append(defsToHeader(procsFieldsDefs)).
	append(defToHeader(intrFieldDef)).
	append(defToHeader(ctxtFieldDef)).
	append(defsToHeader(cpuFieldsDefs))

var linesDefs = map[string]lineDef{
	cpuLineDef.prefix:          cpuLineDef,
	intrLineDef.prefix:         intrLineDef,
	ctxtLineDef.prefix:         ctxtLineDef,
	forksLineDef.prefix:        forksLineDef,
	runningProcsLineDef.prefix: runningProcsLineDef,
	blockedProcsLineDef.prefix: blockedProcsLineDef,
}

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

func makeEmptyVmstatRecord() VmstatRecord {
	return VmstatRecord{
		false,
		makeEmptyProcs(),
		makeEmptyInterrupts(),
		makeEmptyContextSwitches(),
		makeEmptyCpu(),
	}
}
func parseVmstat() (record VmstatRecord, err error) {
	inFile, err := os.Open(procStat)
	if err != nil {
		return
	}
	defer inFile.Close()
	record = makeEmptyVmstatRecord()
	scanner := bufio.NewScanner(inFile)
	for j := 0; scanner.Scan(); j++ {
		line := scanner.Text()
		linePrefix := strings.SplitN(line, " ", 2)[0]
		ld, ok := linesDefs[linePrefix]
		if ok {
			switch ld.prefix {
			case cpuLineDef.prefix:
				err = ld.parser(ld, line, record.cpu)
			case intrLineDef.prefix:
				tmp := make([]uint, 1)
				err = ld.parser(ld, line, tmp)
				record.intr = Interrupts(tmp[0])
			case ctxtLineDef.prefix:
				tmp := make([]uint, 1)
				err = ld.parser(ld, line, tmp)
				record.ctxt = ContextSwitches(tmp[0])
			case forksLineDef.prefix, runningProcsLineDef.prefix, blockedProcsLineDef.prefix:
				err = ld.parser(ld, line, record.procs)
			default:
				err = fmt.Errorf("Unexpected line def, should not be here")
			}
			if err != nil {
				return
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
