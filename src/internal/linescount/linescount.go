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
	h := header(make([]string, 3))
	h[0] = "h"
	h[1] = "count"
	h[2] = "bytes"
	return h
}

func (h header) WriteTo(w io.Writer) (n int64, err error) { // implements io.WriterTo
	err = writeTo(w, strings.Join(h, Separator), &n)
	return
}

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
	bytes          uint
}

func newRecord(isCumul bool) *Record {
	recordPtr := new(Record)
	recordPtr.count = 0
	recordPtr.bytes = 0
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
	err = writeTo(w, Separator, &n)
	if err != nil {
		return
	}
	err = writeTo(w, record.bytes, &n)
	if err != nil {
		return
	}
	return
}

func (recordPtr *Record) diff(prevCount uint, prevBytes uint, diffRecord *Record) {
	diffRecord.Time = recordPtr.Time
	diffRecord.count = recordPtr.count - prevCount
	diffRecord.bytes = recordPtr.bytes - prevBytes
	return
}

// Non-blocking read from Stdin inspired by http://stackoverflow.com/a/27210020
func (recordPtr *Record) countlines(cout chan []byte, substring string, invert bool) (ok bool) {
    var bytes []byte
    loop: for {
        //log.Println("Waiting for 1 line")
        select {
            case bytes, ok = <-cout:
                if !ok {
                    // Reached error or EOF
                    return
                }
                //log.Println("Read 1 line")
                //log.Println(line)
                if (substring=="") || (strings.Contains(string(bytes), substring)!=invert) {
                    recordPtr.count++
		    recordPtr.bytes += uint(len(bytes))
                }
            case <-time.After(1 * time.Second): // Change this delay?
                break loop
        }
    }
	recordPtr.Time = time.Now()
	ok = true
	return
}

// Non-blocking read from Stdin inspired by http://stackoverflow.com/a/27210020
func ReadStdin(cout chan []byte) {
    var inputReader = bufio.NewReader(os.Stdin)
    for {
	bytes, err := inputReader.ReadBytes('\n')
        if err != nil {
            if err!= io.EOF { log.Println(err) }
            close(cout)
            return
        }
        cout <- bytes
    }
}

/* Polling */

// Poll sends a Record in the channel every period until duration.
// If cumul is false, it prints the diff of the accumulators, instead of the accumulators themselves
func Poll(substring string, invert bool, period time.Duration, duration time.Duration, cumul bool, cout chan Record) {
	startTime := time.Now()
	recordPtr := newRecord(true)
	var oldCount, oldBytes uint
	diffRecordPtr := newRecord(false)
	chstdin := make(chan []byte)
	go ReadStdin(chstdin)
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
		//log.Println("Counting lines")
		ok := recordPtr.countlines(chstdin, substring, invert)
		if !ok {
		    log.Println("Stdin terminated")
		}
		//log.Println("Counted lines")
		if cumul {
			cout <- *recordPtr
		} else {
			if i < 1 {
				cout <- *recordPtr
			} else {
				recordPtr.diff(oldCount, oldBytes, diffRecordPtr)
				cout <- *diffRecordPtr
			}
			oldCount = recordPtr.count
			oldBytes = recordPtr.bytes
		}
		if !ok {
		    break
		}
	}
	close(cout)
}
