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
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/chaosblade-io/chaosblade-exec-os/exec"
	"github.com/chaosblade-io/chaosblade-exec-os/exec/category"
	"github.com/chaosblade-io/chaosblade-spec-go/log"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	_ "github.com/go-sql-driver/mysql"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
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
					Name:     "downloadUrl",
					Desc:     "download-url, a url string contains script tar package",
					Required: false,
				},
				&spec.ExpFlag{
					Name:     "uploadUrl",
					Desc:     "upload-url, a url string can upload script excute outfile",
					Required: false,
				},
			},
			ActionExecutor: &ScripExecuteExecutor{},
			ActionExample: `
# Add commands to the execute script "
create script execute --file=/Users/admin/tar_file/main11.tar --file-args=aaa:bbb:ccc --downloadUrl=http://10.148.55.113:8080/chaosblade-cps/script/download/host-main-1669186308408.tar --uploadUrl=http://10.148.55.113:8080/chaosblade-cps/script/upload`,
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
	commands := []string{"cat", "rm", "sed", "awk", "rm", "tar"}
	if response, ok := sde.channel.IsAllCommandsAvailable(ctx, commands); !ok {
		return response
	}
	downloadUrl := model.ActionFlags["downloadUrl"]
	uploadUrl := model.ActionFlags["uploadUrl"]
	scriptFile := model.ActionFlags["file"]
	if downloadUrl != "" { //host模式
		if scriptFile != "" {
			_, fileName := filepath.Split(scriptFile)
			scriptFile = "/tmp/" + fileName
		} else {
			scriptFile = "/tmp/" + fmt.Sprintf("%d", time.Now().UnixNano()) + ".tar"
		}
		err := downloadFile(downloadUrl, scriptFile)
		if err != nil {
			log.Errorf(ctx, fmt.Sprintf("download scriptFile  failed  %s", err.Error()))
			return spec.ResponseFailWithFlags(spec.ParameterInvalid, "params downloadUrl", "it is  invalid")
		}
	}
	if scriptFile == "" { //集群模式
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
	if _, ok := spec.IsDestroy(ctx); ok {
		return sde.stop(ctx, scriptFile)
	}
	return sde.start(ctx, scriptFile, fileArgs, uploadUrl, uid)
}

func (sde *ScripExecuteExecutor) start(ctx context.Context, scriptFile, fileArgs, uploadUrl, uid string) *spec.Response {
	// backup file
	response := backScript(ctx, sde.channel, scriptFile)
	if !response.Success {
		return response
	}
	//todo 有需要再放开
	//timeContent, err := ioutil.ReadFile(time)
	//if err != nil {
	//	errInfo = fmt.Sprintf("os.ReadFile:script-time failed  %s", err.Error())
	//}
	//timeResult := string(timeContent)

	//main.tar是一个或者多个文件直接打的tar，外层没有目录，eg: scriptFile="/Users/apple/tar_file/main.tar
	tarDistDir := filepath.Dir(scriptFile) + "/" + fmt.Sprintf("%d", time.Now().UnixNano())
	if response = sde.channel.Run(ctx, "mkdir", fmt.Sprintf(`-p %s`, tarDistDir)); !response.Success {
		sde.stop(ctx, scriptFile)
	}
	if response = sde.channel.Run(ctx, "tar", fmt.Sprintf(`-xvf %s -C  %s`, scriptFile, tarDistDir)); !response.Success {
		sde.stop(ctx, tarDistDir)
	}
	//UnTar(scriptFile, tarDistDir)
	//判断有没有main主文件，没有直接返错误
	scriptMain := tarDistDir + "/main"
	if !exec.CheckFilepathExists(ctx, sde.channel, scriptMain) {
		log.Errorf(ctx, "`%s`,main file is not exist in tar", scriptMain)
		return spec.ResponseFailWithFlags(spec.ParameterInvalid, "main file", scriptMain, "it is not found in tar")
	}
	if response = sde.channel.Run(ctx, "chmod", fmt.Sprintf(`777 "%s"`, scriptMain)); !response.Success {
		sde.stop(ctx, scriptMain)
	}
	//录制script脚本执行过程
	time := "/tmp/" + uid + ".time"
	out := "/tmp/" + uid + ".out"
	if runtime.GOOS == "darwin" {
		scriptMain = "script  -t 2>" + time + " -a " + out + " " + scriptMain
	} else {
		scriptMain = "script  -t 2>" + time + " -a " + out + "  -c  " + "\"" + scriptMain
		fileArgs += "\""
	}
	response = insertContentToScriptByExecute(ctx, sde.channel, scriptMain, fileArgs)
	if !response.Success {
		sde.stop(ctx, scriptMain)
	}
	//os.RemoveAll(tarDistDir)
	var errInfo, errUploadInfo string
	if uploadUrl != "" { //物理主机上传脚本执行过程
		out = "/tmp/" + uid + ".out"
		outContent, err := ioutil.ReadFile(out)
		if err != nil {
			errInfo = fmt.Sprintf("os.ReadFile:script-out failed  %s", err.Error())
		}
		data := make(map[string]string)
		data["uid"] = uid
		data["outputInfo"] = string(outContent)
		err = uploadFile(uploadUrl, data)
		if err != nil {
			errUploadInfo = fmt.Sprintf("uploadFile script-out failed  %s", err.Error())
		}

		var newResult = make(map[string]interface{})
		newResult["errInfo"] = errInfo
		//newResult["errOsExecInfo"] = errOsExecInfo
		newResult["errUploadInfo"] = errUploadInfo
		newResult["outMsg"] = response.Result
		response.Result = newResult
	}
	return response
}

func (sde *ScripExecuteExecutor) stop(ctx context.Context, scriptFile string) *spec.Response {
	return recoverScript(ctx, sde.channel, scriptFile)
}

func (sde *ScripExecuteExecutor) SetChannel(channel spec.Channel) {
	sde.channel = channel
}

func UnTar(srcTar string, dstDir string) (err error) {
	dstDir = path.Clean(dstDir) + string(os.PathSeparator)
	fr, er := os.Open(srcTar)
	if er != nil {
		return er
	}
	defer fr.Close()
	tr := tar.NewReader(fr)
	for hdr, er := tr.Next(); er != io.EOF; hdr, er = tr.Next() {
		if er != nil {
			return er
		}
		fi := hdr.FileInfo()
		// 获取绝对路径
		dstFullPath := dstDir + hdr.Name
		if hdr.Typeflag == tar.TypeDir {
			os.MkdirAll(dstFullPath, fi.Mode().Perm())
			os.Chmod(dstFullPath, fi.Mode().Perm())
		} else {
			os.MkdirAll(path.Dir(dstFullPath), os.ModePerm)
			if er := unTarFile(dstFullPath, tr); er != nil {
				return er
			}
			os.Chmod(dstFullPath, fi.Mode().Perm())
		}
	}
	return nil
}
func unTarFile(dstFile string, tr *tar.Reader) error {
	fw, er := os.Create(dstFile)
	if er != nil {
		return er
	}
	defer fw.Close()
	_, er = io.Copy(fw, tr)
	if er != nil {
		return er
	}
	return nil
}

func uploadFile(url string, data map[string]string) error {
	bytesData, _ := json.Marshal(data)
	res, err := http.Post(url, "application/json;charset=utf-8", bytes.NewBuffer(bytesData))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	return nil
}

func downloadFile(url string, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	// copy stream
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}
