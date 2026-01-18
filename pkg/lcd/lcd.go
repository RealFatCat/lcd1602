// Yet another lib for LCD1602 I2C in 4-bit mode.
// It is more descriptive in code, and do not rely on external libs.

// Most of I2C modules for LCD uses PCF8574-like circuits.
// The pins of such PCF8574-like circuits are connected to LCD1602 like so:
// - P0: RS - Register selection pin, that toggles between instruction and data register (0: instruction (or command) mode, 1: data mode)
// - P1: RW - Read/Write selection (0: Write mode, 1: Read mode)
// - P2: E - Enable operations, (0: operations not allowed, 1: operations allowed)
// - P3: BT - It looks like P3 is not directly connected to LCD, but it drives LEDs of LCD, according to schematic (0: LEDs off, 1: LEDs on)
// - P4: D4 - D4-D7 are data pins
// - P5: D4
// - P6: D6
// - P7: D7
//
// So, the lower order bits are the "technical" ones: we set RS, RW, E, BT through them;
// and the higher order bits are the real data, that we pass to or get from LCD.
// That is why, we must run LCD1602 in 4-bit mode.

package lcd

import (
	"fmt"
	"time"

	"golang.org/x/exp/io/i2c"
)

const (
	DefaultDevice  = "/dev/i2c-1"
	DefaultAddress = 0x27
)

const (
	registerCommand = 0x0
	registerData    = 0x1
	// readMode        = 0x2 // useless in most cases, due to PCF8574-like circuits
	writeMode    = 0x0
	enableBit    = 0x4
	backlightOn  = 0x08
	backlightOff = 0x00
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

	LCD_CGRAM_ADDR_BASE = 0x40 // 0b01000000
	LCD_DDRAM_ADDR_BASE = 0x80 // 0b10000000
)

// LCD represents an LCD 1602 display connected via I2C.
type LCD struct {
	i2c       *i2c.Device
	backlight byte
	cols      int
	rows      int
}

// New creates a new LCD1602 instance.
func New(bus string, address int, cols, rows int, isBacklightOn bool) (*LCD, error) {
	i2cDevice, err := i2c.Open(&i2c.Devfs{Dev: bus}, address)
	if err != nil {
		return nil, err
	}
	lcd := &LCD{
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
func (lcd *LCD) Close() error {
	return lcd.i2c.Close()
}

// Init initializes the LCD display in 4-bit mode.
// See Figure24 on 46 page of HD44780 datasheet.
func (lcd *LCD) init() error {
	// Initial wait of voltage HIGH on device, probably useless in our case, but respect the docs.
	time.Sleep(50 * time.Millisecond)

	// Initialization commmands.
	// Waiting timings are trying to respect the docs.
	// Send straight to P4 and P5 data pins, so send 0x30 (0b00110000).
	if err := lcd.busWrite(0x30); err != nil { // 1st
		return err
	}
	time.Sleep(5 * time.Millisecond)

	if err := lcd.busWrite(0x30); err != nil { // 2nd
		return err
	}
	time.Sleep(100 * time.Microsecond)

	if err := lcd.busWrite(0x30); err != nil { // 3rd
		return err
	}

	// Switch from 8-bit to 4-bit mode by sending 0x2.
	// This configures the LCD to operate in 4-bit interface mode, sending high nibble first.
	// Send data straight to P5 data pin so, 0x20 (0b00100000)
	if err := lcd.busWrite(0x20); err != nil {
		return err
	}

	// Initial configuration of LCD.
	// 4-bit mode, 2 rows, 5x8 dots.
	// TODO: Add customization, as only on LCD_FUNC_SET we can set this initial parameters of display.
	if err := lcd.sendCommand(LCD_FUNC_SET | LCD_DL_4BIT | LCD_N_TWOLINE | LCD_F_5X8DOT); err != nil {
		return err
	}

	// Display on, cursor off, blink off.
	if err := lcd.sendCommand(LCD_DISPLAY_CTRL | LCD_DISPLAY_ON | LCD_DISPLAY_CURSOR_OFF | LCD_DISPLAY_BLINK_OFF); err != nil {
		return err
	}

	// Entry mode set: cursor moves right, no display shift.
	if err := lcd.sendCommand(LCD_ENTRY_MODE_SET | LCD_ENTRY_MODE_ID_INCR | LCD_ENTRY_MODE_SHIFT_DISABLE); err != nil {
		return err
	}

	// Clear display.
	if err := lcd.Clear(); err != nil {
		return err
	}

	return nil
}

// Clear clears the display.
func (lcd *LCD) Clear() error {
	if err := lcd.sendCommand(LCD_CLEAR); err != nil {
		return err
	}
	// According to docs, it can take a long time to clear display.
	time.Sleep(2 * time.Millisecond)
	return nil
}

// Home moves cursor to home position.
func (lcd *LCD) Home() error {
	if err := lcd.sendCommand(LCD_HOME); err != nil {
		return err
	}
	// According to docs, it can take a long time to return to home.
	time.Sleep(2 * time.Millisecond)
	return nil
}

// UploadCustomChar uploads custom character to location. Possible values for location: 0-7.
// To print this character on LCD, use PrintRaw() or WriteRaw() methods and use location as raw argument.
func (lcd *LCD) UploadCustomChar(location byte, char [8]byte) error {
	location &= 0x7
	data := (LCD_CGRAM_ADDR_BASE | (location << 3))
	if err := lcd.sendCommand(data); err != nil {
		return err
	}

	for i := range char {
		if err := lcd.sendData(char[i]); err != nil {
			return err
		}
	}
	return nil
}

// SetCursor sets cursor position (row: 0-3, col: 0-lcd.cols).
func (lcd *LCD) SetCursor(row, col int) error {
	if col > lcd.cols {
		return fmt.Errorf("invalid col: %d, valid values 0-15", col)
	}

	var addr byte
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

	addr += byte(col)
	return lcd.sendCommand(addr)
}

// Print prints text to the display, starting from specified row and column.
func (lcd *LCD) Print(text string, row, col int) error {
	if err := lcd.SetCursor(row, col); err != nil {
		return err
	}
	return lcd.Write(text)
}

// PrintRAW prints on character by raw address in specified row and column.
func (lcd *LCD) PrintRAW(raw byte, row, col int) error {
	if err := lcd.SetCursor(row, col); err != nil {
		return err
	}
	return lcd.WriteRAW(raw)
}

// WriteRAW prints text to the display, starting from current cursor position.
func (lcd *LCD) Write(text string) error {
	for _, char := range text {
		if err := lcd.sendData(byte(char)); err != nil {
			return err
		}
	}
	return nil
}

// WriteRAW prints one character by raw address in current cursor position.
func (lcd *LCD) WriteRAW(raw byte) error {
	if err := lcd.sendData(raw); err != nil {
		return err
	}
	return nil
}

// EnableBacklight enables LED backlighting.
func (lcd *LCD) EnableBacklight() error {
	lcd.backlight = backlightOn
	return lcd.busWrite(lcd.backlight)
}

// DisableBacklight disables LED backlighting.
func (lcd *LCD) DisableBacklight() error {
	lcd.backlight = backlightOff
	return lcd.busWrite(lcd.backlight)
}

// ToggleBacklight flips LED backlighting. If it was on: turns off; if it was off: turns on.
func (lcd *LCD) ToggleBacklight() error {
	if lcd.backlight == backlightOff {
		return lcd.EnableBacklight()
	}
	return lcd.DisableBacklight()
}

// sendCommand sends a command to the LCD.
func (lcd *LCD) sendCommand(command byte) error {
	return lcd.send(command, registerCommand)
}

// sendData sends data to the LCD.
func (lcd *LCD) sendData(data byte) error {
	return lcd.send(data, registerData)
}

// send sends a byte to the LCD (4-bit mode).
func (lcd *LCD) send(value byte, rs byte) error {
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
func (lcd *LCD) writeByte(value byte, rs byte) error {
	// Prepare I2C data.
	// Start filling data with technical bits (P0-P3).
	data := rs | writeMode | enableBit | lcd.backlight

	// Set data bits (P4-P7).
	data |= (value << 4)

	// Send to I2C device.
	if err := lcd.busWrite(data); err != nil {
		return err
	}

	// Toggle enable bit to latch data.
	data &= ^(byte(enableBit))
	if err := lcd.busWrite(data); err != nil {
		return err
	}

	// Small delay for timing.
	time.Sleep(50 * time.Microsecond)
	return nil
}

func (lcd *LCD) busWrite(data byte) error {
	if err := lcd.i2c.Write([]byte{data}); err != nil {
		return fmt.Errorf("I2C write error: %v", err)
	}
	return nil
}
