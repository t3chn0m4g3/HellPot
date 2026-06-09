package heffalump

import (
	"bufio"
	"io"
	"math/rand"
	"strings"
	"unicode"
	"unicode/utf8"

	"git.tcp.direct/kayos/common/squish"
)

var DefaultMarkovMap MarkovMap

func init() {
	// DefaultMarkovMap is a Markov chain based on src.
	src, err := squish.UnpackStr(srcGz)
	if err != nil {
		panic(err)
	}
	if len(src) < 1 {
		panic("failed to unpack source")
	}
	DefaultMarkovMap = MakeMarkovMap(strings.NewReader(src))
	DefaultHeffalump = NewHeffalump(DefaultMarkovMap, 100*1<<10)
}

// ScanHTML is a basic split function for a Scanner that returns each
// space-separated word of text or HTML tag, with surrounding spaces deleted.
// It will never return an empty string. The definition of space is set by
// unicode.IsSpace.
func ScanHTML(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// Skip leading spaces.
	var r rune
	var start = 0
	for width := 0; start < len(data); start += width {
		r, width = utf8.DecodeRune(data[start:])
		if !unicode.IsSpace(r) {
			break
		}
	}
	switch {
	case r == '<':
		// Scan until closing bracket
		for i := start; i < len(data); i++ {
			if data[i] == '>' {
				return i + 1, data[start : i+1], nil
			}
		}
	default:
		// Scan until space, marking end of word.
		for width, i := 0, start; i < len(data); i += width {
			var r rune
			r, width = utf8.DecodeRune(data[i:])
			if unicode.IsSpace(r) {
				return i + width, data[start:i], nil
			}
			if r == '<' {
				return i, data[start:i], nil
			}
		}
	}
	// If we're at EOF, we have a final, non-empty, non-terminated word. Return it.
	if atEOF && len(data) > start {
		return len(data), data[start:], nil
	}
	// Request more data.
	return start, nil, nil
}

type tokenPair [2]string

// MarkovMap is a map that acts as a Markov chain generator.
type MarkovMap map[tokenPair][]string

// Reader streams generated Markov tokens while preserving chain state across
// small reads.
type Reader struct {
	mm      MarkovMap
	w1      string
	w2      string
	pending string
}

// MakeMarkovMap makes an empty MakeMarkov and fills it with r.
func MakeMarkovMap(r io.Reader) MarkovMap {
	m := MarkovMap{}
	m.Fill(r)
	return m
}

// Fill adds all the tokens in r to a MarkovMap
func (mm MarkovMap) Fill(r io.Reader) {
	var w1, w2, w3 string

	s := bufio.NewScanner(r)
	s.Split(ScanHTML)
	for s.Scan() {
		w3 = s.Text()
		mm.Add(w1, w2, w3)
		w1, w2 = w2, w3
	}

	mm.Add(w1, w2, w3)
}

// Add adds a three token sequence to the map.
func (mm MarkovMap) Add(w1, w2, w3 string) {
	p := tokenPair{w1, w2}
	mm[p] = append(mm[p], w3)
}

// Get pseudo-randomly chooses a possible suffix to w1 and w2.
func (mm MarkovMap) Get(w1, w2 string) string {
	p := tokenPair{w1, w2}
	suffix, ok := mm[p]
	if !ok {
		return ""
	}
	// We don't care about cryptographically sound entropy here, ignore gosec G404.
	/* #nosec */
	r := rand.Intn(len(suffix))
	return suffix[r]
}

// NewReader returns an endless reader backed by mm.
func (mm MarkovMap) NewReader() *Reader {
	return &Reader{mm: mm}
}

// Read fills p with data from calling Get on the MarkovMap.
func (mm MarkovMap) Read(p []byte) (n int, err error) {
	return mm.NewReader().Read(p)
}

// Read fills p with generated Markov output. It never returns 0, nil for a
// non-empty buffer, which keeps io.Copy and bufio.Writer.ReadFrom streaming.
func (r *Reader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	for n < len(p) {
		if r.pending == "" {
			w3 := r.mm.Get(r.w1, r.w2)
			if w3 == "" {
				r.w1, r.w2 = "", ""
				w3 = "\n"
			} else {
				r.w1, r.w2 = r.w2, w3
				w3 += "\n"
			}
			r.pending = w3
		}

		copied := copy(p[n:], r.pending)
		n += copied
		r.pending = r.pending[copied:]
	}
	return n, nil
}
