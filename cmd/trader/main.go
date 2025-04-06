package main

import (
	"context"
	"github.com/nitwhiz/pokemon-red-trade/pkg/serial"
	"github.com/nitwhiz/pokemon-red-trade/pkg/trader"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	s := serial.NewServer(ctx)

	defer cancel()

	log.Println("starting server")

	if err := s.Listen("/tmp/gb-serial.sock"); err != nil {
		log.Fatal(err)
	}

	s.Start()

	interruptChan := make(chan os.Signal)

	signal.Notify(interruptChan, syscall.SIGTERM, syscall.SIGINT)

	log.Println("listening for connections ...")

	for {
		select {
		case <-interruptChan:
			if err := s.Close(); err != nil {
				log.Println(err)
			}

			cancel()

			os.Exit(1)
			return
		case client := <-s.Accept():

			if err := serial.AddLoggerMiddleware(client, "logs/client"); err != nil {
				log.Println(err)
				break
			}

			go trader.NewTrader(client).Start()

			break
		}
	}
}
