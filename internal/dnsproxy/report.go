package dnsproxy

import (
	"fmt"
	"net"
)

// Report is a report for a watched domain.
type Report struct {
	Name string
	IP   net.IP
	TTL  uint32

	// done is used to signal that the report has been handled by
	// its consumer
	done chan struct{}
}

// String returns the report as string.
func (r *Report) String() string {
	return fmt.Sprintf("%s -> %s (ttl: %d)", r.Name, r.IP, r.TTL)
}

// Close signals that the report has been handled by its consumer.
func (r *Report) Close() {
	close(r.done)
}

// Done returns a channel that is closed when the report was handled by its consumer.
func (r *Report) Done() <-chan struct{} {
	return r.done
}

// NewReport returns a new report with domain name, IP and TTL.
func NewReport(name string, ip net.IP, ttl uint32) *Report {
	return &Report{
		Name: name,
		IP:   ip,
		TTL:  ttl,

		done: make(chan struct{}),
	}
}
