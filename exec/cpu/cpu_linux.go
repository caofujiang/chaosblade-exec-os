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

package cpu

import (
	"context"
	"github.com/chaosblade-io/chaosblade-exec-os/exec"
	"github.com/chaosblade-io/chaosblade-spec-go/channel"
	"github.com/chaosblade-io/chaosblade-spec-go/log"
	"github.com/containerd/cgroups"
	"github.com/shirou/gopsutil/cpu"
	"strconv"
	"time"
)

func getUsed(ctx context.Context, percpu bool, cpuIndex int) float64 {

	pid := ctx.Value(channel.NSTargetFlagName)
	cpuCount := ctx.Value("cpuCount").(int)

	if pid != nil {
		p, err := strconv.Atoi(pid.(string))
		if err != nil {
			log.Errorf(ctx, "get cpu usage fail, %s", err.Error())
			return safeUsedPercent
		}

		cgroupRoot := ctx.Value("cgroup-root")
		if cgroupRoot == "" {
			cgroupRoot = "/sys/fs/cgroup/"
		}

		log.Debugf(ctx, "get cpu useage by cgroup, root path: %s", cgroupRoot)

		cgroup, err := cgroups.Load(exec.Hierarchy(cgroupRoot.(string)), exec.PidPath(p))
		if err != nil {
			log.Errorf(ctx, "get cpu usage fail, %s", err.Error())
			return safeUsedPercent
		}

		stats, err := cgroup.Stat(cgroups.IgnoreNotExist)
		if err != nil {
			log.Errorf(ctx, "get cpu usage fail, %s", err.Error())
			return safeUsedPercent
		}
		pre := float64(stats.CPU.Usage.Total) / float64(time.Second)
		time.Sleep(time.Second)
		nextStats, err := cgroup.Stat(cgroups.IgnoreNotExist)
		if err != nil {
			log.Errorf(ctx, "get cpu usage fail, %s", err.Error())
			return safeUsedPercent
		}
		next := float64(nextStats.CPU.Usage.Total) / float64(time.Second)
		return ((next - pre) * 100) / float64(cpuCount)
	}

	totalCpuPercent, err := cpu.Percent(time.Second, percpu)
	if err != nil {
		log.Errorf(ctx, "get cpu usage fail, %s", err.Error())
		return safeUsedPercent
	}
	if percpu {
		if cpuIndex >= len(totalCpuPercent) {
			log.Errorf(ctx, "illegal cpu index %d", cpuIndex)
			return safeUsedPercent
		}
		return totalCpuPercent[cpuIndex]
	}
	if len(totalCpuPercent) == 0 {
		log.Errorf(ctx, "get cpu usage fail, empty result")
		return safeUsedPercent
	}
	return totalCpuPercent[0]
}
