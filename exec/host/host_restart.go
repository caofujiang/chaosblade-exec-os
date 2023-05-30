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

package host

import (
	"context"
	"github.com/chaosblade-io/chaosblade-exec-os/exec/category"
	"github.com/chaosblade-io/chaosblade-spec-go/log"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"os/exec"
)

const HostRestartBin = "chaos_hostRestart"

type HostRestartActionCommandSpec struct {
	spec.BaseExpActionCommandSpec
}

func NewHostRestartActionSpec() spec.ExpActionCommandSpec {
	return &HostRestartActionCommandSpec{
		spec.BaseExpActionCommandSpec{
			ActionMatchers: []spec.ExpFlagSpec{},
			ActionFlags:    []spec.ExpFlagSpec{},
			ActionExecutor: &HostRestartExecutor{},
			ActionExample: `
# Restart local host
./blade create host restart

# Restart remote host: 192.168.56.102
./blade create host restart  --channel ssh --ssh-host 192.168.56.102  --ssh-user root  --install-path /root/chaosblade-1.7.1
`,
			ActionPrograms:   []string{HostRestartBin},
			ActionCategories: []string{category.SystemTime},
		},
	}
}

func (*HostRestartActionCommandSpec) Name() string {
	return "restart"
}

func (*HostRestartActionCommandSpec) Aliases() []string {
	return []string{"r"}
}

func (*HostRestartActionCommandSpec) ShortDesc() string {
	return "Host Restart"
}

func (k *HostRestartActionCommandSpec) LongDesc() string {
	if k.ActionLongDesc != "" {
		return k.ActionLongDesc
	}
	return "Restart host"
}

func (*HostRestartActionCommandSpec) Categories() []string {
	return []string{category.SystemProcess}
}

type HostRestartExecutor struct {
	channel spec.Channel
}

func (sse *HostRestartExecutor) Name() string {
	return "restart"
}

func (sse *HostRestartExecutor) Exec(uid string, ctx context.Context, model *spec.ExpModel) *spec.Response {

	if _, ok := spec.IsDestroy(ctx); ok {
		return spec.ReturnSuccess("destroy restart host success")
	}

	return sse.start(ctx)
}

func (sse *HostRestartExecutor) SetChannel(channel spec.Channel) {
	sse.channel = channel
}

func (sse *HostRestartExecutor) start(ctx context.Context) *spec.Response {

	cmd := exec.Command("reboot")
	_, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := err.Error()
		log.Errorf(ctx, errMsg)
		return spec.ResponseFailWithFlags(spec.ActionNotSupport, errMsg)
	}
	return spec.ReturnSuccess("host reboot success")

}
