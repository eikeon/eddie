package main

import (
	"fmt"
	"log"
	"os"
	"syscall"
)

const (
	EPOLLPRI      = 0x002
	EPOLL_CTL_ADD = 1
)

func GPIOInterrupt(number int, ch chan bool) (err error) {
	defer func() {
		close(ch)
	}()

	if _, err := os.Open(fmt.Sprintf("/sys/class/gpio/gpio%d", number)); err != nil {
		log.Println("exporting")
		ef, err := os.OpenFile("/sys/class/gpio/export", os.O_WRONLY, 0666)
		if err == nil {
			ef.WriteString("38\n")
			ef.Close()
		}
	}

	if ef, err := os.OpenFile(fmt.Sprintf("/sys/class/gpio/gpio%d/edge", number), os.O_WRONLY, 0666); err == nil {
		log.Println("setting edge")
		ef.Write([]byte("rising"))
		ef.Close()
	}

	if f, err := os.Open(fmt.Sprintf("/sys/class/gpio/gpio%d/value", number)); err == nil {
		epfd, err := syscall.EpollCreate(1)
		if err != nil {
			return err
		}
		ee := syscall.EpollEvent{EPOLLPRI, 0, int32(f.Fd()), 0}
		if err = syscall.EpollCtl(epfd, EPOLL_CTL_ADD, int(f.Fd()), &ee); err != nil {
			return err
		}
		b := make([]byte, 1)
		if _, err := f.Read(b); err != nil {
			return err
		}
		events := []syscall.EpollEvent{ee}
		for {
			if nr, err := syscall.EpollWait(epfd, events, -1); err != nil {
				return err
			} else if nr < 1 {
				continue
			}
			if _, err = f.Seek(0, 0); err != nil {
				return err
			}
			if _, err := f.Read(b); err != nil {
				return err
			}
			value := b[0] == '1'
			ch <- value
		}
	}
	return
}
