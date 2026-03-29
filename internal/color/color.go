package color

import "fmt"

const (
	Reset = "\033[0m"
	Bold  = "\033[1m"

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

func Colorize(color, text string) string {
	return color + text + Reset
}

func Print(color, text string) {
	fmt.Print(Colorize(color, text))
}

func Println(color, text string) {
	fmt.Println(Colorize(color, text))
}

func Printf(color, format string, args ...any) {
	fmt.Printf("%s", Colorize(color, fmt.Sprintf(format, args...)))
}
