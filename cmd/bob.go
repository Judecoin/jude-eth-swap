package main

import (
	"errors"
	"fmt"

	"github.com/judecoin/jude-eth-swap/net"
	"github.com/libp2p/go-libp2p-core/peer"
)

func (n *node) doProtocolBob() error {
	if err := n.host.Start(); err != nil {
		return err
	}
	defer n.host.Stop()

	outCh := make(chan *net.MessageInfo)
	n.host.SetOutgoingCh(outCh)
	n.outCh = outCh
	n.inCh = n.host.ReceivedMessageCh()

	for {
		select {
		case <-n.done:
		case msg := <-n.inCh:
			if err := n.handleMessageBob(msg.Who, msg.Message); err != nil {
				fmt.Printf("failed to handle message: error=%s\n", err)
			}
		}
	}

	n.wait()
	return nil
}

func (n *node) handleMessageBob(who peer.ID, msg net.Message) error {
	switch msg := msg.(type) {
	case *net.WantMessage:
		if msg.Want != "JUDE" {
			return errors.New("Bob has JUDE, peer does not want JUDE")
		}

		fmt.Println("found peer that wants JUDE, initiating swap protocol...")
		n.host.SetNextExpectedMessage(&net.SendKeysMessage{})

		sk, vk, err := n.bob.GenerateKeys()
		if err != nil {
			return err
		}

		out := &net.SendKeysMessage{
			PublicSpendKey: sk.Hex(),
			PrivateViewKey: vk.Hex(),
		}

		n.outCh <- &net.MessageInfo{
			Message: out,
			Who:     who,
		}
	default:
		return errors.New("unexpected message type")
	}

	return nil
}
