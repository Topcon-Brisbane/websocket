package websocket

import (
	"bytes"
	"compress/flate"
	"fmt"
	"io"
	"io/ioutil"
	"testing"
)

type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }

func TestTruncWriter(t *testing.T) {
	const data = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijlkmnopqrstuvwxyz987654321"
	for n := 1; n <= 10; n++ {
		var b bytes.Buffer
		w := &truncWriter{w: nopCloser{&b}}
		p := []byte(data)
		for len(p) > 0 {
			m := len(p)
			if m > n {
				m = n
			}
			w.Write(p[:m])
			p = p[m:]
		}
		if b.String() != data[:len(data)-len(w.p)] {
			t.Errorf("%d: %q", n, b.String())
		}
	}
}

func textMessages(num int) [][]byte {
	messages := make([][]byte, num)
	for i := 0; i < num; i++ {
		msg := fmt.Sprintf("planet: %d, country: %d, city: %d, street: %d", i, i, i, i)
		messages[i] = []byte(msg)
	}
	return messages
}

func BenchmarkWriteNoCompression(b *testing.B) {
	w := ioutil.Discard
	c := newConn(fakeNetConn{Reader: nil, Writer: w}, false, 1024, 1024)
	messages := textMessages(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.WriteMessage(TextMessage, messages[i%len(messages)])
	}
	b.ReportAllocs()
}

func BenchmarkWriteWithCompression(b *testing.B) {
	w := ioutil.Discard
	c := newConn(fakeNetConn{Reader: nil, Writer: w}, false, 1024, 1024)
	messages := textMessages(100)
	c.enableWriteCompression = true
	c.newCompressionWriter = compressNoContextTakeover
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.WriteMessage(TextMessage, messages[i%len(messages)])
	}
	b.ReportAllocs()
}

func BenchmarkWriteWithCompressionOfContextTakeover(b *testing.B) {
	w := ioutil.Discard
	c := newConn(fakeNetConn{Reader: nil, Writer: w}, false, 1024, 1024)
	messages := textMessages(100)
	c.enableWriteCompression = true
	c.contextTakeover = true
	var f contextTakeoverWriterFactory
	f.fw, _ = flate.NewWriter(&f.tw, 2) // level is specified in Dialer, Upgrader
	c.newCompressionWriter = f.newCompressionWriter
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.WriteMessage(TextMessage, messages[i%len(messages)])
	}
	b.ReportAllocs()
}

func BenchmarkReadWithCompression(b *testing.B) {
	w := ioutil.Discard
	c := newConn(fakeNetConn{Reader: nil, Writer: w}, false, 1024, 1024)
	c.enableWriteCompression = true
	c.newDecompressionReader = decompressNoContextTakeover
	messages := textMessages(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(messages[i%len(messages)])
		reader := c.newDecompressionReader(r, nil)
		ioutil.ReadAll(reader)
	}
	b.ReportAllocs()
}

func BenchmarkReadWithCompressionOfContextTakeover(b *testing.B) {
	w := ioutil.Discard
	c := newConn(fakeNetConn{Reader: nil, Writer: w}, false, 1024, 1024)
	c.enableWriteCompression = true
	c.contextTakeover = true
	c.newDecompressionReader = decompressContextTakeover
	messages := textMessages(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(messages[i%len(messages)])
		reader := c.newDecompressionReader(r, c.rxDict)
		ioutil.ReadAll(reader)
	}
	b.ReportAllocs()
}

func TestValidCompressionLevel(t *testing.T) {
	c := newConn(fakeNetConn{}, false, 1024, 1024)
	for _, level := range []int{minCompressionLevel - 1, maxCompressionLevel + 1} {
		if err := c.SetCompressionLevel(level); err == nil {
			t.Errorf("no error for level %d", level)
		}
	}
	for _, level := range []int{minCompressionLevel, maxCompressionLevel} {
		if err := c.SetCompressionLevel(level); err != nil {
			t.Errorf("error for level %d", level)
		}
	}
}
