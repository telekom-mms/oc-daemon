package vpncscript

import (
	"net"

	"github.com/T-Systems-MMS/oc-daemon/internal/api"
	"github.com/T-Systems-MMS/oc-daemon/internal/vpnconfig"
	log "github.com/sirupsen/logrus"
)

const (
	runDir   = "/run/oc-daemon"
	sockFile = runDir + "/daemon.sock"
)

// runClient interacts with the daemon over the api
func runClient(configUpdate *vpnconfig.ConfigUpdate) {
	// connect to daemon
	conn, err := net.Dial("unix", sockFile)
	if err != nil {
		log.WithError(err).Fatal("VPNCScript could not connect to Daemon")
	}
	defer func() {
		_ = conn.Close()
	}()

	// send message to daemon
	b, err := configUpdate.JSON()
	if err != nil {
		log.WithError(err).Fatal("VPNCScript could not convert config update to JSON")
	}
	msg := api.NewMessage(api.TypeVPNConfigUpdate, b)
	err = api.WriteMessage(conn, msg)
	if err != nil {
		log.WithError(err).Fatal("VPNCScript could not send message to Daemon")
	}

	// receive reply
	reply, err := api.ReadMessage(conn)
	if err != nil {
		log.WithError(err).Fatal("VPNCScript could not receive reply from Daemon")
	}
	switch reply.Type {
	case api.TypeOK:
		log.WithField("reply", reply.Value).
			Debug("VPNCScript received OK reply from Daemon")
	case api.TypeError:
		log.WithField("error", string(reply.Value)).
			Error("VPNCScript received error reply from Daemon")
	}
}
