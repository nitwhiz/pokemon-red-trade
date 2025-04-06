package trader

type Buffer struct {
	buf []byte
}

func NewBuffer() *Buffer {
	return &Buffer{
		buf: make([]byte, 0),
	}
}

func (b *Buffer) Write(byte uint8) {
	b.buf = append(b.buf, byte)
}

func (b *Buffer) Bytes() []uint8 {
	return b.buf
}
