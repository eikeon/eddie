package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/eikeon/gpio"
	"github.com/eikeon/marvin/nog"
)

var Root = ""

func init() {
	_, filename, _, _ := runtime.Caller(0)
	Root = path.Dir(filename)
}

type Eddie struct {
	nog.InOut
}

func (e *Eddie) Run(in <-chan nog.Message, out chan<- nog.Message) {
	options := nog.BitOptions{Name: "Eddie", Required: false}
	if what, err := json.Marshal(&options); err == nil {
		out <- nog.NewMessage("Eddie", string(what), "register")
	} else {
		log.Println("StateChanged err:", err)
	}

	name := "eddie.html"
	if j, err := os.OpenFile(path.Join(Root, name), os.O_RDONLY, 0666); err == nil {
		if b, err := ioutil.ReadAll(j); err == nil {
			out <- nog.NewMessage("Eddie", string(b), "template")
		} else {
			log.Println("ERROR reading:", err)
		}
	} else {
		log.Println("WARNING: could not open ", name, err)
	}

	ch, err := gpio.GPIOInterrupt(38)
	if err != nil {
		panic(err)
	}

	var pressCount int64
	for {
		select {
		case m := <-in:
			if m.Why == "statechanged" {
				dec := json.NewDecoder(strings.NewReader(m.What))
				if err := dec.Decode(e); err != nil {
					log.Println("eddie decode err:", err)
				}
			}
		case value := <-ch:
			if value {
				out <- nog.NewMessage("Eddie", "hello", "eddie")
				atomic.AddInt64(&pressCount, 1)
				time.AfterFunc(time.Second, func() {
					atomic.AddInt64(&pressCount, -1)
				})
				switch pressCount {
				case 1:
					out <- nog.NewMessage("Eddie", "I am sleeping", "Eddie")
				case 2:
					out <- nog.NewMessage("Eddie", "set light All to nightlight", "Eddie")
				default:
					out <- nog.NewMessage("Eddie", "set light All to chime", "Eddie")
				}
			}
		}
	}
}

func main() {
	log.Println("starting")

	go nog.RemoteAdd("ws://marvin.local:80/message", "", "http://marvin.local/", &Eddie{})

	notifyChannel := make(chan os.Signal, 1)
	signal.Notify(notifyChannel, os.Interrupt)

	sig := <-notifyChannel
	switch sig {
	case os.Interrupt:
		log.Println("handling:", sig)
	default:
		log.Fatal("Unexpected Signal:", sig)
	}

	log.Println("stopping")

}
