package linescount

import (
	"bufio"
	"bytes"
	"io"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

const (
	Separator       = " "
)

/* Header is a list of field names. */

type header []string

func makeHeader() header {
	h := header(make([]string, 2))
	h[0] = "h"
	h[1] = "count"
	return h
}

func (h header) WriteTo(w io.Writer) (n int64, err error) { // implements io.WriterTo
	err = writeTo(w, strings.Join(h, Separator), &n)
	return
}

// init
var inputScanner = bufio.NewScanner(os.Stdin)

func writeTo(w io.Writer, v interface{}, p *int64) (err error) {
	m, err := w.Write([]byte(fmt.Sprint(v)))
	*p += int64(m)
	return
}

/* Record */

var Header = makeHeader()

type Record struct {
	Time           time.Time
	isCumul        bool
	count          uint
}

func newRecord(isCumul bool) *Record {
	recordPtr := new(Record)
	recordPtr.isCumul = isCumul
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
		err = writeTo(w, "d", &n)
	}
	if err != nil {
		return
	}
	err = writeTo(w, Separator, &n)
	if err != nil {
		return
	}
	err = writeTo(w, record.count, &n)
	if err != nil {
		return
	}
	return
}
func (recordPtr *Record) diff(prevRecord, diffRecord *Record) {
	diffRecord.Time = recordPtr.Time
	diffRecord.count = recordPtr.count - prevRecord.count
	return
}

func (recordPtr *Record) countlines(substring string, invert bool) (err error) {
    var line string
    for inputScanner.Scan() {
        line = inputScanner.Text()
        err := inputScanner.Err()
        if (err!=nil) && (err!=io.EOF) {
            return err
        }
        if (substring=="") || (strings.Contains(line, substring)!=invert) {
            recordPtr.count++
        }
        if (err==io.EOF) {
            break
        }

    }
	recordPtr.Time = time.Now()
	err = nil
	return
}

/* Polling */

// Poll sends a Record in the channel every period until duration.
// If cumul is false, it prints the diff of the accumulators, instead of the accumulators themselves
func Poll(substring string, invert bool, period time.Duration, duration time.Duration, cumul bool, cout chan Record) {
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
		err := recordPtr.countlines(substring, invert)
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
