package redisproto

import (
	"testing"
	"bytes"
)

func TestWriter_Write(t *testing.T) {
	buff := bytes.NewBuffer(nil)
	w := NewWriter(buff)
	w.WriteBulkString("hello")
	if string(buff.Bytes()) != "$5\r\nhello\r\n" {
		t.Errorf("Unexpected WriteBulkString")
	}
}
