package main

import (
	"log"
	"time"

	"github.com/realfatcat/lcd1602"
	"github.com/realfatcat/lcd1602/internal/ctulhu"
)

func main() {
	isBacklightOn := true
	cols := 20
	rows := 4
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

	if err := printCat(lcd); err != nil {
		log.Fatal(err)
	}

	wk, err := ctulhu.New(
		3,  // start row
		0,  // start column
		19, // turn back column
		0,  // turn forward column
		1,  // at first move to the right
		0,  // char location: CGRAM location, where to save custom Cthulhu character (0-7)
		lcd,
	)
	if err != nil {
		log.Fatal(err)
	}
	for {
		if err := printDate(lcd); err != nil {
			log.Fatal(err)
		}
		time.Sleep(1 * time.Second)
		wk.MakeStep()
	}
}

func printDate(lcd *lcd1602.LCD) error {
	now := time.Now()
	h, m, s := now.Clock()
	if h == 21 && m == 0 && s == 0 {
		lcd.DisableBacklight()
	}
	if h == 9 && m == 0 && s == 0 {
		lcd.EnableBacklight()
	}
	if err := lcd.Print(now.Format(time.TimeOnly), 1, 0); err != nil {
		log.Fatal(err)
	}
	// TODO: optimize to not print date every time: need state
	if err := lcd.Print(time.Now().Format(time.DateOnly), 0, 0); err != nil {
		log.Fatal(err)
	}

	return nil
}

func printCat(lcd *lcd1602.LCD) error {
	if err := lcd.Print("  () ()", 0, 10); err != nil {
		return err
	}
	if err := lcd.Print("=<^ _ ^>=", 1, 10); err != nil {
		return err
	}
	if err := lcd.Print("(\") (\")", 2, 10); err != nil {
		return err
	}
	return nil
}
