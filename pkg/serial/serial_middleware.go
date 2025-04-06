package serial

import (
	"fmt"
	"log"
	"os"
)

type middlewareFunc func(uint8) uint8

func AddLoggerMiddleware(c *Client, logFile string) error {
	baseName := logFile + "_" + fmt.Sprintf("%02d", c.id)

	readFile, err := os.Create(baseName + "_read.dat")

	if err != nil {
		return err
	}

	writeFile, err := os.Create(baseName + "_write.dat")

	if err != nil {
		return err
	}

	c.AddReadMiddleware(func(b uint8) uint8 {
		if _, err := readFile.Write([]uint8{b}); err != nil {
			log.Println(err)
		}

		return b
	})

	c.AddWriteMiddleware(func(b uint8) uint8 {
		if _, err := writeFile.Write([]uint8{b}); err != nil {
			log.Println(err)
		}

		return b
	})

	return nil
}
