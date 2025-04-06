package trader

type SerialPort interface {
	Read() uint8
	Write(b uint8)
	Alive() bool
	ID() uint64
}

const (
	SerialLead               = uint8(0x01)
	SerialFollow             = uint8(0x02)
	SerialConnected          = uint8(0x60)
	SerialPreamble           = uint8(0xFD)
	SerialSelectFirstPokemon = uint8(0x60)
	SerialSelectLastPokemon  = uint8(0x66)
	SerialDealReject         = uint8(0x61)
	SerialDealAccept         = uint8(0x62)
	SerialCancel             = uint8(0x6F)
)

type SerialCompound []uint8

var (
	SerialStartSeed = SerialCompound{
		SerialPreamble, SerialPreamble, SerialPreamble, SerialPreamble,
		SerialPreamble, SerialPreamble, SerialPreamble, SerialPreamble,
		SerialPreamble, SerialPreamble,
	}
	SerialStartTradeBlock = SerialCompound{
		SerialPreamble, SerialPreamble, SerialPreamble, SerialPreamble,
		SerialPreamble, SerialPreamble, SerialPreamble, SerialPreamble,
		SerialPreamble,
	}
	SerialStartPatchList = SerialCompound{
		SerialPreamble, SerialPreamble, SerialPreamble,
		SerialPreamble, SerialPreamble, SerialPreamble,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	SerialReject = SerialCompound{
		SerialDealReject, 0x00,
	}
	SerialAccept = SerialCompound{
		SerialDealAccept, 0x00,
	}
)
