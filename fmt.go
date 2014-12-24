package redisproto

import (
	"strconv"
	"bufio"
)
var (
	newLine = []byte{'\r','\n'}
	nilBulk = []byte{'$','-','1','\r','\n'}
)
func intToString(val int64) string{
	return strconv.FormatInt(val,10)
}
func SendError(msg string,w *bufio.Writer) error{
	resp:="-"+msg+"\r\n"
	_,e := w.Write([]byte(resp))
	if e!=nil{
		return e
	}
	return w.Flush()
}

func SendString(msg string,w *bufio.Writer) error{
	resp:="+"+msg+"\r\n"
	_,e := w.Write([]byte(resp))
	if e!=nil{
		return e
	}
	return w.Flush()
}

func SendInt(val int64,w *bufio.Writer) error{
	resp:=":"+intToString(val)+"\r\n"
	_,e := w.Write([]byte(resp))
	if e!=nil{
		return e
	}
	return w.Flush()
}
func SendBulk(val []byte,w *bufio.Writer) error{
	if e:=sendBulk(val,w); e!=nil{
		return e
	}
	return w.Flush()
}
func sendBulk(val []byte,w *bufio.Writer) error{
	if val==nil{
		_,e := w.Write(nilBulk)
		if e!=nil{
			return e
		}
		return nil
	}
	pre:="$"+intToString(int64(len(val)))+"\r\n"
	_,e := w.Write([]byte(pre))
	if e!=nil{
		return e
	}
	_,e = w.Write(val)
	if e!=nil{
		return e
	}
	_,e = w.Write(newLine)
	if e!=nil{
		return e
	}
	return nil
}
func SendBulks(vals [][]byte,w *bufio.Writer) error{
	if e:=sendBulks(vals,w); e!=nil{
		return e
	}
	return w.Flush()
}
func sendBulks(vals [][]byte,w *bufio.Writer) error{
	pre:="*"+intToString(int64(len(vals)))+"\r\n"
	var e error
	_,e = w.Write([]byte(pre))
	if e!=nil{
		return e
	}
	numArg:=len(vals)
	for i:=0;i<numArg;i++{
		if e = SendBulk(vals[i],w); e!=nil{
			return e
		}
	}
	return nil
}
func SendBulkString(str string,w *bufio.Writer) error{
	return SendBulk([]byte(str),w)
}
func SendBulkStrings(strs []string,w *bufio.Writer) error{
	t:=make([][]byte,0,len(strs))
	for i:=0;i<len(strs);i++{
		t = append(t,[]byte(strs[i]))
	}
	return SendBulks(t,w)
}