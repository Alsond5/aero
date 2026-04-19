// Package color provides ANSI escape code constants and helper functions
// for writing colored text to the terminal.
package color

import "fmt"

// ANSI escape code constants for terminal text styling.
// Use with [Colorize], [Print], [Println], or [Printf].
const (
	Reset = "\033[0m" // Resets all attributes.
	Bold  = "\033[1m" // Bold text.

	Black  = "\033[30m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	White  = "\033[37m"

	BrightBlack  = "\033[38;5;248m"
	BrightRed    = "\033[91m"
	BrightGreen  = "\033[92m"
	BrightYellow = "\033[93m"
	BrightBlue   = "\033[94m"
	BrightPurple = "\033[95m"
	BrightCyan   = "\033[96m"
	BrightWhite  = "\033[97m"
)

// Colorize wraps text with the given ANSI color code and appends a reset
// sequence, returning the resulting string without printing it.
//
//	s := color.Colorize(color.Green, "success")
func Colorize(color, text string) string {
	return color + text + Reset
}

// Print prints color-wrapped text to stdout without a newline.
func Print(color, text string) {
	fmt.Print(Colorize(color, text))
}

// Println prints color-wrapped text to stdout followed by a newline.
func Println(color, text string) {
	fmt.Println(Colorize(color, text))
}

// Printf formats according to format, wraps the result in the given color,
// and prints it to stdout.
//
//	color.Printf(color.Yellow, "warning: %s", msg)
func Printf(color, format string, args ...any) {
	fmt.Printf("%s", Colorize(color, fmt.Sprintf(format, args...)))
}
