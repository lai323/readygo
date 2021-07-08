package cmd

import (
	"fmt"
	"os"

	"github.com/lai323/readygo/generate"
	"github.com/spf13/cobra"
)

var (
	initWithOutConfig   bool
	enableDefaultConfig bool
	initWithOutLog      bool
	rootCmd             = &cobra.Command{
		Use:   "readygo module_name out_dir",
		Short: "create empty project with cobra and spf13",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("requires module name argument")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			var outdir string
			if len(args) > 1 {
				outdir = args[1]
			}
			err := generate.InitCli(args[0], outdir, initWithOutConfig, enableDefaultConfig, initWithOutLog)
			if err != nil {
				fmt.Printf("error: %s\n", err.Error())
			}
		},
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&initWithOutConfig, "with-out-config", false, "Not use configuration files")
	rootCmd.PersistentFlags().BoolVar(&initWithOutLog, "with-out-log", false, "Not generate logger")
	rootCmd.PersistentFlags().BoolVar(&enableDefaultConfig, "enable-default-config", false, "generate default config in XDG_CONFIG_HOME, if config file not exist")
}
