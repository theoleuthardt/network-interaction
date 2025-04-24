package utils

import (
	"fmt"
	"github.com/fatih/color"
)

// LogError Logging helper functions
func LogError(message string) {
	fmt.Println(color.RedString("[ERROR] %s", message))
}

func LogCritical(message string) {
	fmt.Println(color.New(color.BgRed, color.Bold, color.FgWhite).Sprintf("[CRITICAL] %s", message))
}

func LogWarning(message string) {
	fmt.Println(color.YellowString("[WARNING] %s", message))
}

func LogInfo(message string) {
	fmt.Println(color.WhiteString("[INFO] %s", message))
}

func printTestLogs() {
	LogError("This is an error log example.")
	LogCritical("This is a critical log example.")
	LogWarning("This is a warning log example.")
	LogInfo("This is an informational log example.")
	fmt.Println()
}

func main() {
	printTestLogs()
}
