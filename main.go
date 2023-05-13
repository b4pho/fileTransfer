package main

import (
	"fileTransfer/configuration"
	files "fileTransfer/filesystem"
	"fileTransfer/sftp"
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
	configHostname := cmd.String("host", "localhost", "SFTP host")
	configPort := cmd.Int("port", 22, "SFTP port")
	configUsername := cmd.String("user", "test", "SFTP username")
	configMaxConnections := cmd.Int("max-connections", 3, "SFTP number of max concurrent connections")
	configServerFolder := cmd.String("folder", ".", "SFTP folder on sftp server")
	cmd.Parse(args)
	config := configuration.New()
	config.Hostname = *configHostname
	config.Port = *configPort
	config.Username = *configUsername
	config.MaxConnections = *configMaxConnections
	config.ServerFolder = *configServerFolder
	err := config.Store()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Configuration stored")

	// create and store local and simplified rappresentation of the "filesystem"
	_, err = files.New()
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
	conn, err := sftp.Connect(*config, password)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	err = sftp.PushChanges(conn, *config)
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
	conn, err := sftp.Connect(*config, password)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	err = sftp.Clone(conn, *config)
	if err != nil {
		log.Fatal(err)
	}
}
