# LCD1602 Go Library

Yet another Go library for controlling LCD1602 displays via I2C in 4-bit mode.

## Features

- **Display Configuration**: Support for various display sizes (16x1, 16x2, 16x4, 20x1, 20x2, 20x4) and font options (5x8 and 5x10)
- **Text Display**: Print text at specific positions or write from current cursor position
- **Cursor Control**: Set cursor position, move to home, and enable/disable/toggle cursor visibility, enable/disable/toggle blinking at cursor position
- **Display Control**: Clear display, turn display on/off
- **Backlight Control**: Enable, disable, or toggle LED backlight
- **Custom Characters**: Upload custom characters (5x8 pixel patterns) to CGRAM locations 0-7
- **Raw Character Access**: Direct access to character codes for special characters and custom patterns

## Demo

Run the included demo to see the library in action:

```bash
go run main.go
```

This initializes the LCD, displays some ASCII art, updates the current time every second, and even wakes up Cthulhu!

To build the demo for your architecture use `make` commands.

View all available build targets:

```bash
make help
```

For example, build for Raspberry Pi model 1:

```bash
make arm6
```

This will build a binary named `lcd-demo-armv6` for ARMv6 architecture. 

## Example: Custom Characters

Create and display custom characters using `UploadCustomChar`.

This code will display two cthulhus nearby.

```go
package main

import (
	"log"
	lcd1602 "github.com/realfatcat/lcd1602/pkg/lcd"
)

func main() {
	// Initialize LCD
	lcd, err := lcd1602.New(
		lcd1602.DefaultDevice,
		lcd1602.DefaultAddress,
		16, 2, lcd1602.Font5x8, true,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer lcd.Close()

	// Define a custom heart character (5x8 pixels)
	// Each byte represents one row, bits 0-4 represent the 5 pixels
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

	// Upload to CGRAM location (3), possible locations 0-7
	if err := lcd.UploadCustomChar(3, cthulhu); err != nil {
		log.Fatal(err)
	}

	// Display the custom character at position (0, 0)
	// Use PrintRAW or WriteRAW with the location byte (3)
	if err := lcd.PrintRAW(3, 0, 0); err != nil {
		log.Fatal(err)
	}

	// Or write it at the current cursor position
	if err := lcd.WriteRAW(3); err != nil {
		log.Fatal(err)
	}
}
```