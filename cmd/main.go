package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli"

	"github.com/judecoin/jude-eth-swap/alice"
	"github.com/judecoin/jude-eth-swap/bob"
	"github.com/judecoin/jude-eth-swap/net"
)

const (
	defaultAliceJudecoinEndpoint = "http://127.0.0.1:16060/json_rpc"
	defaultBobJudecoinEndpoint   = "http://127.0.0.1:16063/json_rpc"
	defaultEthEndpoint           = "http://localhost:8545"
	defaultPrivKeyAlice          = "4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d"
	defaultPrivKeyBob            = "6cbed15c793ce57650b9877cf6fa156fbef513c4e6134f022a85b1ffdd59b2a1"
	defaultAlicePort             = 9933
	defaultBobPort               = 9934
	defaultAliceLibp2pKey        = "alice.key"
	defaultBobLibp2pKey          = "bob.key"
)

var (
	app = &cli.App{
		Name:   "atomic-swap",
		Usage:  "A program for doing atomic swaps between ETH and JUDE",
		Action: startAction,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "alice",
				Usage: "run as Alice (have ETH, want JUDE)",
			},
			&cli.BoolFlag{
				Name:  "bob",
				Usage: "run as Bob (have JUDE, want ETH)",
			},
			&cli.UintFlag{
				Name:  "amount",
				Value: 0,
				Usage: "amount to swap (in smallest units of chain)",
			},
			&cli.StringFlag{
				Name:  "judecoin-endpoint",
				Usage: "judecoin-wallet-rpc endpoint",
			},
			&cli.StringFlag{
				Name:  "ethereum-endpoint",
				Usage: "ethereum client endpoint",
			},
			&cli.StringFlag{
				Name:  "ethereum-privkey",
				Usage: "ethereum private key hex string",
			},
			&cli.StringFlag{
				Name:  "bootnodes",
				Usage: "comma-separated string of libp2p bootnodes",
			},
		},
	}
)

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func startAction(c *cli.Context) error {
	fmt.Println("starting...")
	amount := c.Uint("amount")
	if amount == 0 {
		return errors.New("must specify amount")
	}

	if c.Bool("alice") {
		if err := runAlice(c, amount); err != nil {
			return err
		}

		return nil
	}

	if c.Bool("bob") {
		if err := runBob(c, amount); err != nil {
			return err
		}

		return nil
	}

	return errors.New("must specify either --alice or --bob")
}

func runAlice(c *cli.Context, amount uint) error {
	var (
		judecoinEndpoint, ethEndpoint, ethPrivKey string
	)

	if c.String("judecoin-endpoint") != "" {
		judecoinEndpoint = c.String("judecoin-endpoint")
	} else {
		judecoinEndpoint = defaultAliceJudecoinEndpoint
	}

	if c.String("ethereum-endpoint") != "" {
		ethEndpoint = c.String("ethereum-endpoint")
	} else {
		ethEndpoint = defaultEthEndpoint
	}

	if c.String("ethereum-privkey") != "" {
		ethPrivKey = c.String("ethereum-privkey")
	} else {
		ethPrivKey = defaultPrivKeyAlice
	}

	alice, err := alice.NewAlice(judecoinEndpoint, ethEndpoint, ethPrivKey)
	if err != nil {
		return err
	}

	fmt.Println("instantiated Alice session", alice)

	var bootnodes []string
	if c.String("bootnodes") != "" {
		bootnodes = strings.Split(c.String("bootnodes"), ",")
	}

	host, err := net.NewHost(defaultAlicePort, "JUDE", defaultAliceLibp2pKey, bootnodes)
	if err != nil {
		return err
	}

	n := &node{
		alice: alice,
		host:  host,
		done:  make(chan struct{}),
	}

	return n.doProtocolAlice()
}

func runBob(c *cli.Context, amount uint) error {
	var (
		judecoinEndpoint, ethPrivKey string
	)

	if c.String("judecoin-endpoint") != "" {
		judecoinEndpoint = c.String("judecoin-endpoint")
	} else {
		judecoinEndpoint = defaultBobJudecoinEndpoint
	}

	if c.String("ethereum-privkey") != "" {
		ethPrivKey = c.String("ethereum-privkey")
	} else {
		ethPrivKey = defaultPrivKeyBob
	}

	bob, err := bob.NewBob(judecoinEndpoint, ethPrivKey)
	if err != nil {
		return err
	}

	fmt.Println("instantiated Bob session", bob)

	var bootnodes []string
	if c.String("bootnodes") != "" {
		bootnodes = strings.Split(c.String("bootnodes"), ",")
	}

	host, err := net.NewHost(defaultBobPort, "ETH", defaultBobLibp2pKey, bootnodes)
	if err != nil {
		return err
	}

	n := &node{
		bob:  bob,
		host: host,
		done: make(chan struct{}),
	}

	return n.doProtocolBob()
}
