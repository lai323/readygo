package generate

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"text/template"

	"github.com/spf13/afero"
)

var (
	tmpl           *template.Template
	tmplNameMain   = "main"
	tmplNameRoot   = "root"
	tmplNameConfig = "config"
)

func init() {
	tmpl = template.New("tmpl")
	tmpls := map[string]string{
		tmplNameMain:   tmplTextMain,
		tmplNameRoot:   tmplTextRoot,
		tmplNameConfig: tmplTextConfig,
	}
	for name, text := range tmpls {
		template.Must(tmpl.New(name).Parse(text))
	}
}

type tmplVarRoot struct {
	Module     string
	WithConfig bool
}

type tmplVarConfig struct {
	Module string
}

type tmplVarMain struct {
	Module string
}

func getProjectDir(module, outdir string) (string, error) {
	var projectDir string
	if outdir == "" {
		outdir = "."
	}
	exist, err := fileExists(outdir)
	if err != nil {
		return projectDir, err
	}
	if !exist {
		return projectDir, fmt.Errorf("output dir %s not exists", outdir)
	}
	_, moduleNmae := path.Split(module)

	projectDir = path.Join(outdir, moduleNmae)
	exist, err = fileExists(projectDir)
	if err != nil {
		return projectDir, err
	}
	if exist {
		return projectDir, fmt.Errorf("dir %s exists", projectDir)
	}

	return projectDir, nil
}

func InitCli(module, outdir string, initWithConfig bool) error {
	projectDir, err := getProjectDir(module, outdir)
	if err != nil {
		return err
	}

	var (
		cmdDir          = fmt.Sprintf("%s/cmd", projectDir)
		configDir       = fmt.Sprintf("%s/config", projectDir)
		filePathMain    = fmt.Sprintf("%s/main.go", projectDir)
		filePathRoot    = fmt.Sprintf("%s/root.go", cmdDir)
		filePathCocnfig = fmt.Sprintf("%s/config.go", configDir)
	)
	err = mkdirAll(cmdDir)
	if err != nil {
		return err
	}
	if initWithConfig {
		err = mkdirAll(configDir)
		if err != nil {
			return err
		}
	}

	tocreate := map[string]string{
		tmplNameMain: filePathMain,
		tmplNameRoot: filePathRoot,
	}
	tocreateData := map[string]interface{}{
		tmplNameMain: tmplVarMain{module},
		tmplNameRoot: tmplVarRoot{module, initWithConfig},
	}
	if initWithConfig {
		tocreate[tmplNameConfig] = filePathCocnfig
		tocreateData[tmplNameConfig] = tmplVarConfig{module}
	}
	for tmplname, filepath := range tocreate {
		f, err := os.Create(filepath)
		if err != nil {
			return err
		}
		defer f.Close()
		err = tmpl.ExecuteTemplate(f, tmplname, tocreateData[tmplname])
		if err != nil {
			return err
		}
	}
	return goInit(projectDir, module)
}

func fileExists(p string) (bool, error) {
	_, err := os.Stat(p)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func mkdirAll(dir string) error {
	return afero.NewOsFs().MkdirAll(dir, 0755)
}

func goInit(projectDir, module string) error {
	cmdinit := exec.Command("go", "mod", "init", module)
	cmdinit.Dir = projectDir
	err := cmdinit.Run()
	if err != nil {
		return err
	}
	cmdtidy := exec.Command("go", "mod", "tidy")
	cmdtidy.Dir = projectDir
	return cmdtidy.Run()
}

var tmplTextMain = `
package main

import (
	"{{- .Module }}/cmd"
)

func main() {
	cmd.Execute()
}

`
var tmplTextRoot = `
package cmd

import (
	"fmt"
	"os"
	{{ if .WithConfig }}"
	{{- .Module }}/config"
	{{- end }}
	{{- if .WithConfig }}
	"github.com/spf13/afero"
	{{- end }}
	"github.com/spf13/cobra"
	{{- if .WithConfig }}
	"github.com/spf13/viper"
	{{- end }}
)

var (
	configPath string
	rootCmd    = &cobra.Command{
		Use:   "readygo module_name",
		Short: "create empty project with cobra and spf13",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
			// Do Stuff Here
		},
	}
	subCmd = &cobra.Command{
		Use:   "subcmd",
		Short: "sub command example",
		Run: func(cmd *cobra.Command, args []string) {
			// cmd.Help()
			{{- if .WithConfig }}
			fmt.Println(viper.GetString("var"))
			fmt.Println(viper.GetString("VarFromFile"))
			{{- else }}
			// Do Stuff Here
			{{- end }}
		},
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
{{- if .WithConfig }}
func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", fmt.Sprintf("config file (default is %s)", config.DefaultConfigPath))
	rootCmd.PersistentFlags().String("var", "ViperTest var", "use Viper for configuration")
	viper.BindPFlag("var", rootCmd.PersistentFlags().Lookup("var"))
	rootCmd.AddCommand(subCmd)
}

func initConfig() {
	err := config.InitConfig(afero.NewOsFs(), configPath)
	if err != nil {
		panic(err)
	}
}
{{- else}}
func init() {
	rootCmd.AddCommand(subCmd)
}
{{- end }}

`

var tmplTextConfig = `
package config

import (
	"fmt"
	"log"
	"path"

	"github.com/adrg/xdg"
	"github.com/pelletier/go-toml"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

var (
	defaultConfigDir  string
	DefaultConfigPath string
	defaultStorageDir string
	defaultConfig     = map[string]interface{}{
		"VarFromFile": "ViperTest From file",
	}
)

func init() {
	var err error
	DefaultConfigPath, err = xdg.ConfigFile("{{- .Module }}/{{- .Module }}.toml")
	if err != nil {
		log.Fatal(err)
	}
	defaultConfigDir = path.Dir(DefaultConfigPath)
	defaultStorageDir = path.Join(xdg.DataHome, "{{- .Module }}")

	err = createDefaultFile(afero.NewOsFs())
	if err != nil {
		log.Fatal(err)
	}
}

type initConfigErr struct {
	s string
}

func (e *initConfigErr) Error() string {
	return e.s
}

func newInitConfigErr(err error) error {
	return &initConfigErr{
		s: fmt.Sprintf("Init config error: %s", err.Error()),
	}
}

func createDefaultFile(fs afero.Fs) error {
	err := fs.MkdirAll(defaultConfigDir, 0755)
	if err != nil {
		return err
	}
	fs.MkdirAll(defaultStorageDir, 0755)
	if err != nil {
		return err
	}

	exist, err := afero.Exists(fs, DefaultConfigPath)
	if err != nil {
		return err
	}

	if !exist {
		handle, err := fs.Create(DefaultConfigPath)
		if err != nil {
			return err
		}
		defer handle.Close()
		t, err := toml.TreeFromMap(defaultConfig)
		if err != nil {
			return err
		}
		handle.WriteString(t.String())
		if err != nil {
			return err
		}
	}
	return nil
}

func InitConfig(fs afero.Fs, configPath string) error {
	if configPath == "" {
		viper.SetConfigFile(DefaultConfigPath)
	} else {
		exist, err := afero.Exists(fs, configPath)
		if err != nil {
			return newInitConfigErr(err)
		}
		if !exist {
			return &initConfigErr{
				s: fmt.Sprintf("Init config error: %s not exist", configPath),
			}
		}
		viper.SetConfigFile(configPath)
	}
	return viper.ReadInConfig()
}
`
