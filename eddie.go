package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync/atomic"
	"time"
)

func doTransition(name string) {
	host := "10.0.1.16" //"marvin.local"
	values := url.Values{"do_transition": {name}}
	if r, err := http.PostForm("http://"+host+"/post", values); err != nil {
		log.Println(err)
	} else {
		log.Println("response:", r)
	}
}

func main() {
	log.Println("starting")

	notifyChannel := make(chan os.Signal, 1)
	signal.Notify(notifyChannel, os.Interrupt)

	ch := make(chan bool, 1)
	go GPIOInterrupt(38, ch)

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
					doTransition("all off")
				case 2:
					doTransition("all nightlight")
				default:
					doTransition("chime")
				}
			}
		case sig := <-notifyChannel:
			switch sig {
			case os.Interrupt:
				log.Println("handling:", sig)
				goto Done
			default:
				log.Fatal("Unexpected Signal:", sig)
			}
		}
	}
Done:
	log.Println("stopping")

}
