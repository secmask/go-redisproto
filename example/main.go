package main 

import (
	"github.com/secmask/go-redisproto"
	"net"
	"log"
	"bufio"
	"strings"
	"sync"
)
var (
	mchan map[string] chan []byte
	lock *sync.Mutex
)
func init(){
	mchan = make(map[string] chan []byte)
	lock = &sync.Mutex{}
}
func execLPush(qn string, data []byte){
	lock.Lock()
	defer lock.Unlock()
	c,ok := mchan[qn]
	if ok{
		c <- data
	}else{
		c = make(chan []byte,1000000)
		mchan[qn] = c
		c <- data
	}
}
func execRpop(qn string) []byte{
	lock.Lock()
	defer lock.Unlock()
	c,ok := mchan[qn]
	if ok{
		select{
			case msg:= <-c:
				return msg
			default:
				return nil
		}
	}else{
		return nil
	}
}
func handleConnection(conn net.Conn){
	defer conn.Close()
	parser := redisproto.NewParser(conn)
	w := bufio.NewWriter(conn)
	var ew error
	for{
		command,err:=parser.ReadCommand()
		if err!=nil{
			_,ok:= err.(*redisproto.ProtocolError)
			if ok{
				ew = redisproto.SendError(err.Error(),w)
			}else{
				log.Println(err, " closed connection to ",conn.RemoteAddr())
				break
			}
		}else{
			cmd:=strings.ToUpper(string(command.Get(0)))
			switch cmd{
				case "LPUSH":
					qn:=string(command.Get(1))
					execLPush(qn,command.Get(2))
					ew = redisproto.SendInt(1,w)
					break
				case "RPOP":
					qn:=string(command.Get(1))
					data := execRpop(qn)
					ew = redisproto.SendBulk(data,w)
					break
				default:
					ew = redisproto.SendError("Command not support",w)	
			}
		}
		if ew!=nil{
			log.Println("ew",ew)
			break
		}
	}
}


func main(){
	listener,err:=net.Listen("tcp",":6380")
	if err!=nil{
		panic(err)
	}
	for{
		conn,err:=listener.Accept()
		if err!=nil{
			log.Println("Error on accept: ",err)
			continue
		}
		go handleConnection(conn)
	}
}