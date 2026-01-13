package main

import (
	"log"
	"time"

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

func main() {
	isBacklightOn := true
	cols := 16
	rows := 2
	lcd, err := lcd1602.New(lcd1602.DefaultDevice, lcd1602.DefaultAddress, cols, rows, isBacklightOn)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = lcd.Close() }()

	if err := lcd.Print("  () ()", 0, 7); err != nil {
		log.Fatal(err)
	}
	if err := lcd.Print("=<^ _ ^>=", 1, 7); err != nil {
		log.Fatal(err)
	}

	wk := newKtulhu(
		1, // start row
		0, // start column
		5, // turn back column
		0, // turn forward column
		1, // at first move to the right
		0, // char location: CGRAM location, where to save custom Cthulhu character (0-7)
		lcd,
	)
	for {
		if err := lcd.Print(time.Now().Format(time.TimeOnly), 0, 0); err != nil {
			log.Fatal(err)
		}
		time.Sleep(1 * time.Second)
		wk.MakeStep()
	}
}

type walkingCthulhu struct {
	direction      int
	turnBackCol    int
	turnForwardCol int
	curCol         int
	curRow         int
	lcd            *lcd1602.LCD1602
	kthulhu        [8]byte
	charLocation   byte
}

func newKtulhu(startRow, startCol, turnBackCol, turnForwardCol, direction int, charLocation byte, lcd *lcd1602.LCD1602) *walkingCthulhu {
	direct := 1
	if direction < 0 {
		direct = -1
	}

	if err := lcd.UploadCustomChar(charLocation, cthulhu); err != nil {
		log.Fatal(err)
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
	}
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
