/*
Package heffalump contains HellPot's internal adaptation of Carl Johnson's
Heffalump Markov stream generator.

Original project: https://github.com/carlmjohnson/heffalump
*/
package heffalump

import (
	"bufio"
	"fmt"
	"io"
	"sync"

	"github.com/t3chn0m4g3/hellpot/internal/config"
)

var log = config.GetLogger()

// DefaultHeffalump represents a Heffalump type
var DefaultHeffalump *Heffalump

// Heffalump represents our buffer pool and markov map from Heffalump
type Heffalump struct {
	pool     *sync.Pool
	buffsize int
	mm       MarkovMap
}

// NewHeffalump instantiates a new Heffalump for markov generation and buffer/io operations
func NewHeffalump(mm MarkovMap, buffsize int) *Heffalump {
	return &Heffalump{
		pool: &sync.Pool{New: func() interface{} {
			b := make([]byte, buffsize)
			return b
		}},
		buffsize: buffsize,
		mm:       mm,
	}
}

// WriteHell writes markov chain heffalump hell to the provided io.Writer
func (h *Heffalump) WriteHell(bw *bufio.Writer) (n int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Interface("caller", r).Msg("panic recovered!")
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()

	buf := h.pool.Get().([]byte)
	defer h.pool.Put(buf)

	var wn int
	if wn, err = bw.WriteString("<html>\n<body>\n"); err != nil {
		return n, err
	}
	n += int64(wn)
	if err = bw.Flush(); err != nil {
		return n, err
	}

	r := h.mm.NewReader()
	for {
		read, readErr := r.Read(buf)
		if read > 0 {
			wn, err = bw.Write(buf[:read])
			n += int64(wn)
			if err != nil {
				return n, err
			}
			if wn != read {
				return n, io.ErrShortWrite
			}
			if err = bw.Flush(); err != nil {
				return n, err
			}
		}
		if readErr != nil {
			return n, readErr
		}
	}
}
