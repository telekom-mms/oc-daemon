package vpncscript

import (
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/api"
	"github.com/telekom-mms/oc-daemon/internal/daemon"
)

// runClient interacts with the daemon over the api
func runClient(socketFile string, configUpdate *daemon.VPNConfigUpdate) error {
	// connect to daemon
	conn, err := net.Dial("unix", socketFile)
	if err != nil {
		return fmt.Errorf("VPNCScript could not connect to Daemon: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	// send message to daemon
	b, err := configUpdate.JSON()
	if err != nil {
		return fmt.Errorf("VPNCScript could not convert config update to JSON: %w", err)
	}
	msg := api.NewMessage(api.TypeVPNConfigUpdate, b)
	err = api.WriteMessage(conn, msg)
	if err != nil {
		return fmt.Errorf("VPNCScript could not send message to Daemon: %w", err)
	}

	// receive reply
	reply, err := api.ReadMessage(conn)
	if err != nil {
		return fmt.Errorf("VPNCScript could not receive reply from Daemon: %w", err)
	}
	switch reply.Type {
	case api.TypeOK:
		log.WithField("reply", reply.Value).
			Debug("VPNCScript received OK reply from Daemon")
	case api.TypeError:
		log.WithField("error", string(reply.Value)).
			Error("VPNCScript received error reply from Daemon")
	}
	return nil
}
