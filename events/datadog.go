package events

import (
	"bytes"
	"text/template"

	"github.com/PagerDuty/godspeed"

	"github.com/mostlygeek/reaper/aws"
	"github.com/mostlygeek/reaper/http"
)

// implements EventReporter, sends events and statistics to DataDog
// uses godspeed, requires dd-agent running
type DataDog struct {
	eventTemplate template.Template
}

// TODO: make this async?
// TODO: not recreate godspeed
func (d DataDog) NewEvent(title string, text string, fields map[string]string, tags []string) {
	g, err := godspeed.NewDefault()
	if err != nil {

	}
	defer g.Conn.Close()
	err = g.Event(title, text, fields, tags)
	if err != nil {

	}
}

func (d DataDog) NewStatistic(name string, value float64, tags []string) {
	g, err := godspeed.NewDefault()
	if err != nil {
	}
	defer g.Conn.Close()
	err = g.Gauge(name, value, tags)
	if err != nil {
	}
}

var funcMap = template.FuncMap{
	"MakeTerminateLink": http.MakeTerminateLink,
	"MakeIgnoreLink":    http.MakeIgnoreLink,
	"MakeWhitelistLink": http.MakeWhitelistLink,
	"MakeStopLink":      http.MakeStopLink,
	"MakeForceStopLink": http.MakeForceStopLink,
}

func (d DataDog) NewReapableASGEvent(a *aws.AutoScalingGroup) {
	t := template.Must(template.New("reapable").Funcs(funcMap).Parse(reapableASGTemplateDataDog))
	buf := bytes.NewBuffer(nil)

	data := ASGEventData{
		ASG:    a,
		Config: Conf,
	}

	err := t.Execute(buf, data)
	if err != nil {
	}

	d.NewEvent("Reapable ASG Discovered", string(buf.Bytes()), nil, nil)
}

func (d DataDog) NewReapableInstanceEvent(i *aws.Instance) {
	t := template.Must(template.New("reapable").Funcs(funcMap).Parse(reapableInstanceTemplateDataDog))
	buf := bytes.NewBuffer(nil)

	data := InstanceEventData{
		Instance: i,
		Config:   Conf,
	}

	err := t.Execute(buf, data)
	if err != nil {
	}

	d.NewEvent("Reapable Instance Discovered", string(buf.Bytes()), nil, nil)
}

const reapableInstanceTemplateDataDog = `%%%
Reaper has discovered an instance qualified as reapable: {{if .Instance.Name}}"{{.Instance.Name}}" {{end}}[{{.Instance.Id}}]({{.Instance.AWSConsoleURL}}) in region: [{{.Instance.Region}}](https://{{.Instance.Region}}.console.aws.amazon.com/ec2/v2/home?region={{.Instance.Region}}).\n
{{if .Instance.Owned}}Owned by {{.Instance.Owner}}.\n{{end}}
State: {{.Instance.State}}.\n
{{ if .Instance.AWSConsoleURL}}{{.Instance.AWSConsoleURL}}\n{{end}}
[AWS Console URL]({{.Instance.AWSConsoleURL}})\n
[Whitelist this instance.]({{ MakeWhitelistLink .Config.TokenSecret .Config.HTTPApiURL .Instance.Region .Instance.Id }})
[Stop this instance.]({{ MakeStopLink .Config.TokenSecret .Config.HTTPApiURL .Instance.Region .Instance.Id }})
[Terminate this instance.]({{ MakeTerminateLink .Config.TokenSecret .Config.HTTPApiURL .Instance.Region .Instance.Id }})
%%%`

const reapableASGTemplateDataDog = `%%%
Reaper has discovered an ASG qualified as reapable: [{{.ASG.Id}}]({{.ASG.AWSConsoleURL}}) in region: [{{.ASG.Region}}](https://{{.ASG.Region}}.console.aws.amazon.com/ec2/v2/home?region={{.ASG.Region}}).\n
{{if .ASG.Owned}}Owned by {{.ASG.Owner}}.\n{{end}}
{{ if .ASG.AWSConsoleURL}}{{.ASG.AWSConsoleURL}}\n{{end}}
[AWS Console URL]({{.ASG.AWSConsoleURL}})\n
[Whitelist this ASG.]({{ MakeWhitelistLink .Config.TokenSecret .Config.HTTPApiURL .ASG.Region .ASG.Id }})
[Terminate this ASG.]({{ MakeTerminateLink .Config.TokenSecret .Config.HTTPApiURL .ASG.Region .ASG.Id }})\n
[Scale this ASG to 0 instances]({{ MakeStopLink .Config.TokenSecret .Config.HTTPApiURL .ASG.Region .ASG.Id }})
[Force scale this ASG to 0 instances (changes minimum)]({{ MakeForceStopLink .Config.TokenSecret .Config.HTTPApiURL .ASG.Region .ASG.Id }})
%%%`
