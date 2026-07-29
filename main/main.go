// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/cli/upgrade"
	"github.com/aliyun/aliyun-cli/v3/cliext/acrutil"
	"github.com/aliyun/aliyun-cli/v3/cliext/agentbay"
	"github.com/aliyun/aliyun-cli/v3/cliext/appmanagerutil"
	"github.com/aliyun/aliyun-cli/v3/cliext/cms2"
	"github.com/aliyun/aliyun-cli/v3/cliext/codeup"
	"github.com/aliyun/aliyun-cli/v3/cliext/computenestutil"
	"github.com/aliyun/aliyun-cli/v3/cliext/esacli"
	"github.com/aliyun/aliyun-cli/v3/cliext/flowcli"
	"github.com/aliyun/aliyun-cli/v3/cliext/iact3"
	"github.com/aliyun/aliyun-cli/v3/cliext/kmscli"
	"github.com/aliyun/aliyun-cli/v3/cliext/lindormcli"
	"github.com/aliyun/aliyun-cli/v3/cliext/maxc"
	"github.com/aliyun/aliyun-cli/v3/cliext/mseutil"
	"github.com/aliyun/aliyun-cli/v3/cliext/ossutil"
	"github.com/aliyun/aliyun-cli/v3/cliext/otsutil"
	"github.com/aliyun/aliyun-cli/v3/cliext/rostran"
	"github.com/aliyun/aliyun-cli/v3/cliext/saectl"
	"github.com/aliyun/aliyun-cli/v3/cliext/sparksubmit"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/export"
	go_migrate "github.com/aliyun/aliyun-cli/v3/go-migrate"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/mcpproxy"
	"github.com/aliyun/aliyun-cli/v3/mock"
	"github.com/aliyun/aliyun-cli/v3/openapi"
	"github.com/aliyun/aliyun-cli/v3/oss/lib"
	sysmock "github.com/aliyun/aliyun-cli/v3/sysconfig/mock"
)

var (
	newStdoutWriter = cli.DefaultStdoutWriter
	newStderrWriter = cli.DefaultStderrWriter
	exit            = cli.Exit
)

func Main(args []string) {
	stdout := newStdoutWriter()
	stderr := newStderrWriter()

	if len(args) > 0 && args[0] == "__e2e-batch-dryrun" {
		exit(runE2EBatchDryRun(args[1:], os.Stdin, stdout, stderr))
		return
	}

	if sysmock.FirstCommandToken(args) != "mock" {
		result := sysmock.Intercept(sysmock.Options{
			Args:     args,
			Stdout:   stdout,
			Stderr:   stderr,
			MockPath: sysmock.ResolvePath(config.GetConfigPath),
		})
		if result.Handled {
			exit(result.ExitCode)
			return
		}
	}

	// load current configuration
	profile, err := config.LoadOrCreateDefaultProfile()
	if err != nil {
		cli.Errorf(stderr, "ERROR: load current configuration failed %s", err)
		return
	}

	// set language with current profile
	i18n.SetLanguage(profile.Language)

	rootCmd := newRootCommand(profile, stdout)

	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(rootCmd)
	ctx.SetCompletion(cli.ParseCompletionForShell())
	ctx.SetInConfigureMode(openapi.DetectInConfigureMode(ctx.Flags()))
	// use http force, current use in oss bridge
	insecure, _ := ParseInSecure(args)
	ctx.SetInsecure(insecure)

	if os.Getenv("GENERATE_METADATA") == "YES" {
		generateMetadata(rootCmd)
	} else {
		rootCmd.Execute(ctx, args)
	}
}

func newRootCommand(profile config.Profile, stdout io.Writer) *cli.Command {
	return newRootCommandWithCommando(profile, openapi.NewCommando(stdout, profile))
}

func newRootCommandWithCommando(profile config.Profile, commando *openapi.Commando) *cli.Command {
	// create root command
	rootCmd := &cli.Command{
		Name:              "aliyun",
		Short:             i18n.T("Alibaba Cloud Command Line Interface Version "+cli.Version, "阿里云CLI命令行工具 "+cli.Version),
		Usage:             "aliyun <product> <operation> [--parameter1 value1 --parameter2 value2 ...]",
		Sample:            "aliyun ecs DescribeRegions",
		EnableUnknownFlag: true,
	}

	// add default flags
	config.AddFlags(rootCmd.Flags())
	openapi.AddFlags(rootCmd.Flags())

	// new open api commando to process rootCmd
	commando.InitWithCommand(rootCmd)

	rootCmd.AddSubCommand(config.NewConfigureCommand())
	// list-supported-pricing-apis: enumerate every OpenAPI that supports --estimate-cost
	rootCmd.AddSubCommand(openapi.NewListSupportedPricingApisCommand())
	// oss old version, duplicate with ossutil, will remove in future
	rootCmd.AddSubCommand(lib.NewOssCommand())
	rootCmd.AddSubCommand(cli.NewVersionCommand())
	rootCmd.AddSubCommand(cli.NewAutoCompleteCommand())
	// mcp proxy command
	rootCmd.AddSubCommand(mcpproxy.NewMCPProxyCommand())
	// go v1 to v2 migrate command
	rootCmd.AddSubCommand(go_migrate.NewGoMigrateCommand())
	// new oss command
	rootCmd.AddSubCommand(ossutil.NewOssutilCommand())
	// AgentBay command
	rootCmd.AddSubCommand(agentbay.NewAgentBayCommand())
	// tablestore command
	rootCmd.AddSubCommand(otsutil.NewOtsutilCommand())
	// EMR Serverless spark-submit command
	rootCmd.AddSubCommand(sparksubmit.NewSparkSubmitCommand())
	// kmscli command
	rootCmd.AddSubCommand(kmscli.NewKmscliCommand())
	// lindorm command
	rootCmd.AddSubCommand(lindormcli.NewLindormCliCommand())
	// mseutil command
	rootCmd.AddSubCommand(mseutil.NewMseutilCommand())
	// acr command
	rootCmd.AddSubCommand(acrutil.NewAcrutilCommand())
	// codeup command
	rootCmd.AddSubCommand(codeup.NewCodeupCliCommand())
	// sae command
	rootCmd.AddSubCommand(saectl.NewSaectlCommand())
	// appmanager command
	rootCmd.AddSubCommand(appmanagerutil.NewAppManagerCommand())
	// computenest command
	rootCmd.AddSubCommand(computenestutil.NewComputenestCommand())
	// esa-cli command
	rootCmd.AddSubCommand(esacli.NewEsacliCommand())
	// flow-cli command (云效 Flow)
	rootCmd.AddSubCommand(flowcli.NewFlowcliCommand())
	// cms2 command
	rootCmd.AddSubCommand(cms2.NewCms2Command())
	// maxc command
	rootCmd.AddSubCommand(maxc.NewMaxcCommand())
	// iact3 command
	rootCmd.AddSubCommand(iact3.NewIact3Command())
	// rostran command
	rootCmd.AddSubCommand(rostran.NewRostranCommand())
	// plugin command
	rootCmd.AddSubCommand(plugin.NewPluginCommand())
	// upgrade command
	rootCmd.AddSubCommand(upgrade.NewUpgradeCommand())
	// mock command
	rootCmd.AddSubCommand(mock.NewMockCommand(config.GetConfigPath))

	plugin.RegisterReservedTopLevelCommands(rootCmd.SubCommandNames())

	return rootCmd
}

type e2eBatchRequest struct {
	ID   string   `json:"id"`
	Argv []string `json:"argv"`
}

type e2eBatchResponse struct {
	ID              string `json:"id"`
	ReturnCode      int    `json:"returncode"`
	Stdout          string `json:"stdout"`
	Stderr          string `json:"stderr"`
	Error           string `json:"error,omitempty"`
	ElapsedMS       int64  `json:"elapsed_ms"`
	StdoutTruncated bool   `json:"stdout_truncated"`
	StderrTruncated bool   `json:"stderr_truncated"`
}

func runE2EBatchDryRun(args []string, in io.Reader, out io.Writer, errOut io.Writer) int {
	if len(args) == 1 && args[0] == "--probe" {
		cli.Println(out, "ok")
		return 0
	}
	encoder := json.NewEncoder(out)
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		resp := executeE2EBatchLine(line)
		if err := encoder.Encode(resp); err != nil {
			cli.Errorf(errOut, "ERROR: write batch response failed: %v\n", err)
			return 1
		}
	}
	if err := scanner.Err(); err != nil {
		cli.Errorf(errOut, "ERROR: read batch request failed: %v\n", err)
		return 1
	}
	return 0
}

func executeE2EBatchLine(line string) e2eBatchResponse {
	var req e2eBatchRequest
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		return e2eBatchResponse{ReturnCode: 2, Error: "invalid_json", Stderr: err.Error()}
	}
	if req.ID == "" {
		req.ID = strings.Join(req.Argv, " ")
	}
	resp := e2eBatchResponse{ID: req.ID}
	if !hasE2EDryRunFlag(req.Argv) {
		resp.ReturnCode = 2
		resp.Error = "missing_dry_run"
		resp.Stderr = "batch dry-run requires --dryrun, --cli-dry-run, or --cli-dry-run-json"
		return resp
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	exitCode := 0
	start := time.Now()
	runSingleE2EDryRun(req.Argv, stdout, stderr, func(code int) {
		exitCode = code
	})
	resp.ReturnCode = exitCode
	resp.Stdout = stdout.String()
	resp.Stderr = stderr.String()
	resp.ElapsedMS = time.Since(start).Milliseconds()
	return resp
}

func runSingleE2EDryRun(args []string, stdout io.Writer, stderr io.Writer, exitHook func(int)) {
	oldStdoutWriter := newStdoutWriter
	oldStderrWriter := newStderrWriter
	oldExit := exit
	oldArgs := os.Args
	defer func() {
		newStdoutWriter = oldStdoutWriter
		newStderrWriter = oldStderrWriter
		exit = oldExit
		os.Args = oldArgs
	}()
	newStdoutWriter = func() io.Writer { return stdout }
	newStderrWriter = func() io.Writer { return stderr }
	exit = exitHook
	os.Args = append([]string{"aliyun"}, args...)
	cli.WithExitHook(exitHook, func() {
		Main(args)
	})
}

func hasE2EDryRunFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--dryrun" || arg == "--cli-dry-run" || arg == "--cli-dry-run-json" {
			return true
		}
		if strings.HasPrefix(arg, "--dryrun=") {
			if isTruthyE2EFlagValue(strings.TrimPrefix(arg, "--dryrun=")) {
				return true
			}
		}
		if strings.HasPrefix(arg, "--cli-dry-run=") {
			if isTruthyE2EFlagValue(strings.TrimPrefix(arg, "--cli-dry-run=")) {
				return true
			}
		}
		if strings.HasPrefix(arg, "--cli-dry-run-json=") {
			if isTruthyE2EFlagValue(strings.TrimPrefix(arg, "--cli-dry-run-json=")) {
				return true
			}
		}
	}
	return false
}

func isTruthyE2EFlagValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func ParseInSecure(args []string) (bool, interface{}) {
	// check has insecure flag
	for _, arg := range args {
		if arg == "--insecure" {
			return true, nil
		}
	}
	return false, nil
}

func main() {
	Main(os.Args[1:])
}

func dumpFiles(fs embed.FS, filePath string, outputDir string) {
	filePath = strings.TrimPrefix(filePath, "./")

	entries, err := fs.ReadDir(filePath)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	for _, entry := range entries {
		entryPath := path.Join(filePath, entry.Name())
		if entry.IsDir() {
			dumpFiles(fs, entryPath, outputDir)
		} else {
			content, err := fs.ReadFile(entryPath)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			targetPath := path.Join(outputDir, entryPath)
			fmt.Println("copy file from " + entryPath + " to " + targetPath)
			_, err = os.Stat(path.Dir(targetPath))
			if os.IsNotExist(err) {
				err = os.MkdirAll(path.Dir(targetPath), 0755)
				if err != nil {
					fmt.Println(err.Error())
					return
				}
			}
			err = os.WriteFile(targetPath, content, 0666)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
		}
	}
}

func generateMetadata(rootCmd *cli.Command) {
	metadata := make(map[string]*cli.Metadata)
	rootCmd.GetMetadata(metadata)
	b, _ := json.MarshalIndent(metadata, "", "  ")
	cwd, _ := os.Getwd()
	targetDir := cwd + "/cli-metadata"
	_, err := os.Stat(targetDir)
	if os.IsNotExist(err) {
		err := os.Mkdir(targetDir, 0755)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}

	targetPath := targetDir + "/commands.json"
	err = os.WriteFile(targetPath, b, 0666)
	if err != nil {
		fmt.Println(err.Error())
	}

	versionPath := targetDir + "/version"
	os.WriteFile(versionPath, []byte(cli.Version), 0666)

	if err := export.LegacyExportMetadata(targetDir); err != nil {
		fmt.Println(err.Error())
	}
}
