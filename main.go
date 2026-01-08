package main

import (
	"log"
	"time"

	lcd1602 "github.com/realfatcat/lcd1602/pkg/lcd"
)

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
	for {
		if err := lcd.Print(time.Now().Format(time.TimeOnly), 0, 0); err != nil {
			log.Fatal(err)
		}
		time.Sleep(1 * time.Second)
	}
}
