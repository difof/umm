package cmdhelp

import (
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

func AttachAppendix(cmd *cobra.Command, doc Document) {
	target := cmd
	defaultHelp := cmd.HelpFunc()
	cmd.SetHelpFunc(func(helped *cobra.Command, args []string) {
		defaultHelp(helped, args)
		if helped != target {
			return
		}
		block := Render(doc, RenderOptions{Color: isTerminalWriter(helped.OutOrStdout())})
		if strings.TrimSpace(block) == "" {
			return
		}
		_, _ = fmt.Fprint(helped.OutOrStdout(), "\n"+block)
	})
}

func isTerminalWriter(out any) bool {
	file, ok := out.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(file.Fd()) || isatty.IsCygwinTerminal(file.Fd())
}
