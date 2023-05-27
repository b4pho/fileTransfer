package main

import (
	"fileTransfer/configuration"
	files "fileTransfer/filesystem"
	"fileTransfer/protocols"

	"fileTransfer/terminal"
	"flag"
	"fmt"
	"log"
)

func main() {
	commands := terminal.CommandGroup{}
	commands.Add(terminal.NewCommand("init", "prepares local configuration to connect to server", Init))
	commands.Add(terminal.NewCommand("publish", "uploads latest modified files to server", Publish))
	commands.Add(terminal.NewCommand("clone", "downloads all server content to current working directory", Clone))
	commands.Parse()
}

func Init(cmd *flag.FlagSet, args []string) {
	hostname := cmd.String("host", "localhost", "server host")
	port := cmd.Int("port", 22, "server port")
	username := cmd.String("user", "test", "server username")
	maxConnections := cmd.Int("max-connections", 3, "server number of max concurrent connections")
	serverFolder := cmd.String("folder", ".", "folder on server")
	protocol := cmd.String("protocol", "SFTP", "Protocols available: SFTP, FTP, FTPS-IMPLICIT, FTPS-EXPLICIT")

	cmd.Parse(args)
	config := configuration.New()
	config.Hostname = *hostname
	config.Port = *port
	config.Username = *username
	config.MaxConnections = *maxConnections
	config.ServerFolder = *serverFolder
	config.Protocol = *protocol
	err := config.Store()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Configuration stored")

	_, err = files.CreateAndStoreFileList()
	if err != nil {
		log.Fatal(err)
	}
}

func Publish(cmd *flag.FlagSet, args []string) {
	cmd.Parse(args)
	password := terminal.InputPassword()
	config, err := configuration.Read()
	if err != nil {
		log.Fatal(err)
	}
	conn, err := protocols.Connect(*config, password)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	err = protocols.PushChanges(conn, *config)
	if err != nil {
		log.Fatal(err)
	}
}

func Clone(cmd *flag.FlagSet, args []string) {
	cmd.Parse(args)
	password := terminal.InputPassword()
	config, err := configuration.Read()
	if err != nil {
		log.Fatal(err)
	}
	conn, err := protocols.Connect(*config, password)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	err = protocols.Clone(conn, *config)
	if err != nil {
		log.Fatal(err)
	}
}
