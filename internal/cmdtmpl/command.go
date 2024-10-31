package cmdtmpl

import (
	"bytes"
	"context"
	"strings"
	"text/template"

	"github.com/telekom-mms/oc-daemon/internal/execs"
)

// CommandTemplates are command templates.
type CommandTemplates struct {
	templates *template.Template
}

func NewCommandTemplates(templates string) *CommandTemplates {
	t := template.Must(template.New("Templates").Parse(templates))
	return &CommandTemplates{
		templates: t,
	}
}

// LoadTemplates loads the command templates.
func (ct *CommandTemplates) Load() error {
	// TODO: try to load templates from filesystem and update ct.templates
	return nil
}

// Command consists of a command line to be executed and an optional Stdin to
// be passed to the command on execution.
type Command struct {
	Line  string
	Stdin string
}

// executeTemplate executes the template on data and returns the resulting
// output as string.
func (ct *CommandTemplates) executeTemplate(tmpl string, data any) (string, error) {
	t, err := ct.templates.Clone()
	if err != nil {
		return "", err
	}
	t, err = t.Parse(tmpl)
	if err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}
	if err := t.Execute(buf, data); err != nil {
		return "", err
	}

	line := buf.String()
	return line, nil
}

// Run runs the command and returns its output.
func (ct *CommandTemplates) RunCommand(ctx context.Context, cmd *Command, data any) (stdout, stderr []byte, err error) {
	// execute template for command line
	line, err := ct.executeTemplate(cmd.Line, data)
	if err != nil {
		return nil, nil, err
	}

	// execute template for stdin
	stdin, err := ct.executeTemplate(cmd.Stdin, data)
	if err != nil {
		return nil, nil, err
	}

	// extract command from command line
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil, nil, nil
	}
	command := fields[0]

	// extract arguments from command line
	args := []string{}
	if len(fields) > 1 {
		args = fields[1:]
	}

	// run command
	return execs.RunCmd(ctx, command, stdin, args...)
}
