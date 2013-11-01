package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path"
	"runtime"
	"sync/atomic"
	"time"

	"code.google.com/p/go.net/websocket"
	"github.com/eikeon/gpio"
	"github.com/nogiushi/marvin/nog"
)

var Root = ""

func init() {
	_, filename, _, _ := runtime.Caller(0)
	Root = path.Dir(filename)
}

func Run(in <-chan nog.Message, out chan<- nog.Message) {
	out <- nog.Message{What: "started"}
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

	go func() {
		var pressCount int64
		for {
			select {
			case value := <-ch:
				if value {
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
	}()
	for _ = range in {
	}
}

func main() {
	log.Println("starting")

	for {
		ws, err := websocket.Dial("ws://marvin.local:80/message", "", "http://marvin.local/")
		if err != nil {
			log.Fatal(err)
		}

		fromWS := make(chan nog.Message)
		toWS := make(chan nog.Message)
		go func() {
			var m nog.Message
			if err := websocket.JSON.Receive(ws, &m); err == nil {
				fromWS <- m
			} else {
				log.Println("Message Websocket receive err:", err)
				close(fromWS)
				return
			}
		}()
		go func() {
			for m := range toWS {
				if err := websocket.JSON.Send(ws, &m); err != nil {
					log.Println("Message Websocket send err:", err)
					close(toWS)
					return
				}
			}
		}()
		Run(fromWS, toWS)
		time.Sleep(1 * time.Second)
	}

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
