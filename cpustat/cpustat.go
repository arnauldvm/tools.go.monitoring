package cpustat // import "sic.smals.be/tools/monitoring/cpustat"

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

/* << The amount of time, measured in units of USER_HZ
   (1/100ths of a second on most architectures, use
   sysconf(_SC_CLK_TCK) to obtain the right value), that
   the system spent in various states >> */
var cpuIndices = []uint{
	cpuUserIdx,
	cpuNiceIdx,
	cpuSystemIdx,
	cpuidleIdx,
	cpuIowaitIdx,
	cpuIrqIdx,
	cpuSoftIrqIdx,
	cpuStealIdx,
	cpuGuestIdx,
	cpuGuestNiceIdx,
}

var allFieldsDefs = []fieldDef{
	fieldDef{"procs", "forks", true, nil},
	fieldDef{"procs", "running", false, nil},
	fieldDef{"procs", "blocked", false, nil},
	fieldDef{"intr", "total", true, nil},
	fieldDef{"ctxt", "total", true, nil},
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
	for _, i := range cpuIndices {
		total += fields[i]
	}
	return
}

func init() {
	addLineDef("cpu", cpuIndices...)             // CPU
	addLineDef("intr", intrTotalIdx)             // Interrupts
	addLineDef("ctxt", ctxtTotalIdx)             // Context switches
	addLineDef("processes", procsForksIdx)       // Process/Threads
	addLineDef("procs_running", procsRunningIdx) // Process/Threads
	addLineDef("procs_blocked", procsBlockedIdx) // Process/Threads
}

/* Header is a list of field names. */

type header []string

func makeHeader(fdl []fieldDef) header {
	h := header(make([]string, 1+len(fdl)))
	h[0] = "h"
	for i, d := range fdl {
		h[i+1] = d.String()
	}
	return h
}

func (h header) WriteTo(w io.Writer) (n int64, err error) { // implements io.WriterTo
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
	for i, j := range def.fieldsIdx {
		uint64field, err = strconv.ParseUint(fields[i+1], 10, 0)
		if err != nil {
			return
		}
		targetSlice[j] = uint(uint64field)
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
	prefix    string
	fieldsIdx []uint
}

var linesDefs = make(map[string]lineDef, 6)

func addLineDef(prefix string, fieldsIdx ...uint) {
	linesDefs[prefix] = lineDef{prefix, fieldsIdx}
}

/* Record */

var Header = makeHeader(allFieldsDefs)

type Record struct {
	Time           time.Time
	isCumul, isRel bool
	fields         []uint
}

func newRecord(isCumul, isRel bool) *Record {
	recordPtr := new(Record)
	recordPtr.isCumul = isCumul
	recordPtr.isRel = isRel
	recordPtr.fields = make([]uint, fieldsCount)
	return recordPtr
}

func (recordPtr *Record) String() string { // implements fmt.Stringer
	buf := new(bytes.Buffer)
	recordPtr.WriteTo(buf)
	return buf.String()
}
func (record Record) WriteTo(w io.Writer) (n int64, err error) { // implements io.WriterTo
	if record.isCumul {
		err = writeTo(w, "a", &n)
	} else {
		if record.isRel {
			err = writeTo(w, "p", &n)
		} else {
			err = writeTo(w, "d", &n)
		}
	}
	if err != nil {
		return
	}
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
func (recordPtr *Record) diff(prevRecord, diffRecord *Record) {
	diffRecord.Time = recordPtr.Time
	for i, field := range recordPtr.fields {
		if allFieldsDefs[i].isAccumulator {
			diffRecord.fields[i] = field - prevRecord.fields[i]
		} else {
			diffRecord.fields[i] = field
		}
	}
	return
}
func (diffRecordPtr *Record) rel() {
	for _, i := range cpuIndices {
		diffRecordPtr.fields[i] = diffRecordPtr.fields[i] * 100 / diffRecordPtr.fields[cpuTotalIdx]
	}
	return
}

func (recordPtr *Record) parse() (err error) {
	inFile, err := os.Open(procStat)
	if err != nil {
		return
	}
	defer inFile.Close()
	recordPtr.Time = time.Now()
	for i, _ := range recordPtr.fields {
		recordPtr.fields[i] = 0
	}
	scanner := bufio.NewScanner(inFile)
	for j := 0; scanner.Scan(); j++ {
		line := scanner.Text()
		linePrefix := strings.SplitN(line, " ", 2)[0]
		ld, ok := linesDefs[linePrefix]
		if ok {
			err = parseLineToFields(ld, line, recordPtr.fields)
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
			recordPtr.fields[i] = fd.calculator(recordPtr.fields)
		}
	}
	return
}

/* Polling */

// Poll sends a Record in the channel every period until duration.
// If cumul is false, it prints the diff of the accumulators, instead of the accumulators themselves
func Poll(period time.Duration, duration time.Duration, cumul bool, rel bool, cout chan Record) {
	startTime := time.Now()
	recordPtr := newRecord(true, false)
	oldRecordPtr := newRecord(true, false)
	diffRecordPtr := newRecord(false, rel)
	var lastTime, nextTime time.Time
	for i := 0; (0 == duration) || (time.Since(startTime) <= duration); i++ {
		if i > 0 {
			nextTime = lastTime.Add(period)
			toWait := nextTime.Sub(time.Now())
			if toWait > 0 {
				time.Sleep(toWait)
			}
		} else {
			nextTime = time.Now()
		}
		lastTime = nextTime
		err := recordPtr.parse()
		if err != nil {
			log.Println(err)
			continue
		}
		if cumul {
			cout <- *recordPtr
		} else {
			if i < 1 {
				cout <- *recordPtr
			} else {
				recordPtr.diff(oldRecordPtr, diffRecordPtr)
				if rel {
					diffRecordPtr.rel()
				}
				cout <- *diffRecordPtr
			}
			oldRecordPtr, recordPtr = recordPtr, oldRecordPtr
		}
	}
	close(cout)
}
