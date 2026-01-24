// Yet another lib for LCD1602 I2C in 4-bit mode.

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

	Font5x8  = lcdF5x8Dot
	Font5x10 = lcdF5x10Dot
)

const (
	lcdRegisterCommand = 0x0
	lcdRegisterData    = 0x1
	// lcdReadMode        = 0x2 // useless in most cases, due to PCF8574-like circuits
	lcdWriteMode    = 0x0
	lcdEnableBit    = 0x4
	lcdBacklightOn  = 0x08
	lcdBacklightOff = 0x00

	lcdClear = 0x01
	lcdHome  = 0x02

	lcdEntryModeSet          = 0x04 // 0b100
	lcdEntryModeIDIncr       = 0x02 // 0b010
	lcdEntryModeIDDecr       = 0x00
	lcdEntryModeShiftEnable  = 0x01 // 0b001
	lcdEntryModeShiftDisable = 0x00

	lcdFuncSet = 0x20 // 0b100000
	lcdDL4Bit  = 0x00
	// lcdDL8Bit   = 0x10 // 0b010000 // useless in our case
	lcdNOneLine = 0x00
	lcdNTwoLine = 0x08 // 0b001000
	lcdF5x8Dot  = 0x00
	lcdF5x10Dot = 0x04 // 0b000100

	lcdDisplayCtrl      = 0x08 // 0b1000
	lcdDisplayOn        = 0x04 // 0b0100
	lcdDisplayOff       = 0x00
	lcdDisplayCursorOn  = 0x02 // 0b0010
	lcdDisplayCursorOff = 0x00
	lcdDisplayBlinkOn   = 0x01 // 0b0001
	lcdDisplayBlinkOff  = 0x00

	lcdCGRAMAddrBase = 0x40 // 0b01000000
	lcdDDRAMAddrBase = 0x80 // 0b10000000
)

// LCD represents an LCD 1602 display connected via I2C.
type LCD struct {
	i2c       *i2c.Device
	backlight byte
	cols      int
	rows      int
	font      byte

	displayState byte
}

// New creates and initializes a new LCD1602 display instance connected via I2C.

// Parameters:
//   - bus: The I2C bus device path (e.g., "/dev/i2c-1" or use DefaultDevice constant).
//   - address: The I2C device address (typically 0x27 for PCF8574-based modules, or use DefaultAddress constant).
//   - cols: Number of columns (characters per line). Valid values: 16, 20.
//   - rows: Number of display rows. Valid values: 1, 2, 4.
//   - font: Font size specification. Must be one of: Font5x8 (standard 5x8 pixel font) or Font5x10 (5x10 pixel font).
//   - isBacklightOn: Whether to enable the backlight LED immediately upon initialization.
//
// Valid combinations:
//   - 16 columns × 1 row (16x1 display)
//   - 16 columns × 2 rows (16x2 display, most commonm, this package has been tested on that type of LCD)
//   - 16 columns × 4 rows (16x4 display)
//   - 20 columns × 1 row (20x1 display)
//   - 20 columns × 2 rows (20x2 display)
//   - 20 columns × 4 rows (20x4 display)
//
// Note: The 5x10 font is typically only available for single-line displays (rows=1).
// For multi-line displays, Font5x8 should be used.
func New(bus string, address int, cols int, rows int, font byte, isBacklightOn bool) (*LCD, error) {
	if err := validateInputs(cols, rows, font); err != nil {
		return nil, fmt.Errorf("invalid inputs: %w", err)
	}

	i2cDevice, err := i2c.Open(&i2c.Devfs{Dev: bus}, address)
	if err != nil {
		return nil, err
	}
	lcd := &LCD{
		i2c:  i2cDevice,
		cols: cols,
		rows: rows,
		font: font,
		// initial display state
		displayState: lcdDisplayOn | lcdDisplayCursorOff | lcdDisplayBlinkOff,
	}
	if err := lcd.init(); err != nil {
		return nil, err
	}

	if isBacklightOn {
		lcd.backlight = lcdBacklightOn
	}
	return lcd, nil
}

// validateInputs validates the input parameters for New function.
// It checks that cols is 16 or 20, rows is 1, 2, or 4, font is Font5x8 or Font5x10,
// and that Font5x10 is only used with single-line displays (rows=1).
func validateInputs(cols, rows int, font byte) error {
	if cols != 16 && cols != 20 {
		return fmt.Errorf("invalid cols value: %d, must be 16 or 20", cols)
	}

	if rows != 1 && rows != 2 && rows != 4 {
		return fmt.Errorf("invalid rows value: %d, must be 1, 2, or 4", rows)
	}

	if font != Font5x8 && font != Font5x10 {
		return fmt.Errorf("invalid font value: %d, must be Font5x8 (%d) or Font5x10 (%d)", font, Font5x8, Font5x10)
	}

	if font == Font5x10 && rows != 1 {
		return fmt.Errorf("Font5x10 is only available for single-line displays (rows=1), got rows=%d", rows)
	}
	return nil
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
	if err := lcd.sendCommand(0x03); err != nil { // 1st
		return err
	}
	time.Sleep(5 * time.Millisecond)

	if err := lcd.sendCommand(0x03); err != nil { // 2nd
		return err
	}
	time.Sleep(100 * time.Microsecond)

	if err := lcd.sendCommand(0x03); err != nil { // 3rd
		return err
	}

	// Switch from 8-bit to 4-bit mode by sending 0x2.
	// This configures the LCD to operate in 4-bit interface mode, sending high nibble first.
	if err := lcd.sendCommand(0x02); err != nil {
		return err
	}

	// Initial configuration of LCD.
	lineMode := byte(lcdNOneLine)
	if lcd.rows > 1 {
		lineMode = lcdNTwoLine
	}
	if err := lcd.sendCommand(lcdFuncSet | lcdDL4Bit | lineMode | lcd.font); err != nil {
		return err
	}

	// Display on, cursor off, blink off.
	if err := lcd.sendCommand(lcdDisplayCtrl | lcd.displayState); err != nil {
		return err
	}

	// Entry mode set: cursor moves right, no display shift.
	if err := lcd.sendCommand(lcdEntryModeSet | lcdEntryModeIDIncr | lcdEntryModeShiftDisable); err != nil {
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
	if err := lcd.sendCommand(lcdClear); err != nil {
		return err
	}
	// According to docs, it can take a long time to clear display.
	time.Sleep(2 * time.Millisecond)
	return nil
}

// Home moves cursor to home position.
func (lcd *LCD) Home() error {
	if err := lcd.sendCommand(lcdHome); err != nil {
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
	data := (lcdCGRAMAddrBase | (location << 3))
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

// SetCursor sets cursor position. Numeration is from 0 to your module number of rows/columns - 1.
// For example, for module with 16 columns and 2 rows, valid values are:
//   - columns: [0-15]
//   - rows: [0-1]
func (lcd *LCD) SetCursor(row, col int) error {
	if (col < 0) || (col >= lcd.cols) {
		return fmt.Errorf("invalid col: %d", col)
	}

	var addr byte
	switch row {
	case 0:
		addr = lcdDDRAMAddrBase
	case 1:
		addr = lcdDDRAMAddrBase + 0x40
	case 2:
		addr = lcdDDRAMAddrBase + 0x14
	case 3:
		addr = lcdDDRAMAddrBase + 0x54
	default:
		return fmt.Errorf("invalid row: %d", row)
	}

	addr += byte(col)
	return lcd.sendCommand(addr)
}

// Print prints text to the display, starting from specified row and column.
// Check SetCursor documentation for valid row, column values.
func (lcd *LCD) Print(text string, row, col int) error {
	if err := lcd.SetCursor(row, col); err != nil {
		return err
	}
	return lcd.Write(text)
}

// PrintRAW prints on character by raw address in specified row and column.
// See table 4 on pages 17-18, depending on your module.
// Check SetCursor documentation for valid row, column values.
func (lcd *LCD) PrintRAW(raw byte, row, col int) error {
	if err := lcd.SetCursor(row, col); err != nil {
		return err
	}
	return lcd.WriteRAW(raw)
}

// Write prints text to the display, starting from current cursor position.
func (lcd *LCD) Write(text string) error {
	for _, char := range text {
		if err := lcd.sendData(byte(char)); err != nil {
			return err
		}
	}
	return nil
}

// WriteRAW prints one character by raw address in current cursor position.
// See table 4 on pages 17-18, depending on your module.
func (lcd *LCD) WriteRAW(raw byte) error {
	if err := lcd.sendData(raw); err != nil {
		return err
	}
	return nil
}

// EnableBacklight enables LED backlighting.
func (lcd *LCD) EnableBacklight() error {
	lcd.backlight = lcdBacklightOn
	return lcd.busWrite(lcd.backlight)
}

// DisableBacklight disables LED backlighting.
func (lcd *LCD) DisableBacklight() error {
	lcd.backlight = lcdBacklightOff
	return lcd.busWrite(lcd.backlight)
}

// ToggleBacklight flips LED backlighting. If it was on: turns off; if it was off: turns on.
func (lcd *LCD) ToggleBacklight() error {
	if lcd.backlight == lcdBacklightOff {
		return lcd.EnableBacklight()
	}
	return lcd.DisableBacklight()
}

// DisplayOn turns on the LCD display.
// The display will show characters but the cursor and blink settings remain unchanged.
func (lcd *LCD) DisplayOn() error {
	lcd.displayState |= lcdDisplayOn
	return lcd.sendCommand(lcdDisplayCtrl | lcd.displayState)
}

// DisplayOff turns off the LCD display.
// The display will be blank but the cursor position and content are preserved.
// Use DisplayOn() to turn the display back on.
// Data on display is not cleared when display is off.
func (lcd *LCD) DisplayOff() error {
	lcd.displayState &= ^(byte(lcdDisplayOn))
	return lcd.sendCommand(lcdDisplayCtrl | lcd.displayState)
}

// ToggleDisplay toggles the display state.
// If the display is on, it turns it off; if it's off, it turns it on.
func (lcd *LCD) ToggleDisplay() error {
	if (lcd.displayState & lcdDisplayOn) == lcdDisplayOn {
		return lcd.DisplayOff()
	}
	return lcd.DisplayOn()
}

// CursorOn makes the cursor visible on the display.
// The cursor appears as an underscore at the current cursor position.
func (lcd *LCD) CursorOn() error {
	lcd.displayState |= lcdDisplayCursorOn
	return lcd.sendCommand(lcdDisplayCtrl | lcd.displayState)
}

// CursorOff hides the cursor on the display.
// The cursor position is still tracked, but it won't be visible.
func (lcd *LCD) CursorOff() error {
	lcd.displayState &= ^(byte(lcdDisplayCursorOn))
	return lcd.sendCommand(lcdDisplayCtrl | lcd.displayState)
}

// ToggleCursor toggles the cursor visibility.
// If the cursor is visible, it hides it; if it's hidden, it makes it visible.
func (lcd *LCD) ToggleCursor() error {
	if (lcd.displayState & lcdDisplayCursorOn) == lcdDisplayCursorOn {
		return lcd.CursorOff()
	}
	return lcd.CursorOn()
}

// BlinkOn enables blinking of the character at the cursor position.
// The character at the cursor will blink on and off.
func (lcd *LCD) BlinkOn() error {
	lcd.displayState |= lcdDisplayBlinkOn
	return lcd.sendCommand(lcdDisplayCtrl | lcd.displayState)
}

// BlinkOff disables blinking of the character at the cursor position.
// The character will remain static at the cursor position.
func (lcd *LCD) BlinkOff() error {
	lcd.displayState &= ^(byte(lcdDisplayBlinkOn))
	return lcd.sendCommand(lcdDisplayCtrl | lcd.displayState)
}

// ToggleBlink toggles the blink state of the character at the cursor position.
// If blinking is enabled, it disables it; if it's disabled, it enables it.
func (lcd *LCD) ToggleBlink() error {
	if (lcd.displayState & lcdDisplayBlinkOn) == lcdDisplayBlinkOn {
		return lcd.BlinkOff()
	}
	return lcd.BlinkOn()
}

// sendCommand sends a command to the LCD.
func (lcd *LCD) sendCommand(command byte) error {
	return lcd.send(command, lcdRegisterCommand)
}

// sendData sends data to the LCD.
func (lcd *LCD) sendData(data byte) error {
	return lcd.send(data, lcdRegisterData)
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
	data := rs | lcdWriteMode | lcdEnableBit | lcd.backlight

	// Set data bits (P4-P7).
	data |= (value << 4)

	// Send to I2C device.
	if err := lcd.busWrite(data); err != nil {
		return err
	}

	// Toggle enable bit to latch data.
	data &= ^(byte(lcdEnableBit))
	if err := lcd.busWrite(data); err != nil {
		return err
	}

	// Small delay for timing.
	time.Sleep(50 * time.Microsecond)
	return nil
}

// busWrite writes a single byte to the I2C bus.
func (lcd *LCD) busWrite(data byte) error {
	if err := lcd.i2c.Write([]byte{data}); err != nil {
		return fmt.Errorf("I2C write error: %v", err)
	}
	return nil
}
