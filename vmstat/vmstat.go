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

/* << The amount of time, measured in units of USER_HZ
   (1/100ths of a second on most architectures, use
   sysconf(_SC_CLK_TCK) to obtain the right value), that
   the system spent in various states >> */

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

/* Line definition */

type lineDef struct {
	prefix    string
	fieldsIdx []uint
}

var linesDefs = make(map[string]lineDef, 6)

func addLineDef(prefix string, fieldsIdx ...uint) {
	linesDefs[prefix] = lineDef{prefix, fieldsIdx}
}

func init() {
	addLineDef("cpu", cpuIndices...)             // CPU
	addLineDef("intr", intrTotalIdx)             // Interrupts
	addLineDef("ctxt", ctxtTotalIdx)             // Context switches
	addLineDef("processes", procsForksIdx)       // Process/Threads
	addLineDef("procs_running", procsRunningIdx) // Process/Threads
	addLineDef("procs_blocked", procsBlockedIdx) // Process/Threads
}

/* Vmstat record */

var VmstatHeader = makeHeader(allFieldsDefs)

type VmstatRecord struct {
	isCumul bool
	fields  []uint
}

func newVmstatRecord(isCumul bool) *VmstatRecord {
	recordPtr := new(VmstatRecord)
	recordPtr.isCumul = isCumul
	recordPtr.fields = make([]uint, fieldsCount)
	return recordPtr
}

func (record VmstatRecord) String() string { // implements fmt.Stringer
	buf := new(bytes.Buffer)
	record.WriteTo(buf)
	return buf.String()
}
func (record VmstatRecord) WriteTo(w io.Writer) (n int64, err error) { // implements io.WriterTo
	if record.isCumul {
		err = writeTo(w, "a", &n)
	} else {
		err = writeTo(w, "d", &n)
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
func (record VmstatRecord) diff(prevRecord VmstatRecord) (diffRecord VmstatRecord) {
	diffRecord.isCumul = false
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

func parseVmstat(record VmstatRecord) (err error) {
	inFile, err := os.Open(procStat)
	if err != nil {
		return
	}
	defer inFile.Close()
	for i, _ := range record.fields {
		record.fields[i] = 0
	}
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
	return
}

/* Polling */

// Poll sends a VmstatLine in the channel every period until duration.
// If cumul is false, it prints the diff of the accumulators, instead of the accumulators themselves
func Poll(period time.Duration, duration time.Duration, cumul bool, cout chan VmstatRecord) {
	startTime := time.Now()
	recordPtr, oldRecordPtr := newVmstatRecord(true), newVmstatRecord(true)
	for i := 0; (0 == duration) || (time.Since(startTime) <= duration); i++ {
		if i > 0 {
			time.Sleep(period)
		}
		err := parseVmstat(*recordPtr)
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
				cout <- recordPtr.diff(*oldRecordPtr)
			}
			oldRecordPtr, recordPtr = recordPtr, oldRecordPtr
		}
	}
	close(cout)
}
