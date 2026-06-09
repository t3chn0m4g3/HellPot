package heffalump

import "testing"

func TestMarkovReaderMakesProgressWithTinyBuffers(t *testing.T) {
	r := DefaultMarkovMap.NewReader()
	buf := make([]byte, 1)

	for i := 0; i < 128; i++ {
		n, err := r.Read(buf)
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}
		if n != 1 {
			t.Fatalf("Read() = %d, want 1", n)
		}
	}
}
