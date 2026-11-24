package cmd

import (
	"fmt"

	"github.com/rwilgaard/alfred-1password-search/src/internal/util"
	"github.com/spf13/cobra"
)

const passwordLengthDefault = 32

var (
	lengthFlag int

	generateCmd = &cobra.Command{
		Use:          "generate",
		Short:        "generate a new password",
		SilenceUsage: true,
		Run: func(_ *cobra.Command, _ []string) {
			pws, err := util.GeneratePassword(lengthFlag, true, cfg.AllowedSymbols)
			if err != nil {
				wf.FatalError(err)
			}
			pwn, err := util.GeneratePassword(lengthFlag, false, "")
			if err != nil {
				wf.FatalError(err)
			}

			sub := fmt.Sprintf("⏎ to copy to clipboard  •  Length: %d", lengthFlag)

			wf.NewItem(pws).
				Subtitle(sub).
				Var("password", pws).
				Arg("copy").
				Valid(true)

			wf.NewItem(pwn).
				Subtitle(sub+"  •  No symbols").
				Var("password", pwn).
				Arg("copy").
				Valid(true)

			wf.Send()
		},
	}
)

func init() {
	generateCmd.Flags().IntVarP(&lengthFlag, "length", "l", passwordLengthDefault, "password length")
	rootCmd.AddCommand(generateCmd)
}
