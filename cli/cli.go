package cli

import (
	"context"
	"fmt"

	"github.com/jessevdk/go-flags"

	"github.com/raedahgroup/godcr/cli/commands"
	"github.com/raedahgroup/godcr/cli/terminalprompt"
	"github.com/raedahgroup/godcr/config"
	"github.com/raedahgroup/godcr/walletsource"

	"github.com/raedahgroup/dcrcli/cli/terminalprompt"
	"github.com/raedahgroup/dcrcli/config"
	"github.com/raedahgroup/dcrcli/core"

	"github.com/raedahgroup/dcrcli/app"
	"github.com/raedahgroup/dcrcli/app/config"
)

var (
	Wallet core.Wallet
	StdoutWriter = tabWriter(os.Stdout)
)

// Root is the entrypoint to the cli application.
// It defines both the commands and the options available.
type Root struct {
	Commands commands.CliCommands
	Config   config.Config
}

// commandHandler provides a type name for the command handler to register on flags.Parser
type commandHandler func(flags.Commander, []string) error

// CommandHandlerWrapper provides a command handler that provides walletrpcclient.Client
// to commands.WalletCommandRunner types. Other command that satisfy flags.Commander and do not
// depend on walletrpcclient.Client will be run as well.
// If the command does not satisfy any of these types, ErrNotSupported will be returned.
func CommandHandlerWrapper(parser *flags.Parser, walletSource ws.WalletSource) commandHandler {
	return func(command flags.Commander, args []string) error {
		if command == nil {
			return brokenCommandError(parser.Command)
		}
		if commandRunner, ok := command.(commands.WalletCommandRunner); ok {
			return commandRunner.Run(walletSource, args)
		}
		return command.Execute(args)
	}
}

func brokenCommandError(command *flags.Command) error {
	return fmt.Errorf("The command %q was not properly setup.\n" +
		"Please report this bug at https://github.com/raedahgroup/godcr/issues",
		commandName(command))
}

func commandName(command *flags.Command) string {
	name := command.Name
	if command.Active != nil {
		return fmt.Sprintf("%s %s", name, commandName(command.Active))
	}
	return name
}

// Run starts the app in cli interface mode
func Run(ctx context.Context, walletMiddleware app.WalletMiddleware, appConfig *config.Config) error {
	if appConfig.CreateWallet {
		return createWallet(ctx, walletMiddleware)
	}

	// open wallet, subsequent operations including blockchain sync and command handlers need wallet to be open
	walletExists, err := openWallet(ctx, walletMiddleware)
	if err != nil || !walletExists {
		return err
	}

	if appConfig.SyncBlockchain {
		err = syncBlockChain(ctx, walletMiddleware)
		if err != nil {
			return err
		}
	}

	// Set the core wallet object that will be used by the command handlers
	utils.Wallet = walletMiddleware

	// parser.Parse checks if a command is passed and invokes the Execute method of the command
	// if no command is passed, parser.Parse returns an error of type ErrCommandRequired
	parser := flags.NewParser(appConfig, flags.HelpFlag|flags.PassDoubleDash)
	_, err = parser.Parse()

	// help flag error should have been caught and handled in config.LoadConfig, so only check for ErrCommandRequired
	noCommandPassedError := config.IsFlagErrorType(err, flags.ErrCommandRequired)

	// command mustn't be passed with --sync flag
	if noCommandPassedError && !appConfig.SyncBlockchain {
		displayAvailableCommandsHelpMessage(parser)
	} else if err != nil && !noCommandPassedError {
		fmt.Println(err)
	}

	return err
}
