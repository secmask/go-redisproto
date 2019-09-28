package redisproto

import (
	"bytes"
	"errors"
	"io"
)

var (
	ExpectNumber   = &ProtocolError{"Expect Number"}
	ExpectNewLine  = &ProtocolError{"Expect Newline"}
	ExpectTypeChar = &ProtocolError{"Expect TypeChar"}

	InvalidNumArg   = errors.New("TooManyArg")
	InvalidBulkSize = errors.New("Invalid bulk size")
	LineTooLong     = errors.New("LineTooLong")

	ReadBufferInitSize = 1 << 16
	MaxBulkSize        = 1 << 16
	MaxTelnetLine      = 1 << 10
	spaceSlice         = []byte{' '}
	emptyBulk          = [0]byte{}
)

type ProtocolError struct {
	message string
}

func (p *ProtocolError) Error() string {
	return p.message
}

type Command struct {
	argv [][]byte
	last bool
}

func (c *Command) Get(index int) []byte {
	if index >= 0 && index < len(c.argv) {
		return c.argv[index]
	} else {
		return nil
	}
}

func (c *Command) ArgCount() int {
	return len(c.argv)
}

func (c *Command) IsLast() bool {
	return c.last
}

type Parser struct {
	reader        io.Reader
	buffer        []byte
	parsePosition int
	writeIndex    int
	maxNumArg     int
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func NewParser(reader io.Reader) *Parser {
	return &Parser{reader: reader, buffer: make([]byte, ReadBufferInitSize), maxNumArg: 20}
}

func NewParserWithMaxArgCount(reader io.Reader, maxArgs int) *Parser {
	return &Parser{reader: reader, buffer: make([]byte, ReadBufferInitSize), maxNumArg: maxArgs}
}

// ensure that we have enough space for writing 'req' byte
func (r *Parser) requestSpace(req int) {
	ccap := cap(r.buffer)
	if r.writeIndex+req > ccap {
		newbuff := make([]byte, max(ccap*2, ccap+req+ReadBufferInitSize))
		copy(newbuff, r.buffer)
		r.buffer = newbuff
	}
}
func (r *Parser) readSome(min int) error {
	r.requestSpace(min)
	nr, err := io.ReadAtLeast(r.reader, r.buffer[r.writeIndex:], min)
	if err != nil {
		return err
	}
	r.writeIndex += nr
	return nil
}

// check for at least 'num' byte available in buffer to use, wait if need
func (r *Parser) requireNBytes(num int) error {
	a := r.writeIndex - r.parsePosition
	if a >= num {
		return nil
	}
	if err := r.readSome(num - a); err != nil {
		return err
	}
	return nil
}
func (r *Parser) readNumber() (int, error) {
	var neg bool = false
	err := r.requireNBytes(1)
	if err != nil {
		return 0, err
	}
	switch r.buffer[r.parsePosition] {
	case '-':
		neg = true
		r.parsePosition++
		break
	case '+':
		neg = false
		r.parsePosition++
		break
	}
	var num uint64 = 0
	var startpos int = r.parsePosition
OUTTER:
	for {
		for i := r.parsePosition; i < r.writeIndex; i++ {
			c := r.buffer[r.parsePosition]
			if c >= '0' && c <= '9' {
				num = num*10 + uint64(c-'0')
				r.parsePosition++
			} else {
				break OUTTER
			}
		}
		if r.parsePosition == r.writeIndex {
			if e := r.readSome(1); e != nil {
				return 0, e
			}
		}
	}
	if r.parsePosition == startpos {
		return 0, ExpectNumber
	}
	if neg {
		return -int(num), nil
	} else {
		return int(num), nil
	}

}
func (r *Parser) discardNewLine() error {
	if e := r.requireNBytes(2); e != nil {
		return e
	}
	if r.buffer[r.parsePosition] == '\r' && r.buffer[r.parsePosition+1] == '\n' {
		r.parsePosition += 2
		return nil
	}
	return ExpectNewLine
}

func (r *Parser) parseBinary() (*Command, error) {
	r.parsePosition++
	numArg, err := r.readNumber()
	if err != nil {
		return nil, err
	}
	var e error
	if e = r.discardNewLine(); e != nil {
		return nil, e
	}
	switch {
	case numArg == -1:
		return nil, r.discardNewLine() // null array
	case numArg < -1:
		return nil, InvalidNumArg
	case numArg > r.maxNumArg:
		return nil, InvalidNumArg
	}
	argv := make([][]byte, 0, numArg)
	for i := 0; i < numArg; i++ {
		if e = r.requireNBytes(1); e != nil {
			return nil, e
		}
		if r.buffer[r.parsePosition] != '$' {
			return nil, ExpectTypeChar
		}
		r.parsePosition++
		var plen int
		if plen, e = r.readNumber(); e != nil {
			return nil, e
		}
		if e = r.discardNewLine(); e != nil {
			return nil, e
		}
		switch {
		case plen == -1:
			argv = append(argv, nil) // null bulk
		case plen == 0:
			argv = append(argv, emptyBulk[:]) // empty bulk
		case plen > 0 && plen <= MaxBulkSize:
			if e = r.requireNBytes(plen); e != nil {
				return nil, e
			}
			argv = append(argv, r.buffer[r.parsePosition:(r.parsePosition+plen)])
			r.parsePosition += plen
		default:
			return nil, InvalidBulkSize
		}
		if e = r.discardNewLine(); e != nil {
			return nil, e
		}
	}
	return &Command{argv: argv}, nil
}

func (r *Parser) parseTelnet() (*Command, error) {
	nlPos := -1
	for {
		nlPos = bytes.IndexByte(r.buffer, '\n')
		if nlPos == -1 {
			if e := r.readSome(1); e != nil {
				return nil, e
			}
		} else {
			break
		}
		if r.writeIndex > MaxTelnetLine {
			return nil, LineTooLong
		}
	}
	r.parsePosition = r.writeIndex // we don't support pipeline in telnet mode
	return &Command{argv: bytes.Split(r.buffer[:nlPos-1], spaceSlice)}, nil
}

func (r *Parser) reset() {
	r.writeIndex = 0
	r.parsePosition = 0
	//r.buffer = make([]byte, len(r.buffer))
}

func (r *Parser) ReadCommand() (*Command, error) {
	// if the buffer is empty, try to fetch some
	if r.parsePosition >= r.writeIndex {
		if err := r.readSome(1); err != nil {
			return nil, err
		}
	}

	var cmd *Command
	var err error
	if r.buffer[r.parsePosition] == '*' {
		cmd, err = r.parseBinary()
	} else {
		cmd, err = r.parseTelnet()
	}
	if r.parsePosition >= r.writeIndex {
		if cmd != nil {
			cmd.last = true
		}
		r.reset()
	}
	return cmd, err
}

func (r *Parser) Commands() <-chan *Command {
	cmds := make(chan *Command)
	go func() {
		for cmd, err := r.ReadCommand(); err == nil; cmd, err = r.ReadCommand() {
			cmds <- cmd
		}
		close(cmds)

	}()
	return cmds
}
