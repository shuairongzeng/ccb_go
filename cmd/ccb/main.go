package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anthropics/claude_code_bridge/internal/client"
	"github.com/anthropics/claude_code_bridge/internal/daemon"
	"github.com/anthropics/claude_code_bridge/internal/launcher"
	"github.com/anthropics/claude_code_bridge/internal/output"
	"github.com/anthropics/claude_code_bridge/internal/protocol"
)

var version = "dev"

// knownSubcommands lists all cobra subcommands so we can distinguish
// "ccb codex,claude" (provider launch) from "ccb daemon start" (subcommand).
var knownSubcommands = map[string]bool{
	"ask": true, "ping": true, "pend": true, "daemon": true,
	"help": true, "completion": true,
	"cask": true, "gask": true, "oask": true, "dask": true, "lask": true,
	"cping": true, "gping": true, "oping": true, "dping": true, "lping": true,
	"cpend": true, "gpend": true, "opend": true, "dpend": true, "lpend": true,
}

func main() {
	// Pre-cobra interception: if the first non-flag arg is NOT a known
	// subcommand, treat it as a provider launch (e.g. "ccb -a codex,claude").
	if shouldRunLauncher(os.Args[1:]) {
		runLauncher(os.Args[1:])
		return
	}

	rootCmd := buildRootCmd()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// shouldRunLauncher checks if the CLI args look like a provider launch
// rather than a subcommand invocation.
func shouldRunLauncher(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" || arg == "--version" || arg == "-v" {
			return false
		}
		// Skip flags
		if strings.HasPrefix(arg, "-") {
			continue
		}
		// First positional arg: is it a known subcommand?
		return !knownSubcommands[arg]
	}
	return false
}

// runLauncher handles "ccb [-a] [-r] provider1,provider2 ..." directly.
func runLauncher(args []string) {
	auto := false
	resume := false
	var providerArgs []string

	for _, arg := range args {
		switch arg {
		case "-a", "--auto":
			auto = true
		case "-r", "--resume":
			resume = true
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(os.Stderr, "unknown flag: %s\n", arg)
				os.Exit(1)
			}
			providerArgs = append(providerArgs, arg)
		}
	}

	if len(providerArgs) == 0 {
		fmt.Fprintln(os.Stderr, "no providers specified. Available: codex, gemini, opencode, claude, droid")
		os.Exit(1)
	}

	providers := launcher.ParseProviders(providerArgs)
	if len(providers) == 0 {
		fmt.Fprintln(os.Stderr, "no valid providers specified. Available: codex, gemini, opencode, claude, droid")
		os.Exit(1)
	}

	cwd, _ := os.Getwd()

	results, err := launcher.Launch(launcher.LaunchConfig{
		Providers: providers,
		Auto:      auto,
		Resume:    resume,
		WorkDir:   cwd,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	ok := 0
	for _, r := range results {
		if r.Error == nil {
			ok++
		}
	}

	if ok == 0 {
		fmt.Fprintln(os.Stderr, "failed to start any provider")
		os.Exit(1)
	}

	fmt.Printf("\n%d/%d providers started", ok, len(providers))
	if resume {
		fmt.Printf(" (resume mode)")
	}
	if auto {
		fmt.Printf(" (auto-approve mode)")
	}
	fmt.Println()
}

func buildRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "ccb [providers...]",
		Short: "Claude Code Bridge - Multi-model AI collaboration tool",
		Long: `Claude Code Bridge - Multi-model AI collaboration tool

Launch multiple AI providers simultaneously:
  ccb codex,claude              Start codex and claude
  ccb -a codex,gemini,claude    Start with auto-approve mode (skip confirmations)
  ccb -r codex,claude           Resume previous sessions
  ccb -a -r codex,claude        Resume with auto-approve mode
  ccb codex gemini              Space-separated is also supported

Available providers: codex, gemini, opencode, claude, droid`,
		Version: version,
	}

	// --- daemon subcommand ---
	daemonCmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage the CCB daemon",
	}

	daemonStartCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			return daemon.RunDefault()
		},
	}

	daemonStopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := client.ReadState("")
			if err != nil {
				return fmt.Errorf("daemon not running")
			}
			if err := client.ShutdownDaemon(state); err != nil {
				return err
			}
			fmt.Println("Daemon stopped")
			return nil
		},
	}

	daemonStatusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show daemon status",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, status, err := client.DaemonStatus()
			if err != nil {
				if state == nil {
					return fmt.Errorf("daemon not running")
				}
				return err
			}
			fmt.Printf("PID:       %d\n", state.PID)
			fmt.Printf("Address:   %s:%d\n", state.Host, state.Port)
			if providers, ok := status["providers"].([]interface{}); ok {
				names := make([]string, 0, len(providers))
				for _, p := range providers {
					if s, ok := p.(string); ok {
						names = append(names, s)
					}
				}
				fmt.Printf("Providers: %s\n", strings.Join(names, ", "))
			}
			if workers, ok := status["workers"].(float64); ok {
				fmt.Printf("Workers:   %d\n", int(workers))
			}
			return nil
		},
	}

	daemonCmd.AddCommand(daemonStartCmd, daemonStopCmd, daemonStatusCmd)

	// --- ask subcommand ---
	var askTimeout float64
	var askQuiet bool

	askCmd := &cobra.Command{
		Use:   "ask <provider> <message...>",
		Short: "Send a message to an AI provider",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			provider := args[0]
			message := strings.Join(args[1:], " ")

			// Read from stdin if message is "-"
			if message == "-" {
				data, err := os.ReadFile("/dev/stdin")
				if err != nil {
					return fmt.Errorf("failed to read stdin: %w", err)
				}
				message = output.DecodeStdinBytes(data)
			}

			result, err := client.Ask(client.AskRequest{
				Provider: provider,
				Message:  message,
				TimeoutS: askTimeout,
				Quiet:    askQuiet,
			})
			if err != nil {
				return err
			}

			if result.Error != "" && result.ExitCode != 0 {
				output.Errorf("%s", result.Error)
			}
			if result.Reply != "" {
				fmt.Println(result.Reply)
			}
			os.Exit(result.ExitCode)
			return nil
		},
	}
	askCmd.Flags().Float64VarP(&askTimeout, "timeout", "t", 120, "Timeout in seconds")
	askCmd.Flags().BoolVarP(&askQuiet, "quiet", "q", false, "Suppress progress output")

	// --- ping subcommand ---
	pingCmd := &cobra.Command{
		Use:   "ping <provider>",
		Short: "Test connectivity with an AI provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			provider := args[0]
			if err := client.Ping(provider); err != nil {
				fmt.Printf("%s: offline (%s)\n", provider, err)
				os.Exit(1)
			}
			fmt.Printf("%s: online\n", provider)
			return nil
		},
	}

	// --- pend subcommand ---
	pendCmd := &cobra.Command{
		Use:   "pend <provider>",
		Short: "View latest reply from an AI provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			provider := args[0]
			reply, err := client.Pend(provider)
			if err != nil {
				return err
			}
			if reply == "" {
				fmt.Println("(no reply)")
				os.Exit(output.ExitNoReply)
			}
			// Strip trailing markers for clean display
			reply = protocol.StripTrailingMarkers(reply)
			fmt.Println(reply)
			return nil
		},
	}

	// --- Provider shortcut commands ---
	providerShortcuts := map[string]string{
		"cask": "codex",
		"gask": "gemini",
		"oask": "opencode",
		"dask": "droid",
		"lask": "claude",
	}

	for shortcut, provider := range providerShortcuts {
		p := provider // capture
		shortcutCmd := &cobra.Command{
			Use:   shortcut + " <message...>",
			Short: fmt.Sprintf("Send a message to %s (shortcut for 'ask %s')", p, p),
			Args:  cobra.MinimumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				message := strings.Join(args, " ")
				if message == "-" {
					data, err := os.ReadFile("/dev/stdin")
					if err != nil {
						return fmt.Errorf("failed to read stdin: %w", err)
					}
					message = output.DecodeStdinBytes(data)
				}

				result, err := client.Ask(client.AskRequest{
					Provider: p,
					Message:  message,
					TimeoutS: askTimeout,
					Quiet:    askQuiet,
				})
				if err != nil {
					return err
				}

				if result.Error != "" && result.ExitCode != 0 {
					output.Errorf("%s", result.Error)
				}
				if result.Reply != "" {
					fmt.Println(result.Reply)
				}
				os.Exit(result.ExitCode)
				return nil
			},
		}
		shortcutCmd.Flags().Float64VarP(&askTimeout, "timeout", "t", 120, "Timeout in seconds")
		shortcutCmd.Flags().BoolVarP(&askQuiet, "quiet", "q", false, "Suppress progress output")
		rootCmd.AddCommand(shortcutCmd)
	}

	// --- Provider ping shortcuts ---
	for shortcut, provider := range providerShortcuts {
		p := provider
		pingShortcut := &cobra.Command{
			Use:   shortcut[:1] + "ping",
			Short: fmt.Sprintf("Ping %s (shortcut for 'ping %s')", p, p),
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := client.Ping(p); err != nil {
					fmt.Printf("%s: offline (%s)\n", p, err)
					os.Exit(1)
				}
				fmt.Printf("%s: online\n", p)
				return nil
			},
		}
		rootCmd.AddCommand(pingShortcut)
	}

	// --- Provider pend shortcuts ---
	for shortcut, provider := range providerShortcuts {
		p := provider
		pendShortcut := &cobra.Command{
			Use:   shortcut[:1] + "pend",
			Short: fmt.Sprintf("View latest reply from %s", p),
			RunE: func(cmd *cobra.Command, args []string) error {
				reply, err := client.Pend(p)
				if err != nil {
					return err
				}
				if reply == "" {
					fmt.Println("(no reply)")
					os.Exit(output.ExitNoReply)
				}
				reply = protocol.StripTrailingMarkers(reply)
				fmt.Println(reply)
				return nil
			},
		}
		rootCmd.AddCommand(pendShortcut)
	}

	rootCmd.AddCommand(daemonCmd, askCmd, pingCmd, pendCmd)

	return rootCmd
}
