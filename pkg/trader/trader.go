package trader

import (
	"log"
	"os"
)

type Trader struct {
	serial SerialPort
	stage  *Stage

	recvLogFile *os.File
	sendLogFile *os.File
}

func NewTrader(serial SerialPort) *Trader {
	return &Trader{
		serial: serial,
		stage:  InitialStage(),
	}
}

func (t *Trader) Start() {
	for {
		nextStage := t.stage.Update(t)

		if !t.serial.Alive() {
			log.Printf("client %d: serial is dead.\n", t.serial.ID())
			break
		}

		if nextStage == t.stage {
			continue
		}

		log.Printf("client %d: %s -> %s\n", t.serial.ID(), t.stage, nextStage)

		if nextStage == nil {
			break
		}

		t.stage = nextStage
	}

	log.Printf("byte client %d.\n", t.serial.ID())
}
