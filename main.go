package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/raedahgroup/dcrcli/cli/commands"

	"github.com/raedahgroup/dcrcli/config"

	"github.com/jessevdk/go-flags"

	"github.com/raedahgroup/dcrcli/cli"
	"github.com/raedahgroup/dcrcli/walletrpcclient"
	"github.com/raedahgroup/dcrcli/web"
)

func main() {
	config, parser, err := config.LoadConfig()
	if err != nil {
		handleParseError(err, parser)
		os.Exit(1)
	}
	if config == nil {
		os.Exit(0)
	}

	client, err := walletrpcclient.New(config.WalletRPCServer, config.RPCCert, config.NoDaemonTLS)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error connecting to RPC server")
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if config.HTTPMode {
		enterHTTPMode(config, client)
	} else {
		enterCliMode(client)
	}
}

func enterHTTPMode(config *config.Config, client *walletrpcclient.Client) {
	fmt.Println("Running in http mode")
	web.StartHttpServer(config.HTTPServerAddress, client)
}

func enterCliMode(client *walletrpcclient.Client) {
	// Set the walletrpcclient.Client object that will be used by the command handlers
	cli.WalletClient = client

	parser := flags.NewParser(&commands.AppCommands{}, flags.HelpFlag|flags.PassDoubleDash)
	_, err := parser.Parse()
	if config.IsFlagErrorType(err, flags.ErrCommandRequired) {
		// No command was specified, print the available commands.
		availableCommands := supportedCommands(parser)
		fmt.Fprintln(os.Stderr, "Available Commands: ", strings.Join(availableCommands, ", "))
	} else {
		handleParseError(err, parser)
	}
	os.Exit(1)
}

func supportedCommands(parser *flags.Parser) []string {
	registeredCommands := parser.Commands()
	commands := make([]string, 0, len(registeredCommands))
	for _, command := range registeredCommands {
		commands = append(commands, command.Name)
	}
	sort.Strings(commands)
	return commands
}

func handleParseError(err error, parser *flags.Parser) {
	if err == nil {
		return
	}
	if (parser.Options & flags.PrintErrors) != flags.None {
		// error printing is already handled by go-flags.
		return
	}
	if config.IsFlagErrorType(err, flags.ErrHelp) {
		if parser.Active == nil {
			parser.WriteHelp(os.Stderr)
		} else {
			helpParser := flags.NewParser(nil, flags.HelpFlag)
			helpParser.Name = parser.Name
			helpParser.Active = parser.Active
			helpParser.WriteHelp(os.Stderr)
		}
	} else {
		fmt.Println(err)
	}
}
