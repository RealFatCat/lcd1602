package main

import (
	"log"
	"time"

	"github.com/realfatcat/lcd1602/internal/ctulhu"
	lcd1602 "github.com/realfatcat/lcd1602/pkg/lcd"
)

func main() {
	isBacklightOn := true
	cols := 16
	rows := 2
	lcd, err := lcd1602.New(lcd1602.DefaultDevice, lcd1602.DefaultAddress, cols, rows, lcd1602.Font5x8, isBacklightOn)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { lcd.Clear(); _ = lcd.Close() }()

	lcd.Print("Hello!", 0, 7)

	for range 7 {
		lcd.DisplayShiftLeft()
		time.Sleep(300 * time.Millisecond)
	}
	time.Sleep(1 * time.Second)

	lcd.Clear()

	if err := lcd.Print("  () ()", 0, 7); err != nil {
		log.Fatal(err)
	}
	if err := lcd.Print("=<^ _ ^>=", 1, 7); err != nil {
		log.Fatal(err)
	}

	wk, err := ctulhu.New(
		1, // start row
		0, // start column
		5, // turn back column
		0, // turn forward column
		1, // at first move to the right
		0, // char location: CGRAM location, where to save custom Cthulhu character (0-7)
		lcd,
	)
	if err != nil {
		log.Fatal(err)
	}
	for {
		if err := lcd.Print(time.Now().Format(time.TimeOnly), 0, 0); err != nil {
			log.Fatal(err)
		}
		time.Sleep(1 * time.Second)
		wk.MakeStep()
	}
}
