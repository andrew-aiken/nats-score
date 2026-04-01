package checks

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func Purge(force bool) error {
	if !force && askForConfirmation("Are you sure you want to proceed?") {
		fmt.Println("Proceeding...")
	} else {
		fmt.Println("Aborting.")
	}

	return listChecks(true)
}

func askForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/N]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		switch response {
			case "y", "yes":
				return true
			case "n", "no", "":
				return false
			default:
		}

		// If the input is something else, the loop continues and prompts again
		fmt.Println("Invalid input. Please enter 'y' or 'n'.")
	}
}
