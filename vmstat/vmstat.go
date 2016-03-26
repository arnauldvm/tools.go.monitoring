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

const (
	procsForksIdx               = iota
	procsRunningIdx             = iota
	procsBlockedIdx             = iota
	intrTotalIdx                = iota
	ctxtTotalIdx                = iota
	cpuTotalIdx                 = iota
	cpuUserIdx, firstCpuIdx     = iota, iota
	cpuNiceIdx                  = iota
	cpuSystemIdx                = iota
	cpuidleIdx                  = iota
	cpuIowaitIdx                = iota
	cpuIrqIdx                   = iota
	cpuSoftIrqIdx               = iota
	cpuStealIdx                 = iota
	cpuGuestIdx                 = iota
	cpuGuestNiceIdx, lastCpuIdx = iota, iota
	fieldsCount                 = iota
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

func parseFirstField(def lineDef, line string, targetSlice []uint) (err error) {
	fields := strings.Fields(line)
	err = checkPrefix(def.prefix, fields[0])
	if err != nil {
		return
	}
	uint64field, err := strconv.ParseUint(fields[1], 10, 0)
	if err != nil {
		return
	}
	targetSlice[0] = uint(uint64field)
	return
}

type parserFunction func(def lineDef, line string, targetSlice []uint) error

func writeTo(w io.Writer, v interface{}, p int64) (int64, error) {
	m, err := w.Write([]byte(fmt.Sprint(v)))
	return p + int64(m), err
}

/* Field Definition */

type fieldCalculator func(vals []uint) uint

type fieldDef struct {
	category      string
	name          string
	isAccumulator bool
	calculator    fieldCalculator
}

func (fd fieldDef) String() string { // implements fmt.Stringer
	if fd.isAccumulator {
		return fd.category + ":" + fd.name + "/a"
	} else {
		return fd.category + ":" + fd.name + "/i"
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

/* << The amount of time, measured in units of USER_HZ
   (1/100ths of a second on most architectures, use
   sysconf(_SC_CLK_TCK) to obtain the right value), that
   the system spent in various states >> */

var cpuFieldsDefs = []fieldDef{
	fieldDef{"cpu", "total", true, totalCpuCalculator},
	fieldDef{"cpu", "user", true, nil},
	fieldDef{"cpu", "nice", true, nil},
	fieldDef{"cpu", "system", true, nil},
	fieldDef{"cpu", "idle", true, nil},
	fieldDef{"cpu", "iowait", true, nil},
	fieldDef{"cpu", "irq", true, nil},
	fieldDef{"cpu", "softirq", true, nil},
	fieldDef{"cpu", "steal", true, nil},
	fieldDef{"cpu", "guest", true, nil},
	fieldDef{"cpu", "guest_nice", true, nil},
}

func totalCpuCalculator(vals []uint) (total uint) {
	for i := 1; i < len(vals); i++ {
		total += vals[i]
	}
	return
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
	}
	for i, fd := range cpuFieldsDefs {
		if fd.calculator != nil {
			targetSlice[i] = fd.calculator(targetSlice)
		}
	}
	return
}

/* Interrupts */

var intrLineDef = lineDef{"intr", parseFirstField}

var intrFieldDef = fieldDef{"intr", "total", true, nil}

/* Context switches */

var ctxtLineDef = lineDef{"ctxt", parseFirstField}

var ctxtFieldDef = fieldDef{"ctxt", "total", true, nil}

/* Process/Threads */

var forksLineDef = lineDef{"processes", parseFirstField}
var runningProcsLineDef = lineDef{"procs_running", parseFirstField}
var blockedProcsLineDef = lineDef{"procs_blocked", parseFirstField}

var procsFieldsDefs = []fieldDef{
	fieldDef{"procs", "forks", true, nil},
	fieldDef{"procs", "running", false, nil},
	fieldDef{"procs", "blocked", false, nil},
}

/* Vmstat record */

var allFieldsDefs []fieldDef = append(append(append(append(make([]fieldDef, 0, fieldsCount), procsFieldsDefs...), intrFieldDef), ctxtFieldDef), cpuFieldsDefs...)

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
	fields  []uint
}

func (record VmstatRecord) String() string { // implements fmt.Stringer
	buf := new(bytes.Buffer)
	record.WriteTo(buf)
	return buf.String()
}
func (record VmstatRecord) WriteTo(w io.Writer) (n int64, err error) { // implements io.WriterTo
	n, err = writeTo(w, record.isCumul, n)
	for _, field := range record.fields {
		n, err = writeTo(w, Separator, n)
		if err != nil {
			return
		}
		n, err = writeTo(w, field, n)
		if err != nil {
			return
		}
	}
	return
}
func (record VmstatRecord) diff(prevRecord VmstatRecord) (diffRecord VmstatRecord) {
	diffRecord.isCumul = cumulFlag(false)
	diffRecord.fields = make([]uint, len(record.fields))
	for i, field := range record.fields {
		if allFieldsDefs[i].isAccumulator {
			diffRecord.fields[i] = field - prevRecord.fields[i]
		} else {
			diffRecord.fields[i] = field
		}
	}
	return
}

func makeEmptyVmstatRecord() VmstatRecord {
	return VmstatRecord{
		false,
		make([]uint, fieldsCount),
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
				err = ld.parser(ld, line, record.fields[firstCpuIdx-1:lastCpuIdx+1])
			case intrLineDef.prefix:
				err = ld.parser(ld, line, record.fields[intrTotalIdx:intrTotalIdx+1])
			case ctxtLineDef.prefix:
				err = ld.parser(ld, line, record.fields[ctxtTotalIdx:ctxtTotalIdx+1])
			case forksLineDef.prefix:
				err = ld.parser(ld, line, record.fields[procsForksIdx:procsForksIdx+1])
			case runningProcsLineDef.prefix:
				err = ld.parser(ld, line, record.fields[procsRunningIdx:procsRunningIdx+1])
			case blockedProcsLineDef.prefix:
				err = ld.parser(ld, line, record.fields[procsBlockedIdx:procsBlockedIdx+1])
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
