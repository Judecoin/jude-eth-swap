package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/judecoin/jude-eth-swap/alice"
	"github.com/judecoin/jude-eth-swap/bob"
	"github.com/judecoin/jude-eth-swap/net"
)

type node struct {
	amount uint
	alice  alice.Alice
	bob    bob.Bob
	host   net.Host
	done   chan struct{}
	outCh  chan<- *net.MessageInfo
	inCh   <-chan *net.MessageInfo
}

func (n *node) wait() {
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)

		select {
		case <-sigc:
			fmt.Println("signal interrupt, shutting down...")
			close(n.done)
		case <-n.done:
			fmt.Println("protocol complete, shutting down...")
		}

		os.Exit(0)
	}()

	wg.Wait()
}
