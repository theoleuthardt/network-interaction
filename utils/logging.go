package utils

import (
	"fmt"
	"github.com/fatih/color"
)

// Logging helper functions
func LogError(message string) {
	fmt.Println(color.RedString("[ERROR] %s", message))
}

func logCritical(message string) {
	fmt.Println(color.New(color.BgRed, color.Bold, color.FgWhite).Sprintf("[CRITICAL] %s", message))
}

func logWarning(message string) {
	fmt.Println(color.YellowString("[WARNING] %s", message))
}

func logInfo(message string) {
	fmt.Println(color.WhiteString("[INFO] %s", message))
}

func printTestLogs() {
	LogError("This is an error log example.")
	logCritical("This is a critical log example.")
	logWarning("This is a warning log example.")
	logInfo("This is an informational log example.")
	fmt.Println()
}

func main() {
	printTestLogs()
}
