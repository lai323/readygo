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
	tmplNameLog    = "log"
)

func init() {
	tmpl = template.New("tmpl")
	tmpls := map[string]string{
		tmplNameMain:   tmplTextMain,
		tmplNameRoot:   tmplTextRoot,
		tmplNameConfig: tmplTextConfig,
		tmplNameLog:    tmplTextLog,
	}
	for name, text := range tmpls {
		template.Must(tmpl.New(name).Parse(text))
	}
}

type tmplVarRoot struct {
	Module              string
	WithConfig          bool
	EnableDefaultConfig bool
}

type tmplVarConfig struct {
	Module              string
	EnableDefaultConfig bool
}

type tmplVarLog struct {
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

func InitCli(module, outdir string, initWithOutConfig, enableDefaultConfig, initWithOutLog bool) error {
	projectDir, err := getProjectDir(module, outdir)
	if err != nil {
		return err
	}

	var (
		cmdDir         = fmt.Sprintf("%s/cmd", projectDir)
		configDir      = fmt.Sprintf("%s/config", projectDir)
		logDir         = fmt.Sprintf("%s/logger", projectDir)
		filePathMain   = fmt.Sprintf("%s/main.go", projectDir)
		filePathRoot   = fmt.Sprintf("%s/root.go", cmdDir)
		filePathConfig = fmt.Sprintf("%s/config.go", configDir)
		filePathLog    = fmt.Sprintf("%s/logger.go", logDir)
	)
	err = mkdirAll(cmdDir)
	if err != nil {
		return err
	}
	if !initWithOutConfig {
		err = mkdirAll(configDir)
		if err != nil {
			return err
		}
	}
	if !initWithOutLog {
		err = mkdirAll(logDir)
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
		tmplNameRoot: tmplVarRoot{module, !initWithOutConfig, enableDefaultConfig},
	}
	if !initWithOutConfig {
		tocreate[tmplNameConfig] = filePathConfig
		tocreateData[tmplNameConfig] = tmplVarConfig{module, enableDefaultConfig}
	}
	if !initWithOutLog {
		tocreate[tmplNameLog] = filePathLog
		tocreateData[tmplNameLog] = tmplVarLog{}
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

	cmdreplace := exec.Command("go", "mod", "edit", "-replace", "github.com/spf13/afero=github.com/spf13/afero@v1.5.1")
	cmdreplace.Dir = projectDir
	err = cmdreplace.Run()
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
	{{- if .EnableDefaultConfig }}
	"github.com/spf13/afero"
	{{- end }}
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
	{{- if .EnableDefaultConfig }}
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", fmt.Sprintf("config file (default is %s)", config.DefaultConfigPath))
	{{- end }}
	rootCmd.PersistentFlags().String("var", "ViperTest var", "use Viper for configuration")
	viper.BindPFlag("var", rootCmd.PersistentFlags().Lookup("var"))
	rootCmd.AddCommand(subCmd)
}

func initConfig() {
	{{- if .EnableDefaultConfig }}
	err := config.InitConfig(afero.NewOsFs(), configPath)
	{{- else }}
	err := config.InitConfig(configPath)
	{{- end }}
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

{{- if .EnableDefaultConfig }}
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
{{- else}}
import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

func InitConfig(configPath string) error {
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("InitConfig error: config file (%s) not exist", configPath)
		}
		return fmt.Errorf("InitConfig error: %s", err.Error())
	}

	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("InitConfig error: %s", err.Error())
	}
	return nil
}
{{- end }}
`
var tmplTextLog = `
package logger

import (
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/lestrrat/go-file-rotatelogs"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func NewLoggerByConfig() (*logrus.Logger, error) {
	level := viper.GetString("log.level")
	stdout := viper.GetBool("log.stdout")
	format := viper.GetString("log.format")
	logdir := viper.GetString("log.logdir")
	logfile := viper.GetString("log.logfile")
	if logdir == "" {
		logdir = "./log"
	}
	if logfile == "" {
		logfile = "log"
	}
	fmt.Println(level, stdout, format, logdir, logfile)
	return NewLogger(level, format, stdout, logdir, logfile)
}

func newloggerErr(format string, a ...interface{}) error {
	return fmt.Errorf("NewLogger error: %s", fmt.Sprintf(format, a...))
}

func NewLogger(loglevel, format string, stdout bool, logdir, logfile string) (*logrus.Logger, error) {
	var logger = logrus.New()
	var level = logrus.DebugLevel
	switch loglevel {
	case "panic":
		level = logrus.PanicLevel
	case "fatal":
		level = logrus.FatalLevel
	case "error":
		level = logrus.ErrorLevel
	case "warn":
		level = logrus.WarnLevel
	case "info":
		level = logrus.InfoLevel
	case "debug":
		level = logrus.DebugLevel
	case "trace":
		level = logrus.TraceLevel
	default:
		return logger, newloggerErr("NewLogger error log level not allow: %s allow: panic fatal error warn info debug trace, current:", loglevel)
	}
	logger.SetLevel(level)

	fileSrc, err := os.OpenFile(os.DevNull, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return logger, newloggerErr(err.Error())
	}
	var src io.Writer
	if stdout {
		stdoutSrc := os.Stdout
		src = io.MultiWriter(fileSrc, stdoutSrc)
	} else {
		src = io.MultiWriter(fileSrc)
	}
	logger.SetOutput(src)

	logger.SetFormatter(getformatter(format))
	logger.AddHook(getLogHook(format, logdir, logfile))
	return logger, nil
}

func getformatter(f string) logrus.Formatter {
	switch f {
	case "text":
		return &logrus.TextFormatter{
			ForceColors:   true,
			FullTimestamp: true,
		}
	case "text_disable_fulltimestamp":
		return &logrus.TextFormatter{
			ForceColors:   true,
			FullTimestamp: false,
		}
	case "json":
		fallthrough
	default:
		return &logrus.JSONFormatter{}
	}

}

func getLogHook(format, logdir, logfile string) *lfshook.LfsHook {
	logWriter, _ := getLogWriter(logdir, logfile)
	writeMap := lfshook.WriterMap{
		logrus.TraceLevel: logWriter,
		logrus.InfoLevel:  logWriter,
		logrus.FatalLevel: logWriter,
		logrus.DebugLevel: logWriter,
		logrus.WarnLevel:  logWriter,
		logrus.ErrorLevel: logWriter,
		logrus.PanicLevel: logWriter,
	}
	return lfshook.NewHook(writeMap, getformatter(format))
}

func getLogWriter(logdir, logfile string) (*rotatelogs.RotateLogs, error) {
	if _, err := os.Stat(logdir); err != nil {
		if os.IsNotExist(err) {
			err := os.Mkdir(logdir, os.ModePerm)
			if err != nil {
				return nil, newloggerErr("create log dir (%s) error %s", logdir, err.Error())
			}
		}
		return nil, newloggerErr(err.Error())
	}

	filepath := path.Join(logdir, logfile)
	logWriter, err := rotatelogs.New(
		filepath+".%Y%m%d.log",
		rotatelogs.WithLinkName(filepath),
		rotatelogs.WithMaxAge(7*24*time.Hour),
		rotatelogs.WithRotationTime(24*time.Hour),
	)
	if err != nil {
		return nil, newloggerErr(err.Error())
	}
	return logWriter, nil
}
`
