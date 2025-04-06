package trader

import (
	"bytes"
	"github.com/nitwhiz/pokemon-red-trade/pkg/pokemon"
	"log"
)

type StageUntilFunc func([]uint8) bool
type StageUpdateFunc func(s *Stage, t *Trader)

type StageUntil struct {
	check        StageUntilFunc
	nextCallback func() *Stage
}

type Stage struct {
	name           string
	updateCallback StageUpdateFunc
	buf            *Buffer
	until          []StageUntil
}

func NewStage(name string, update StageUpdateFunc) *Stage {
	return &Stage{
		name:           name,
		updateCallback: update,
		buf:            NewBuffer(),
		until:          []StageUntil{},
	}
}

func (s *Stage) String() string {
	return s.name
}

func (s *Stage) consumeByte(t *Trader) (b uint8) {
	b = t.serial.Read()
	s.buf.Write(b)
	return
}

func (s *Stage) Update(t *Trader) *Stage {
	for _, n := range s.until {
		if n.check != nil && n.check(s.buf.Bytes()) {
			return n.nextCallback()
		}
	}

	s.updateCallback(s, t)

	return s
}

func EchoDebugStage() *Stage {
	return NewStage("Echo Debug", func(s *Stage, t *Trader) {
		t.serial.Write(s.consumeByte(t))
	})
}

func EchoUntilSerialCompoundStage(name string, compound SerialCompound, next func() *Stage) *Stage {
	s := NewStage(name, func(s *Stage, t *Trader) {
		t.serial.Write(s.consumeByte(t))
	})

	s.until = append(s.until, StageUntil{
		check: func(bs []uint8) bool {
			return bytes.HasSuffix(bs, compound)
		},
		nextCallback: next,
	})

	return s
}

func SeedStage() *Stage {
	return EchoUntilSerialCompoundStage(
		"Seed",
		SerialStartTradeBlock,
		func() *Stage {
			return ExchangeTradeBlockStage()
		},
	)
}

func ConnectStage() *Stage {
	return EchoUntilSerialCompoundStage(
		"Connect",
		SerialStartSeed,
		func() *Stage {
			return SeedStage()
		},
	)
}

func InitialStage() *Stage {
	s := NewStage("Init", func(s *Stage, t *Trader) {
		t.serial.Write(SerialFollow)
		s.consumeByte(t)
		t.serial.Write(SerialFollow)
	})

	s.until = append(s.until, StageUntil{
		check: func(bs []uint8) bool {
			return len(bs) > 0
		},
		nextCallback: func() *Stage {
			return ConnectStage()
		},
	})

	return s
}

func WaitForAcceptStage() *Stage {
	s := NewStage("Wait For Deal", func(s *Stage, t *Trader) {
		s.consumeByte(t)
		t.serial.Write(SerialDealAccept)
	})

	s.until = append(s.until, StageUntil{
		check: func(bs []uint8) bool {
			return bytes.HasSuffix(bs, SerialReject)
		},
		nextCallback: func() *Stage {
			return SelectTradeStage()
		},
	})

	s.until = append(s.until, StageUntil{
		check: func(bs []uint8) bool {
			return bytes.HasSuffix(bs, SerialAccept)
		},
		nextCallback: func() *Stage {
			return ConnectStage()
		},
	})

	return s
}

func SelectTradeStage() *Stage {
	cancel := false

	s := NewStage("Select Trade", func(s *Stage, t *Trader) {
		b := s.consumeByte(t)

		if b == SerialCancel {
			cancel = true
		} else if b >= SerialSelectFirstPokemon {
			t.serial.Write(SerialSelectFirstPokemon)
		} else {
			t.serial.Write(0x00)
		}
	})

	s.until = append(s.until, StageUntil{
		check: func(bs []uint8) bool {
			return cancel
		},
		nextCallback: func() *Stage {
			return ConnectStage()
		},
	})

	s.until = append(s.until, StageUntil{
		check: func(bs []uint8) bool {
			bslen := len(bs)

			if bslen < 2 {
				return false
			}

			return bs[bslen-2] > SerialSelectFirstPokemon && bs[bslen-2] < (SerialSelectLastPokemon+1) && bs[bslen-1] == 0x00
		},
		nextCallback: func() *Stage {
			return WaitForAcceptStage()
		},
	})

	return s
}

func ExchangePatchListStage(patchList *pokemon.PatchListData) *Stage {
	plDataPtr := 0
	plDataLen := len(*patchList)

	s := NewStage("Patch List", func(s *Stage, t *Trader) {
		s.consumeByte(t)
		t.serial.Write((*patchList)[plDataPtr])
		plDataPtr++
	})

	s.until = append(s.until, StageUntil{
		check: func(bs []uint8) bool {
			return plDataPtr == plDataLen
		},
		nextCallback: func() *Stage {
			return SelectTradeStage()
		},
	})

	return s
}

func ExchangeTradeBlockStage() *Stage {
	tradeBlock := pokemon.TradeBlock{
		TrainerName: [11]uint8{0x80},
		PartySize:   1,
		PartyMembers: [7]uint8{
			0x85,
			0xFF,
		},
		Party: [6]pokemon.PartyData{
			{
				Index:             0x85,
				HP:                4,
				Level:             5,
				StatusCondition:   pokemon.StatusNone,
				Type1:             pokemon.TypeWater,
				Type2:             pokemon.TypeWater,
				CatchRate:         255,
				Moves:             [4]uint8{},
				OriginalTrainerID: 1337,
				Experience: [3]uint8{
					0, 0, 0xFF,
				},
				EffortValues:     pokemon.EffortValues{},
				IndividualValues: 0,
				MovesPowerPoints: [4]uint8{},
				Level2:           5,
				Stats: pokemon.Stats{
					HP:      10,
					Attack:  5,
					Defense: 5,
					Speed:   5,
					Special: 5,
				},
			},
		},
		OriginalTrainerNames: [6]pokemon.Name{
			{0x80},
		},
		Nicknames: [6]pokemon.Name{
			{0x80},
		},
	}

	tradeBlockData, patchList, err := tradeBlock.MarshalPatched()

	if err != nil {
		log.Println(err)
		return nil
	}

	tbDataPtr := 0
	tbDataLen := len(tradeBlockData)

	s := NewStage("Trade Block", func(s *Stage, t *Trader) {
		s.consumeByte(t)
		t.serial.Write(tradeBlockData[tbDataPtr])
		tbDataPtr++
	})

	s.until = append(s.until, StageUntil{
		check: func(bs []uint8) bool {
			return tbDataPtr == tbDataLen
		},
		nextCallback: func() *Stage {
			return EchoUntilSerialCompoundStage(
				"Finalize Trade Block",
				SerialStartPatchList,
				func() *Stage {
					return ExchangePatchListStage(patchList)
				},
			)
		},
	})

	return s
}
