/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package script

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/chaosblade-io/chaosblade-exec-os/exec"
	"github.com/chaosblade-io/chaosblade-exec-os/exec/category"
	"github.com/chaosblade-io/chaosblade-spec-go/log"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"runtime"
	"strings"
)

type ScripExecuteActionCommand struct {
	spec.BaseExpActionCommandSpec
}

func NewScripExecuteActionCommand() spec.ExpActionCommandSpec {
	return &ScripExecuteActionCommand{
		spec.BaseExpActionCommandSpec{
			ActionMatchers: []spec.ExpFlagSpec{},
			ActionFlags: []spec.ExpFlagSpec{
				&spec.ExpFlag{
					Name:     "file-args",
					Desc:     "file-args, a string separated by :",
					Required: true,
				},
				&spec.ExpFlag{
					Name:     "dsn",
					Desc:     "dsn, a string contains db connetion info",
					Required: false,
				},
			},
			ActionExecutor: &ScripExecuteExecutor{},
			ActionExample: `
# Add commands to the execute script "
blade create script execute --file test.sh --file-args this:is:file:args:string --dsn root:Spx#123456@tcp(10.148.55.116:3306)/blade_ops`,
			ActionCategories: []string{category.SystemScript},
		},
	}
}

func (*ScripExecuteActionCommand) Name() string {
	return "execute"
}

func (*ScripExecuteActionCommand) Aliases() []string {
	return []string{}
}

func (*ScripExecuteActionCommand) ShortDesc() string {
	return "Script execute"
}

func (s *ScripExecuteActionCommand) LongDesc() string {
	if s.ActionLongDesc != "" {
		return s.ActionLongDesc
	}
	return "Execute script"
}

type ScripExecuteExecutor struct {
	channel spec.Channel
}

func (*ScripExecuteExecutor) Name() string {
	return "execute"
}

func (sde *ScripExecuteExecutor) Exec(uid string, ctx context.Context, model *spec.ExpModel) *spec.Response {
	commands := []string{"cat", "rm", "sed", "awk", "rm"}
	if response, ok := sde.channel.IsAllCommandsAvailable(ctx, commands); !ok {
		return response
	}
	scriptFile := model.ActionFlags["file"]
	if scriptFile == "" {
		log.Errorf(ctx, "file is nil")
		return spec.ResponseFailWithFlags(spec.ParameterLess, "file")
	}
	if !exec.CheckFilepathExists(ctx, sde.channel, scriptFile) {
		log.Errorf(ctx, "`%s`, file is invalid. it not found", scriptFile)
		return spec.ResponseFailWithFlags(spec.ParameterInvalid, "file", scriptFile, "it is not found")
	}
	fileArgs := model.ActionFlags["file-args"]
	if fileArgs != "" {
		ret := strings.Split(fileArgs, ":")
		fileArgs = strings.Join(ret, " ")
	}

	dsn := model.ActionFlags["dsn"]

	if _, ok := spec.IsDestroy(ctx); ok {
		return sde.stop(ctx, scriptFile)
	}
	return sde.start(ctx, scriptFile, fileArgs, dsn, uid)
}

func (sde *ScripExecuteExecutor) start(ctx context.Context, scriptFile, fileArgs, dsn, uid string) *spec.Response {
	// backup file
	response := backScript(ctx, sde.channel, scriptFile)
	if !response.Success {
		return response
	}
	//录制script脚本执行过程
	time := scriptFile + ".time"
	out := scriptFile + ".out"
	if runtime.GOOS == "darwin" {
		scriptFile = "script  -t 2>" + time + " -a " + out + " " + scriptFile
	} else {
		scriptFile = "script  -t 2>" + time + " -a " + out + "  -c  " + "\"" + scriptFile
		fileArgs += "\""
	}
	response = insertContentToScriptByExecute(ctx, sde.channel, scriptFile, fileArgs)
	if !response.Success {
		sde.stop(ctx, scriptFile)
	}

	var errInfo, errMysqlInfo, errMysqlExecInfo string
	//todo 有需要再放开
	//timeContent, err := ioutil.ReadFile(time)
	//if err != nil {
	//	errInfo = fmt.Sprintf("os.ReadFile:script-time failed  %s", err.Error())
	//}
	//timeResult := string(timeContent)
	outContent, err := ioutil.ReadFile(out)
	if err != nil {
		errInfo = fmt.Sprintf("os.ReadFile:script-out failed  %s", err.Error())
	}
	outResult := string(outContent)
	//录制文件存放到mysql
	//dsn = "root:Spx#123456@tcp(10.148.55.116:3306)/blade_ops"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		errMysqlInfo = fmt.Sprintf("open mysql failed  %s", err.Error())
	}
	defer db.Close()
	_, err = db.Exec("update t_chaos_experiment_flow_node_record set output_info =? where result = ? ", outResult, uid)
	if err != nil {
		errMysqlExecInfo = fmt.Sprintf("mysql exec failed,%s", err.Error())
	}

	var newResult = make(map[string]interface{})
	newResult["errInfo"] = errInfo
	newResult["errMysqlInfo"] = errMysqlInfo
	newResult["errMysqlExecInfo"] = errMysqlExecInfo
	newResult["outMsg"] = response.Result
	response.Result = newResult
	return response
}

func (sde *ScripExecuteExecutor) stop(ctx context.Context, scriptFile string) *spec.Response {
	return recoverScript(ctx, sde.channel, scriptFile)
}

func (sde *ScripExecuteExecutor) SetChannel(channel spec.Channel) {
	sde.channel = channel
}
