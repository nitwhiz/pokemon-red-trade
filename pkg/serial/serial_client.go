package serial

import (
	"log"
	"net"
	"sync"
)

type Client struct {
	id               uint64
	conn             net.Conn
	readBuf          []uint8
	writeBuf         []uint8
	aliveChan        chan struct{}
	dead             bool
	writeMiddlewares []middlewareFunc
	readMiddlewares  []middlewareFunc
	mu               *sync.RWMutex
}

func NewClient(id uint64, aliveChan chan struct{}, conn net.Conn) *Client {
	return &Client{
		id:               id,
		conn:             conn,
		readBuf:          make([]uint8, 1),
		writeBuf:         make([]uint8, 1),
		aliveChan:        aliveChan,
		dead:             false,
		writeMiddlewares: []middlewareFunc{},
		readMiddlewares:  []middlewareFunc{},
		mu:               &sync.RWMutex{},
	}
}

func (c *Client) ID() uint64 {
	return c.id
}

func (c *Client) Alive() bool {
	return !c.dead
}

func (c *Client) AddWriteMiddleware(m ...middlewareFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.writeMiddlewares = append(c.writeMiddlewares, m...)
}

func (c *Client) AddReadMiddleware(m ...middlewareFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.readMiddlewares = append(c.readMiddlewares, m...)
}

func (c *Client) die() {
	if c.dead {
		return
	}

	c.dead = true

	close(c.aliveChan)
}

func (c *Client) Close() error {
	log.Printf("closing client %d ...", c.id)

	err := c.conn.Close()

	log.Printf("client %d closed.", c.id)

	return err
}

func (c *Client) Read() uint8 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.dead {
		log.Printf("client %d is reading from dead connection.", c.id)
		return NoData
	}

	if _, err := c.conn.Read(c.readBuf); err != nil {
		log.Printf("client %d Read(): %s", c.id, err)

		c.die()
		return NoData
	}

	b := c.readBuf[0]

	for _, m := range c.readMiddlewares {
		b = m(b)
	}

	return b
}

func (c *Client) Write(b uint8) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.dead {
		log.Printf("client %d is writing to dead connection.", c.id)
		return
	}

	for _, m := range c.writeMiddlewares {
		b = m(b)
	}

	c.writeBuf[0] = b

	if _, err := c.conn.Write(c.writeBuf); err != nil {
		log.Printf("client %d Write(): %s", c.id, err)
		c.die()
	}
}
