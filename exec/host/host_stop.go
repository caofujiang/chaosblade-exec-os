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

const HostStopBin = "chaos_hostStop"

type HostStopActionCommandSpec struct {
	spec.BaseExpActionCommandSpec
}

func NewHostStopActionSpec() spec.ExpActionCommandSpec {
	return &HostStopActionCommandSpec{
		spec.BaseExpActionCommandSpec{
			ActionMatchers: []spec.ExpFlagSpec{},
			ActionFlags: []spec.ExpFlagSpec{
				&spec.ExpFlag{
					Name:     "time",
					Desc:     "restart after time, unit: now、minute such as +1、time such as 20:35、now",
					Required: false,
				},
				&spec.ExpFlag{
					Name:     "forced",
					Desc:     "force shutdown , such as -f  ",
					Required: false,
				},
			},
			ActionExecutor: &HostStopExecutor{},
			ActionExample: `
# Stop local host
./blade create host stop

# Stop  host after time
./blade create host stop --time=22:00  //在当天晚上10点关机
./blade create host stop --time=2   //在 2 分钟后关机
./blade create host stop --time=now  //立即关机


# Stop  host forced shutdown
./blade create host stop  --time= -f  //强制关机

# Stop remote host: 192.168.56.102
./blade create host stop  --channel ssh --ssh-host 192.168.56.102  --ssh-user root  --install-path /root/chaosblade-1.7.1
`,
			ActionPrograms:   []string{HostStopBin},
			ActionCategories: []string{category.SystemTime},
		},
	}
}

func (*HostStopActionCommandSpec) Name() string {
	return "stop"
}

func (*HostStopActionCommandSpec) Aliases() []string {
	return []string{"s"}
}

func (*HostStopActionCommandSpec) ShortDesc() string {
	return "Host Stop"
}

func (k *HostStopActionCommandSpec) LongDesc() string {
	if k.ActionLongDesc != "" {
		return k.ActionLongDesc
	}
	return "Stop host"
}

func (*HostStopActionCommandSpec) Categories() []string {
	return []string{category.SystemProcess}
}

type HostStopExecutor struct {
	channel spec.Channel
}

func (sse *HostStopExecutor) Name() string {
	return "stop"
}

func (sse *HostStopExecutor) Exec(uid string, ctx context.Context, model *spec.ExpModel) *spec.Response {

	if _, ok := spec.IsDestroy(ctx); ok {
		return spec.ReturnSuccess(uid)
	}
	stopTime := model.ActionFlags["time"]
	if stopTime != "" {
		return sse.channel.Run(ctx, "shutdown", fmt.Sprintf("-%s %s", "h", stopTime))
	} else {
		return sse.channel.Run(ctx, "poweroff", "")
	}
}

func (sse *HostStopExecutor) SetChannel(channel spec.Channel) {
	sse.channel = channel
}
