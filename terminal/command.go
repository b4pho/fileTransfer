package terminal

import (
	"flag"
	"fmt"
	"os"
)

type CommandGroup map[string]Command

type Command struct {
	Name        string
	Description string
	flagSet     *flag.FlagSet
	callback    func(*flag.FlagSet, []string)
}

func (cg CommandGroup) Add(c Command) {
	cg[c.Name] = c
}

func (cg CommandGroup) GetFlagSet(name string) *flag.FlagSet {
	return cg[name].flagSet
}

func (cg CommandGroup) PrintUsage() {
	fmt.Println("USAGE: [command] [options]")
	fmt.Println("Possible commands:")
	for _, command := range cg {
		fmt.Printf("\t* %-15s %s\n", command.Name, command.Description)
	}
	fmt.Println()
	fmt.Println("For info about options use: [command] --help")
	os.Exit(1)
}

func (cg CommandGroup) Parse() {
	if len(os.Args) < 2 {
		cg.PrintUsage()
	}

	commandName := os.Args[1]
	if command, ok := cg[commandName]; ok {
		command.callback(command.flagSet, os.Args[2:])
	} else {
		fmt.Printf("Unknown command: '%s'\n\n", commandName)
		cg.PrintUsage()
	}
}

func NewCommand(name, description string, callback func(*flag.FlagSet, []string)) Command {
	return Command{
		name,
		description,
		flag.NewFlagSet(name, flag.ExitOnError),
		callback,
	}
}
