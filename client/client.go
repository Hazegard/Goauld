package main

import (
	"Goauld/client/common"
	"Goauld/client/compiler"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"Goauld/client/api"
	_common "Goauld/common"
	"Goauld/common/log"
)

func RewriteArgs(bin string, mode string) []string {
	path := filepath.Dir(os.Args[0])
	if strings.HasSuffix(path, ".exe") {
		path = fmt.Sprintf("%s%s%s.exe", path, string(filepath.Separator), strings.TrimSuffix(bin, fmt.Sprintf("-%s.exe", mode)))
	} else {
		path = fmt.Sprintf("%s%s%s", path, string(filepath.Separator), strings.TrimSuffix(bin, fmt.Sprintf("-%s", mode)))
	}
	args := []string{path, mode}
	args = append(args, os.Args[1:]...)
	return args
}

func PreParseArgs() {
	// Rewrite the command line arguments to include subcommand depending on the argv[0]
	// To allow symlink SCP/SSH required by vscode
	bin := filepath.Base(os.Args[0])
	if bin == "ssh" {
		args := RewriteArgs(bin, "ssh")
		os.Args = args
	}
	if bin == "scp" {
		args := RewriteArgs(bin, "scp")
		os.Args = args
	}

	isVscodeCommand := false
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "ConnectTimeout=") {
			isVscodeCommand = true
			break
		}
	}
	// rewrite the arguments to ssh provided by vscode to match the parameter order we need
	// VSCode append the target in last, but we need it before the optional parameters fed to SSH
	if (isVscodeCommand) && len(os.Args) >= 3 && (bin == "ssh") {
		l := len(os.Args)
		var args []string
		args = append(args, os.Args[0])
		args = append(args, os.Args[1])
		args = append(args, os.Args[l-1])
		args = append(args, os.Args[2:l-1]...)
		os.Args = args
	}

	if len(os.Args) < 2 {
		// Hijack args if empty to show help if no argument is provided
		os.Args = append(os.Args, "--help")
	}
}

func main() {
	// To handle vscode scp check (call scp and checks that the output starts with "usage: scp"
	if len(os.Args) == 1 && filepath.Base(os.Args[0]) == "scp" {
		_ = ExecSCp()
		return
	}
	// Preparsing/reordering of the os.Args
	PreParseArgs()

	kong, cfg, ctx, err := InitConfig()
	if err != nil {
		fmt.Println(err)
		return
	}

	if strings.Fields(ctx.Command())[0] == "compile" {
		os.Args = append([]string{"compile"}, cfg.Compile.Args...)
		//os.Args = os.Args[1:]
		kong, cfg, err := compiler.InitCompilerConfig(APP_NAME, defaultValues)
		if err != nil {
			fmt.Println(err)
			return
		}
		kong.Bind(*cfg)
		err = kong.Run(cfg)
		if err != nil {
			log.Error().Err(err).Msg("error running compiler")
		}
		return
	}

	if cfg.Version {
		if strings.HasPrefix(ctx.Command(), "ssh") {
			ExecuteSystemSSH("-V")
		} else {
			fmt.Println(_common.GetVersion())
		}
		return
	}
	if cfg.GenerateConfig {
		cfg.GenerateConfig = false
		c, err := cfg.GenerateYAMLConfig()
		if err != nil {
			log.Error().Err(err).Msg("error generating the agent config")
			return
		}
		fmt.Println(c)
		return
	}
	httpclient := api.NewAPI(cfg.ServerUrl(), cfg.AccessToken, cfg.Insecure, cfg.AdminToken)
	CheckApiVersion(httpclient)
	kong.Bind(*cfg, httpclient)

	err = kong.Run(httpclient, cfg)
	if err != nil {
		if len(os.Args) > 1 {
			log.Error().Err(err).Str("Mode", kong.Command()).Msg("error running " + common.APP_NAME)
			return
		}
		log.Error().Err(err).Msg("error running " + common.APP_NAME)
	}
}

// CheckApiVersion fetches the server side version and compares it to the client version
// It prints a warning to the user if the versions mismatch
func CheckApiVersion(api *api.API) {
	srvVersion, err := api.Version()
	if err != nil {
		log.Warn().Err(err).Msg("error getting version")
		return
	}
	clientVersion := _common.JsonVersion()
	if srvVersion.Compare(clientVersion) != 0 {
		log.Warn().Str("Server", srvVersion.Version).Str("Client", clientVersion.Version).Msg("version mismatch")
		log.Trace().Str("Server Commit", srvVersion.Commit).Str("Client Commit", clientVersion.Commit).Msg("version mismatch")
		log.Trace().Str("Server Date", srvVersion.Date).Str("Client Date", clientVersion.Date).Msg("version mismatch")
	}
}
