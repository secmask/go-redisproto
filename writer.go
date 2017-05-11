package redisproto

import (
	"bufio"
	"fmt"
	"io"
)

var (
	star   = []byte{'*'}
	colon  = []byte{':'}
	dollar = []byte{'$'}
	plus   = []byte{'+'}
	subs   = []byte{'-'}
	//newLine  = []byte{'\r', '\n'}
	//nilBulk  = []byte{'$', '-', '1', '\r', '\n'}
	//nilArray = []byte{'*', '-', '1', '\r', '\n'}
)

type Writer struct {
	w io.Writer
}

func NewWriter(sink io.Writer) *Writer {
	return &Writer{
		w: sink,
	}
}

func (w *Writer) Write(data []byte) (int, error) {
	return w.w.Write(data)
}

func (w *Writer) Flush() error {
	if f, ok := w.w.(*bufio.Writer); ok {
		return f.Flush()
	}
	return nil
}

func (w *Writer) WriteInt(val int64) error {
	w.Write(colon)
	w.Write([]byte(intToString(val)))
	_, err := w.Write(newLine)
	return err
}

func (w *Writer) WriteBulk(val []byte) error {
	if val == nil {
		_, err := w.Write(nilBulk)
		return err
	}
	w.Write(dollar)
	w.Write([]byte(intToString(int64(len(val)))))
	w.Write(newLine)
	w.Write(val)
	_, err := w.Write(newLine)
	return err
}

func (w *Writer) WriteBulkString(s string) error {
	return w.WriteBulk([]byte(s))
}

func (w *Writer) WriteSimpleString(s string) error {
	w.Write(plus)
	w.Write([]byte(s))
	_, err := w.Write(newLine)
	return err
}

func (w *Writer) WriteError(s string) error {
	w.Write(subs)
	w.Write([]byte(s))
	_, err := w.Write(newLine)
	return err
}

func (w *Writer) WriteObjects(objs ...interface{}) error {
	if objs == nil {
		_, err := w.Write(nilArray)
		return err
	}

	w.Write(star)
	w.Write([]byte(intToString(int64(len(objs)))))
	w.Write(newLine)

	numArg := len(objs)
	for i := 0; i < numArg; i++ {
		v := objs[i]
		if v == nil {
			if err := w.WriteBulk(nil); err != nil {
				return err
			}
			continue
		}
		switch v := v.(type) {
		case []byte:
			if err := w.WriteBulk(v); err != nil {
				return err
			}
		case string:
			if err := w.WriteBulkString(v); err != nil {
				return err
			}
		case int:
			if err := w.WriteInt(int64(v)); err != nil {
				return err
			}
		case int32:
			if err := w.WriteInt(int64(v)); err != nil {
				return err
			}
		case int64:
			if err := w.WriteInt(int64(v)); err != nil {
				return err
			}
		default:
			return fmt.Errorf("Value not suppport %v", v)
		}
	}
	return nil
}

func (w *Writer) WriteBulks(bulks ...[]byte) error {
	if bulks == nil {
		_, err := w.Write(nilArray)
		return err
	}

	w.Write(star)
	numElement := len(bulks)
	w.Write([]byte(intToString(int64(numElement))))
	w.Write(newLine)

	for i := 0; i < numElement; i++ {
		if err := w.WriteBulk(bulks[i]); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) WriteBulkStrings(bulks []string) error {
	if bulks == nil {
		_, err := w.Write(nilArray)
		return err
	}

	w.Write(star)
	numElement := len(bulks)
	w.Write([]byte(intToString(int64(numElement))))
	w.Write(newLine)

	for i := 0; i < numElement; i++ {
		if err := w.WriteBulkString(bulks[i]); err != nil {
			return err
		}
	}
	return nil
}
