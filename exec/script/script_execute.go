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
	"github.com/chaosblade-io/chaosblade-exec-os/exec"
	"github.com/chaosblade-io/chaosblade-exec-os/exec/category"
	"github.com/chaosblade-io/chaosblade-spec-go/log"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"strings"
)

type  ScripExecuteActionCommand struct {
	spec.BaseExpActionCommandSpec
}

func NewScripExecuteActionCommand() spec.ExpActionCommandSpec {
	return & ScripExecuteActionCommand{
		spec.BaseExpActionCommandSpec{
			ActionMatchers: []spec.ExpFlagSpec{},
			ActionFlags: []spec.ExpFlagSpec{
				&spec.ExpFlag{
					Name:     "file-args",
					Desc:     "file-args, a string separated by -",
					Required: true,
				},
			},
			ActionExecutor: & ScripExecuteExecutor{},
			ActionExample: `
# Add commands to the execute script "
blade create script execute --file test.sh --file-args this-is-file-args-string`,
			ActionCategories: []string{category.SystemScript},
		},
	}
}

func (* ScripExecuteActionCommand) Name() string {
	return "execute"
}

func (* ScripExecuteActionCommand) Aliases() []string {
	return []string{}
}

func (* ScripExecuteActionCommand) ShortDesc() string {
	return "Script execute"
}

func (s * ScripExecuteActionCommand) LongDesc() string {
	if s.ActionLongDesc != "" {
		return s.ActionLongDesc
	}
	return "Execute script"
}

type  ScripExecuteExecutor struct {
	channel spec.Channel
}

func (* ScripExecuteExecutor) Name() string {
	return "execute"
}

func (sde * ScripExecuteExecutor) Exec(uid string, ctx context.Context, model *spec.ExpModel) *spec.Response {
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
	if fileArgs!=""{
		ret := strings.Split(fileArgs, "-")
		fileArgs=strings.Join(ret," ")
	}
	if _, ok := spec.IsDestroy(ctx); ok {
		return sde.stop(ctx, scriptFile)
	}
	return sde.start(ctx, scriptFile,fileArgs)
}

func (sde * ScripExecuteExecutor) start(ctx context.Context, scriptFile ,fileArgs string) *spec.Response {
	// backup file
	response := backScript(ctx, sde.channel, scriptFile)
	if !response.Success {
		return response
	}

	response = insertContentToScriptByExecute(ctx, sde.channel,scriptFile,fileArgs)
	if !response.Success {
		sde.stop(ctx, scriptFile)
	}
	return response
}

func (sde * ScripExecuteExecutor) stop(ctx context.Context, scriptFile string) *spec.Response {
	return recoverScript(ctx, sde.channel, scriptFile)
}

func (sde * ScripExecuteExecutor) SetChannel(channel spec.Channel) {
	sde.channel = channel
}
