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

func makeHeader(fdl []fieldDef) Header {
	h := Header(make([]string, 1+len(fdl)))
	h[0] = "h"
	for i, d := range fdl {
		h[i+1] = d.String()
	}
	return h
}

func (h Header) WriteTo(w io.Writer) (n int64, err error) { // implements io.WriterTo
	err = writeTo(w, strings.Join(h, Separator), &n)
	return
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

func parseLineToFields(def lineDef, line string, targetSlice []uint) (err error) {
	fields := strings.Fields(line)
	err = checkPrefix(def.prefix, fields[0])
	if err != nil {
		return
	}
	var uint64field uint64
	for i := def.firstFieldIdx; i <= def.lastFieldIdx; i++ {
		uint64field, err = strconv.ParseUint(fields[i+1-def.firstFieldIdx], 10, 0)
		if err != nil {
			return
		}
		targetSlice[i] = uint(uint64field)
	}
	return
}

func writeTo(w io.Writer, v interface{}, p *int64) (err error) {
	m, err := w.Write([]byte(fmt.Sprint(v)))
	*p += int64(m)
	return
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

/* Line definition */

type lineDef struct {
	prefix                      string
	firstFieldIdx, lastFieldIdx uint
}

func (ld lineDef) String() string {
	return ld.prefix
}

/* CPU */

var cpuLineDef = lineDef{"cpu", firstCpuIdx, lastCpuIdx}

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

func totalCpuCalculator(fields []uint) (total uint) {
	for i := firstCpuIdx; i <= lastCpuIdx; i++ {
		total += fields[i]
	}
	return
}

/* Interrupts */

var intrLineDef = lineDef{"intr", intrTotalIdx, intrTotalIdx}

var intrFieldDef = fieldDef{"intr", "total", true, nil}

/* Context switches */

var ctxtLineDef = lineDef{"ctxt", ctxtTotalIdx, ctxtTotalIdx}

var ctxtFieldDef = fieldDef{"ctxt", "total", true, nil}

/* Process/Threads */

var forksLineDef = lineDef{"processes", procsForksIdx, procsForksIdx}
var runningProcsLineDef = lineDef{"procs_running", procsRunningIdx, procsRunningIdx}
var blockedProcsLineDef = lineDef{"procs_blocked", procsBlockedIdx, procsBlockedIdx}

var procsFieldsDefs = []fieldDef{
	fieldDef{"procs", "forks", true, nil},
	fieldDef{"procs", "running", false, nil},
	fieldDef{"procs", "blocked", false, nil},
}

/* Vmstat record */

var allFieldsDefs []fieldDef = append(append(append(append(make([]fieldDef, 0, fieldsCount), procsFieldsDefs...), intrFieldDef), ctxtFieldDef), cpuFieldsDefs...)

var VmstatHeader = makeHeader(allFieldsDefs)

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
	err = writeTo(w, record.isCumul, &n)
	for _, field := range record.fields {
		err = writeTo(w, Separator, &n)
		if err != nil {
			return
		}
		err = writeTo(w, field, &n)
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
			err = parseLineToFields(ld, line, record.fields)
			if err != nil {
				return
			}
		}
	}
	err = scanner.Err()
	if err != nil {
		return
	}
	for i, fd := range allFieldsDefs {
		if fd.calculator != nil {
			record.fields[i] = fd.calculator(record.fields)
		}
	}
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
