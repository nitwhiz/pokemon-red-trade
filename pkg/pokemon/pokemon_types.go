package pokemon

import (
	"bytes"
	"encoding/binary"
	"strings"
)

type StatusCondition uint8

const (
	StatusNone      = StatusCondition(0x00)
	StatusAsleep    = StatusCondition(0x04)
	StatusPoisoned  = StatusCondition(0x08)
	StatusBurned    = StatusCondition(0x10)
	StatusFrozen    = StatusCondition(0x20)
	StatusParalyzed = StatusCondition(0x40)
)

type SpeciesType uint8

const (
	TypeNormal   = SpeciesType(0x00)
	TypeFighting = SpeciesType(0x01)
	TypeFlying   = SpeciesType(0x02)
	TypePoison   = SpeciesType(0x03)
	TypeGround   = SpeciesType(0x04)
	TypeRock     = SpeciesType(0x05)
	TypeBird     = SpeciesType(0x06)
	TypeBug      = SpeciesType(0x07)
	TypeGhost    = SpeciesType(0x08)
	TypeFire     = SpeciesType(0x14)
	TypeWater    = SpeciesType(0x15)
	TypeGrass    = SpeciesType(0x16)
	TypeElectric = SpeciesType(0x17)
	TypePsychic  = SpeciesType(0x18)
	TypeIce      = SpeciesType(0x19)
	TypeDragon   = SpeciesType(0x1A)
)

type Stats struct {
	HP      uint16
	Attack  uint16
	Defense uint16
	Speed   uint16
	Special uint16
}

type EffortValues struct {
	Stats
}

type Name [11]uint8

func (n *Name) String() string {
	res := strings.Builder{}

	for _, b := range n {
		if b == 0x50 {
			break
		}

		res.WriteString(DecodeText(b))
	}

	return res.String()
}

type PartyData struct {
	Index             uint8
	HP                uint16
	Level             uint8
	StatusCondition   StatusCondition
	Type1             SpeciesType
	Type2             SpeciesType
	CatchRate         uint8
	Moves             [4]uint8
	OriginalTrainerID uint16
	Experience        [3]uint8
	EffortValues      EffortValues
	IndividualValues  uint16
	MovesPowerPoints  [4]uint8
	// Level is repeated here
	Level2 uint8
	Stats  Stats
}

type TradeBlock struct {
	TrainerName [11]uint8
	PartySize   uint8
	// PartyMembers is terminated with 0xFF
	PartyMembers         [7]uint8
	Party                [6]PartyData
	OriginalTrainerNames [6]Name
	Nicknames            [6]Name
}

type PatchListData [190]uint8

func NewPatchListData() *PatchListData {
	return &PatchListData{0xFF, 0xFF}
}

type PatchIndex struct {
	index int
	next  *PatchIndex
}

func (p PatchListData) Parse() *PatchIndex {
	var root *PatchIndex
	var current *PatchIndex

	base := 0x00

	for _, relativePatchIndex := range p {
		if relativePatchIndex == 0xFF {
			if base == 0x00 {
				base = 0xFC
				continue
			} else {
				// end of patch list
				break
			}
		}

		pi := base + int(relativePatchIndex) - 1

		if root == nil {
			root = &PatchIndex{
				index: pi,
			}

			current = root
		} else {
			next := &PatchIndex{
				index: pi,
			}

			current.next = next
			current = next
		}
	}

	return root
}

func (p *PatchIndex) Marshal() *PatchListData {
	res := NewPatchListData()
	maxResLen := uint8(len(res))

	current := p
	base := 0x00

	for idx := uint8(0); idx < maxResLen; idx++ {
		if current == nil {
			res[idx] = 0xFF
			idx++

			if base == 0x00 {
				// still in part zero, write the final terminator, too
				res[idx] = 0xFF
			}

			break
		}

		if base == 0x00 && current.index >= 0xFC {
			base = 0xFC
			res[idx] = 0xFF
			idx++
		}

		res[idx] = uint8(current.index - base)
		current = current.next
	}

	return res
}

func PatchTradeBlock(tb []uint8) *PatchIndex {
	var root *PatchIndex
	var current *PatchIndex

	for idx, v := range tb {
		if v == 0xFE {
			tb[idx] = 0xFF

			patchedIdx := idx + 1

			if root == nil {
				root = &PatchIndex{
					index: patchedIdx,
				}

				current = root
			} else {
				next := &PatchIndex{
					index: patchedIdx,
				}

				current.next = next
				current = next
			}
		}
	}

	return root
}

func (t *TradeBlock) MarshalPatched() ([]uint8, *PatchListData, error) {
	data, err := Marshal(t)

	if err != nil {
		return nil, nil, err
	}

	var pld *PatchListData

	pi := PatchTradeBlock(data)

	if pi == nil {
		pld = NewPatchListData()
	} else {
		pld = pi.Marshal()
	}

	return data, pld, nil
}

func Unmarshal(data []uint8, v any) error {
	return binary.Read(bytes.NewReader(data), binary.BigEndian, v)
}

func Marshal(v any) ([]uint8, error) {
	buf := bytes.Buffer{}

	if err := binary.Write(&buf, binary.BigEndian, v); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
