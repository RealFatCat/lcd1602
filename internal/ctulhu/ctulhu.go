package ctulhu

import (
	"log"

	lcd1602 "github.com/realfatcat/lcd1602/pkg/lcd"
)

// custom Cthulhu character
var cthulhu = [8]byte{
	0b01110,
	0b11111,
	0b10101,
	0b11111,
	0b01110,
	0b11111,
	0b10101,
	0b10101,
}

type walkingCthulhu struct {
	direction      int
	turnBackCol    int
	turnForwardCol int
	curCol         int
	curRow         int
	lcd            *lcd1602.LCD
	kthulhu        [8]byte
	charLocation   byte
}

func New(startRow, startCol, turnBackCol, turnForwardCol, direction int, charLocation byte, lcd *lcd1602.LCD) (*walkingCthulhu, error) {
	direct := 1
	if direction < 0 {
		direct = -1
	}

	if err := lcd.UploadCustomChar(charLocation, cthulhu); err != nil {
		return nil, err
	}

	if err := lcd.PrintRAW(charLocation, startRow, startCol); err != nil {
		log.Fatal(err)
	}

	return &walkingCthulhu{
		kthulhu:        cthulhu,
		lcd:            lcd,
		curRow:         startRow,
		curCol:         startCol,
		turnBackCol:    turnBackCol,
		turnForwardCol: turnForwardCol,
		direction:      direct,
		charLocation:   charLocation,
	}, nil
}

func (wk *walkingCthulhu) MakeStep() {
	switch wk.direction {
	case 1:
		if wk.curCol == wk.turnBackCol {
			wk.direction = -1
		}
	case -1:
		if wk.curCol == wk.turnForwardCol {
			wk.direction = 1
		}
	}
	wk.lcd.Print(" ", wk.curRow, wk.curCol)
	wk.curCol += wk.direction
	wk.lcd.PrintRAW(wk.charLocation, wk.curRow, wk.curCol)
}
