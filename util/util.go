package iptbutil

import (
	"fmt"
)

func YesNoPrompt(prompt string) bool {
	var s string
	for {
		fmt.Printf("%s [y/n] ", prompt)
		fmt.Scanf("%s", &s)
		switch s {
		case "y", "Y":
			return true
		case "n", "N":
			return false
		}
		fmt.Println("Please press either 'y' or 'n'")
	}
}
