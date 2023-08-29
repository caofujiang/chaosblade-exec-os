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
	"fmt"
	"github.com/chaosblade-io/chaosblade-exec-os/exec/category"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
)

const HostRestartBin = "chaos_hostRestart"

type HostRestartActionCommandSpec struct {
	spec.BaseExpActionCommandSpec
}

func NewHostRestartActionSpec() spec.ExpActionCommandSpec {
	return &HostRestartActionCommandSpec{
		spec.BaseExpActionCommandSpec{
			ActionMatchers: []spec.ExpFlagSpec{},
			ActionFlags: []spec.ExpFlagSpec{
				&spec.ExpFlag{
					Name:     "time",
					Desc:     "restart after time, unit: now、minute such as 1、time such as 20:35",
					Required: false,
				},
			},
			ActionExecutor: &HostRestartExecutor{},
			ActionExample: `
# Restart remote host
./blade create host restart now
./blade create host restart  1

# Restart remote host: 192.168.56.102
./blade create host restart  --channel ssh --ssh-host 192.168.56.102  --ssh-user root  --install-path /root/chaosblade-1.7.3
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
		return spec.ReturnSuccess(uid)
	}
	restartTime := model.ActionFlags["time"]

	if restartTime != "" {
		return sse.channel.Run(ctx, "shutdown", fmt.Sprintf("-%s %s", "r", restartTime))
	} else {
		return sse.channel.Run(ctx, "reboot", "")
	}
}

func (sse *HostRestartExecutor) SetChannel(channel spec.Channel) {
	sse.channel = channel
}
