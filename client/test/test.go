package main

import (
	"fmt"
	"os/exec"
)

func main() {
	fmt.Println("Hello World")

	// run the bash command ./test.sh
	// Define the path to the bash script
	scriptPath := "./test.sh"

	// Create the command to run the script
	cmd := exec.Command("bash", scriptPath)

	// Run the command and capture its output
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to run script: %s\n", err)
	} else {
		fmt.Printf("Script output: %s\n", output)
	}
}
