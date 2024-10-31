package splitrt

import (
	"context"
	"net/netip"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/cmdtmpl"
)

// addExclude adds exclude address to netfilter.
func addExclude(ctx context.Context, address netip.Prefix) {
	log.WithField("address", address).Debug("SplitRouting adding exclude to netfilter")

	ct := cmdtmpl.NewCommandTemplates(DefaultTemplates)
	data := address
	commands := []*cmdtmpl.Command{
		{Line: "nft -f -",
			Stdin: `
				{{- if .Addr.Is6 -}}
				add element inet oc-daemon-routing excludes6 { {{.}} }
				{{- else -}}
				add element inet oc-daemon-routing excludes4 { {{.}} }
				{{- end}}`},
	}
	for _, c := range commands {
		// TODO: get final command and stdin
		stdout, stderr, err := ct.RunCommand(ctx, c, data)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"address": address,
				"command": c.Line,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
			}).Error("Error adding exclude")
		}
	}
}

// setExcludes resets the excludes to addresses in netfilter.
func setExcludes(ctx context.Context, addresses []netip.Prefix) {

	ct := cmdtmpl.NewCommandTemplates(DefaultTemplates)
	data := addresses
	commands := []*cmdtmpl.Command{
		// flush existing entries
		// add entries
		{Line: "nft -f -",
			Stdin: `flush set inet oc-daemon-routing excludes4
flush set inet oc-daemon-routing excludes6
{{range . -}}
{{if .Addr.Is6 -}}
add element inet oc-daemon-routing excludes6 { {{.}} }
{{else -}}
add element inet oc-daemon-routing excludes4 { {{.}} }
{{end -}}
{{end}}`},
	}
	for _, c := range commands {
		// TODO: get final command and stdin
		stdout, stderr, err := ct.RunCommand(ctx, c, data)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"addresses": addresses,
				"command":   c.Line,
				"stdin":     c.Stdin,
				"stdout":    string(stdout),
				"stderr":    string(stderr),
			}).Error("Error setting excludes")
		}
	}
}
