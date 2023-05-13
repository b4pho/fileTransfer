package terminal

import (
	"fmt"
	"log"
	"strings"
	"syscall"

	"golang.org/x/term"
)

func InputPassword() string {
	fmt.Println("Password:")
	passwordInBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal(err)
	}
	return string(passwordInBytes)
}

func Confirm(message string) bool {
	fmt.Println(message, "(yes/y/no/n)")
	var confirmation string
	fmt.Scanf("%s", &confirmation)
	confirmation = strings.ToLower(confirmation)
	return (confirmation == "yes" || confirmation == "y")
}
