// Yet another lib for LCD1602 I2C in 4-bit mode.
// It is more descriptive in code, and do not rely on external libs.
package lcd

import (
	"fmt"
	"time"

	"golang.org/x/exp/io/i2c"
)

const (
	backlightOn = 0x08

	registerCommand = 0x0
	registerData    = 0x1
	enableBit       = byte(0x4)

	DefaultDevice  = "/dev/i2c-1"
	DefaultAddress = 0x27
)

const (
	LCD_CLEAR = 0x01
	LCD_HOME  = 0x02

	LCD_ENTRY_MODE_SET           = 0x04 // 0b100
	LCD_ENTRY_MODE_ID_INCR       = 0x02 // 0b010
	LCD_ENTRY_MODE_ID_DECR       = 0x00
	LCD_ENTRY_MODE_SHIFT_ENABLE  = 0x01 // 0b001
	LCD_ENTRY_MODE_SHIFT_DISABLE = 0x00

	LCD_FUNC_SET  = 0x20 // 0b100000
	LCD_DL_4BIT   = 0x00
	LCD_DL_8BIT   = 0x10 // 0b010000
	LCD_N_ONELINE = 0x00
	LCD_N_TWOLINE = 0x08 // 0b001000
	LCD_F_5X8DOT  = 0x00
	LCD_F_5X10DOT = 0x04 // 0b000100

	LCD_DISPLAY_CTRL       = 0x08 // 0b1000
	LCD_DISPLAY_ON         = 0x04 // 0b0100
	LCD_DISPLAY_OFF        = 0x00
	LCD_DISPLAY_CURSOR_ON  = 0x02 // 0b0010
	LCD_DISPLAY_CURSOR_OFF = 0x00
	LCD_DISPLAY_BLINK_ON   = 0x01 // 0b0001
	LCD_DISPLAY_BLINK_OFF  = 0x00

	LCD_DDRAM_ADDR_BASE = 0x80 // 0b10000000
)

// LCD1602 represents an LCD 1602 display connected via I2C.
type LCD1602 struct {
	i2c       *i2c.Device
	backlight uint8
	cols      int
	rows      int
}

// New creates a new LCD1602 instance.
func New(bus string, address int, cols, rows int, isBacklightOn bool) (*LCD1602, error) {
	i2cDevice, err := i2c.Open(&i2c.Devfs{Dev: bus}, address)
	if err != nil {
		return nil, err
	}
	lcd := &LCD1602{
		i2c:  i2cDevice,
		cols: cols,
		rows: rows,
	}
	if err := lcd.init(); err != nil {
		return nil, err
	}

	if isBacklightOn {
		lcd.backlight = backlightOn
	}
	return lcd, nil
}

// Close device.
func (lcd *LCD1602) Close() error {
	return lcd.i2c.Close()
}

// Init initializes the LCD display in 4-bit mode.
func (lcd *LCD1602) init() error {
	// Initial wait of voltage HIGH on device, probably useless in our case.
	time.Sleep(30 * time.Millisecond)

	// Initialization commmands. See Figure24 on 46 page 4 bit mode.
	// Send 0x3 three times to initialize LCD in 8-bit mode (required even for 4-bit operation).
	// The HD44780 controller starts in 8-bit mode, so we send 0x33 to ensure it's ready for 4-bit mode switch.
	if err := lcd.sendCommand(0x3); err != nil {
		return err
	}
	time.Sleep(5 * time.Millisecond)

	if err := lcd.sendCommand(0x3); err != nil {
		return err
	}
	time.Sleep(100 * time.Microsecond)

	if err := lcd.sendCommand(0x3); err != nil {
		return err
	}

	// Switch from 8-bit to 4-bit mode by sending 0x2
	// This configures the LCD to operate in 4-bit interface mode, sending high nibble first
	if err := lcd.sendCommand(0x2); err != nil {
		return err
	}

	// Configuration of LCD.
	// 4-bit mode, 2 rows, 5x8 dots
	if err := lcd.sendCommand(LCD_FUNC_SET | LCD_DL_4BIT | LCD_N_TWOLINE | LCD_F_5X8DOT); err != nil {
		return err
	}

	// Display on, cursor off, blink off
	if err := lcd.sendCommand(LCD_DISPLAY_CTRL | LCD_DISPLAY_ON | LCD_DISPLAY_CURSOR_OFF | LCD_DISPLAY_BLINK_OFF); err != nil {
		return err
	}

	// Entry mode set: cursor moves right, no display shift
	if err := lcd.sendCommand(LCD_ENTRY_MODE_SET | LCD_ENTRY_MODE_ID_INCR | LCD_ENTRY_MODE_SHIFT_DISABLE); err != nil {
		return err
	}
	if err := lcd.sendCommand(LCD_CLEAR); err != nil { // Clear display
		return err
	}

	return nil
}

// Clear clears the display.
func (lcd *LCD1602) Clear() error {
	if err := lcd.sendCommand(LCD_CLEAR); err != nil {
		return err
	}
	return nil
}

// Home moves cursor to home position (0,0).
func (lcd *LCD1602) Home() error {
	if err := lcd.sendCommand(LCD_HOME); err != nil {
		return err
	}
	time.Sleep(2 * time.Millisecond)
	return nil
}

// setCursor sets cursor position (row: 0-3, col: lcd.col).
func (lcd *LCD1602) setCursor(row, col int) error {
	if col > lcd.cols || col < 0 {
		return fmt.Errorf("invalid col: %d, valid values 0-15", col)
	}

	var addr uint8
	switch row {
	case 0:
		addr = LCD_DDRAM_ADDR_BASE
	case 1:
		addr = LCD_DDRAM_ADDR_BASE + 0x40
	case 2:
		addr = LCD_DDRAM_ADDR_BASE + 0x14
	case 3:
		addr = LCD_DDRAM_ADDR_BASE + 0x54
	default:
		return fmt.Errorf("invalid row: %d, valid values 0-3 ", row)
	}

	addr += uint8(col)
	return lcd.sendCommand(addr)
}

// Print prints text to the display.
func (lcd *LCD1602) Print(text string, row, col int) error {
	if err := lcd.setCursor(row, col); err != nil {
		return err
	}
	for _, char := range text {
		if err := lcd.sendData(uint8(char)); err != nil {
			return err
		}
	}
	return nil
}

// sendCommand sends a command to the LCD.
func (lcd *LCD1602) sendCommand(command uint8) error {
	return lcd.send(command, registerCommand)
}

// sendData sends data to the LCD.
func (lcd *LCD1602) sendData(data uint8) error {
	return lcd.send(data, registerData)
}

// send sends a byte to the LCD (4-bit mode).
func (lcd *LCD1602) send(value uint8, rs uint8) error {
	// Prepare data for I2C communication
	// High nibble
	high := value >> 4
	if err := lcd.writeByte(high, rs); err != nil {
		return err
	}

	// Low nibble
	low := value & 0x0F
	return lcd.writeByte(low, rs)
}

// writeByte writes a byte to the I2C device.
func (lcd *LCD1602) writeByte(value uint8, rs uint8) error {
	// Prepare I2C data (assuming your backpack uses specific pin mapping)

	// Start filling data, use backlight value as starting point.
	data := lcd.backlight

	// Set RS bit (Register Select)
	data |= rs

	// Set Enable bit (usually bit 2)
	data |= enableBit

	// Set data bits (bits 4-7)
	data |= (value << 4)

	// Send to I2C device
	if err := lcd.i2c.Write([]byte{data}); err != nil {
		return fmt.Errorf("I2C write error: %v", err)
	}

	// Toggle enable bit to latch data
	data &= ^(enableBit)
	if err := lcd.i2c.Write([]byte{data}); err != nil {
		return fmt.Errorf("I2C write error: %v", err)
	}

	// Small delay for timing
	time.Sleep(100 * time.Microsecond)
	return nil
}
