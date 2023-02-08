package http

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/chaosblade-io/chaosblade-exec-os/exec"
	"github.com/chaosblade-io/chaosblade-exec-os/exec/category"
	"github.com/chaosblade-io/chaosblade-spec-go/log"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
)

const Http2DelayBin = "chaos_httpdelay"

type DelayHttpActionCommandSpec struct {
	spec.BaseExpActionCommandSpec
}

func NewDelayHttpActionCommandSpec() spec.ExpActionCommandSpec {
	return &DelayHttpActionCommandSpec{
		spec.BaseExpActionCommandSpec{
			ActionMatchers: []spec.ExpFlagSpec{
				&spec.ExpFlag{
					Name:                  "url",
					Desc:                  "The Url of the target http2",
					Required:              true,
					RequiredWhenDestroyed: true,
				},
			},
			ActionFlags: []spec.ExpFlagSpec{
				&spec.ExpFlag{
					Name:     "time",
					Desc:     "sleep time, unit is millisecond",
					Required: true,
				},
			},
			ActionExample: `
# Create a http2 10000(10s) delay experiment "
blade create http2 delay --url https://www.taobao.com --time 10000`,
			ActionExecutor:   &HttpDelayExecutor{},
			ActionCategories: []string{category.SystemHttp},
		},
	}
}

func (*DelayHttpActionCommandSpec) Name() string {
	return "delay"
}

func (*DelayHttpActionCommandSpec) Aliases() []string {
	return []string{"d"}
}

func (*DelayHttpActionCommandSpec) ShortDesc() string {
	return "delay url"
}

func (impl *DelayHttpActionCommandSpec) LongDesc() string {
	if impl.ActionLongDesc != "" {
		return impl.ActionLongDesc
	}
	return "delay http2 by url"
}

// HttpDelayExecutor for action
type HttpDelayExecutor struct {
	channel spec.Channel
}

func (*HttpDelayExecutor) Name() string {
	return "delay"
}

func (impl *HttpDelayExecutor) SetChannel(channel spec.Channel) {
	impl.channel = channel
}

func (impl *HttpDelayExecutor) Exec(uid string, ctx context.Context, model *spec.ExpModel) *spec.Response {
	if impl.channel == nil {
		return spec.ResponseFailWithFlags(spec.ChannelNil)
	}
	if _, ok := spec.IsDestroy(ctx); ok {
		return impl.stop(ctx, uid)
	}
	urlStr := model.ActionFlags["url"]
	if urlStr == "" {
		log.Errorf(ctx, "url-is-nil")
		return spec.ResponseFailWithFlags(spec.ParameterIllegal, "url")
	}
	if !strings.Contains(urlStr, "https://") {
		log.Errorf(ctx, "url is not unsupported protocol scheme")
		return spec.ResponseFailWithFlags(spec.ParameterLess, "url")
	}

	t := model.ActionFlags["time"]
	if t == "" {
		log.Errorf(ctx, "time-is-nil")
		return spec.ResponseFailWithFlags(spec.ParameterLess, "time")
	}
	t1, err := strconv.Atoi(t)
	if err != nil {
		log.Errorf(ctx, "time %v it must be a positive integer", t1)
		return spec.ResponseFailWithFlags(spec.ParameterInvalid, "time", t1, "time must be a positive integer")
	}

	return impl.start(ctx, urlStr, t1)
}

func (impl *HttpDelayExecutor) start(ctx context.Context, url string, t int) *spec.Response {
	time.Sleep(time.Duration(t) * time.Millisecond)
	return impl.channel.Run(ctx, "curl", url)
}

func (impl *HttpDelayExecutor) stop(ctx context.Context, uid string) *spec.Response {
	//ctx = context.WithValue(ctx, "bin", Http2DelayBin)
	return exec.Destroy(ctx, impl.channel, uid)
}
