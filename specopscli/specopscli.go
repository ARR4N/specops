// Package specopscli provides a CLI for developing specops.Code.
package specopscli

import (
	"fmt"
	"log"

	"github.com/arr4n/specops"
	"github.com/spf13/cobra"
)

// Run runs a CLI for performing commands on the Code. It should be called from
// a main.main() function and will parse command-line arguments and flags to
// perform available commands. For usage, invoke the binary without any
// arguments.
func Run(code specops.Code) {
	if err := run(code); err != nil {
		log.Fatal(err)
	}
}

func run(code specops.Code) error {
	compile := &cobra.Command{
		Use:   "compile",
		Short: "Compile bytecode",
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := code.Compile()
			if err != nil {
				return err
			}
			fmt.Printf("%#x\n", out)
			return nil
		},
	}

	var callData []byte

	exec := &cobra.Command{
		Use:   "exec",
		Short: "Compile then execute bytecode",
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := code.Run(nil)
			if err != nil {
				return err
			}
			fmt.Printf("%#x\n", out)
			return nil
		},
	}

	debug := &cobra.Command{
		Use:   "debug",
		Short: "Compile then debug bytecode execution",
		RunE: func(cmd *cobra.Command, args []string) error {
			return code.RunTerminalDebugger(callData)
		},
	}

	for _, c := range []*cobra.Command{exec, debug} {
		c.Flags().BytesHexVarP(&callData, "calldata", "d", nil, "Call data")
	}

	cmd := &cobra.Command{
		Short: "SPEC0PS domain-specific language & compiler for Ethereum VM bytecode",
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}
	cmd.AddCommand(
		compile,
		exec,
		debug,
	)
	return cmd.Execute()
}
