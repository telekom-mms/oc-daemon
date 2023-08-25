package dnsproxy

var (
	// Address is the default listen address of the DNS proxy
	Address = "127.0.0.1:4253"

	// ListenUDP specifies whether the DNS proxy listens on UDP
	ListenUDP = true

	// ListenTCP specifies whether the DNS proxy listens on TCP
	ListenTCP = true
)

// Config is a DNS proxy configuration
type Config struct {
	Address   string
	ListenUDP bool
	ListenTCP bool
}

// Valid returns whether the DNS proxy configuration is valid
func (c *Config) Valid() bool {
	if c == nil ||
		c.Address == "" ||
		(!c.ListenUDP && !c.ListenTCP) {

		return false
	}
	return true
}

// NewConfig returns a new DNS proxy configuration
func NewConfig() *Config {
	return &Config{
		Address:   Address,
		ListenUDP: ListenUDP,
		ListenTCP: ListenTCP,
	}
}
