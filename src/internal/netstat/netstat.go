package netstat

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
	defaultProcNetDev = "/proc/net/dev"
	Separator         = " "
)

const (
	// Inter-|   Receive                                                |  Transmit
	//  face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
	rxBytesIdx      = iota
	rxPacketsIdx    = iota
	rxErrsIdx       = iota
	rxDropsIdx      = iota
	rxFifoIdx       = iota
	rxFrameIdx      = iota
	rxCompressedIdx = iota
	rxMulticastIdx  = iota
	txBytesIdx      = iota
	txPacketsIdx    = iota
	txErrsIdx       = iota
	txDropsIdx      = iota
	txFifoIdx       = iota
	txCollsIdx      = iota
	txCarrierIdx    = iota
	txCompressedIdx = iota
	fieldsCount     = iota
)

var allFieldsDefs = []fieldDef{
	fieldDef{"rx", "bytes", true, nil},
	fieldDef{"rx", "packets", true, nil},
	fieldDef{"rx", "errs", true, nil},
	fieldDef{"rx", "drops", true, nil},
	fieldDef{"rx", "fifo", true, nil},
	fieldDef{"rx", "frame", true, nil},
	fieldDef{"rx", "compressed", true, nil},
	fieldDef{"rx", "multicast", true, nil},
	fieldDef{"tx", "bytes", true, nil},
	fieldDef{"tx", "packets", true, nil},
	fieldDef{"tx", "errs", true, nil},
	fieldDef{"tx", "drops", true, nil},
	fieldDef{"tx", "fifo", true, nil},
	fieldDef{"tx", "colls", true, nil},
	fieldDef{"tx", "carrier", true, nil},
	fieldDef{"tx", "compressed", true, nil},
}

/* Header is a list of field names. */

type header []string

func makeHeader(fdl []fieldDef) header {
	h := header(make([]string, 2+len(fdl)))
	h[0] = "interface"
	h[1] = "h"
	for i, d := range fdl {
		h[i+2] = d.String()
	}
	return h
}

func (h header) WriteTo(w io.Writer) (n int64, err error) { // implements io.WriterTo
	err = writeTo(w, strings.Join(h, Separator), &n)
	return
}

var procNetDev string = defaultProcNetDev

func init() {
	fsRoot := os.Getenv("FS_ROOT")
	if fsRoot != "" {
		procNetDev = path.Join(fsRoot, defaultProcNetDev)
	}
}

func (recordPtr *Record) parseLineToFields(line string) (err error) {
	parsedFields := strings.Fields(line)
	prefix := parsedFields[0]
	if prefix[len(prefix)-1] != ':' {
		return
	}
	iface := prefix[:len(prefix)-1]
	recordFields := recordPtr.getFields(iface)
	var uint64field uint64
	for i, str := range parsedFields[1:] {
		uint64field, err = strconv.ParseUint(str, 10, 0)
		if err != nil {
			return
		}
		recordFields[i] = uint(uint64field)
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
	Time      time.Time
	isCumul   bool
	fieldsMap map[string][]uint // key is the interface
}

func newRecord(isCumul bool) *Record {
	recordPtr := new(Record)
	recordPtr.isCumul = isCumul
	recordPtr.fieldsMap = make(map[string][]uint)
	return recordPtr
}

func (recordPtr *Record) getFields(iface string) (fields []uint) {
	fields, ok := recordPtr.fieldsMap[iface]
	if ok {
		return
	}
	fields = make([]uint, fieldsCount)
	recordPtr.fieldsMap[iface] = fields
	return
}

func (recordPtr *Record) String() string { // implements fmt.Stringer
	buf := new(bytes.Buffer)
	recordPtr.WriteTo(buf)
	return buf.String()
}
func (record Record) WriteTo(w io.Writer) (n int64, err error) { // implements io.WriterTo
	for iface, fields := range record.fieldsMap {
		err = writeTo(w, iface, &n)
		if err != nil {
			return
		}
		err = writeTo(w, Separator, &n)
		if err != nil {
			return
		}
		if record.isCumul {
			err = writeTo(w, "a", &n)
		} else {
			err = writeTo(w, "d", &n)
		}
		if err != nil {
			return
		}
		for _, field := range fields {
			err = writeTo(w, Separator, &n)
			if err != nil {
				return
			}
			err = writeTo(w, field, &n)
			if err != nil {
				return
			}
		}
		err = writeTo(w, "\n", &n)
		if err != nil {
			return
		}
	}
	return
}
func (recordPtr *Record) diff(prevRecord, diffRecord *Record) {
	diffRecord.Time = recordPtr.Time
	for iface, fields := range recordPtr.fieldsMap {
		prevFields := prevRecord.getFields(iface)
		diffFields := diffRecord.getFields(iface)
		for i, field := range fields {
			if allFieldsDefs[i].isAccumulator {
				diffFields[i] = field - prevFields[i]
			} else {
				diffFields[i] = field
			}
		}
	}
	return
}

func (recordPtr *Record) parse() (err error) {
	inFile, err := os.Open(procNetDev)
	if err != nil {
		return
	}
	defer inFile.Close()
	recordPtr.Time = time.Now()
	for _, fields := range recordPtr.fieldsMap {
		for i, _ := range fields {
			fields[i] = 0
		}
	}
	scanner := bufio.NewScanner(inFile)
	for j := 0; scanner.Scan(); j++ {
		line := scanner.Text()
		err = recordPtr.parseLineToFields(line)
		if err != nil {
			return
		}
	}
	err = scanner.Err()
	if err != nil {
		return
	}
	for i, fd := range allFieldsDefs {
		if fd.calculator != nil {
			for _, fields := range recordPtr.fieldsMap {
				fields[i] = fd.calculator(fields)
			}
		}
	}
	return
}

/* Polling */

// Poll sends a Record in the channel every period until duration.
// If cumul is false, it prints the diff of the accumulators, instead of the accumulators themselves
func Poll(period time.Duration, duration time.Duration, cumul bool, cout chan Record) {
	startTime := time.Now()
	recordPtr := newRecord(true)
	oldRecordPtr := newRecord(true)
	diffRecordPtr := newRecord(false)
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
				cout <- *diffRecordPtr
			}
			oldRecordPtr, recordPtr = recordPtr, oldRecordPtr
		}
	}
	close(cout)
}
