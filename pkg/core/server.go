package core

type Address struct {
	Port       int
	IP         string
	Password   string
	Name       string
	Key        string
	ServerName string
	Pem        string
}

type Server struct {
	Address Address
}

func (s *Server) Init() (*Cli, error) {
	c, err := New(s.Address.IP, s.Address.Name, s.Address.Password, s.Address.Pem, s.Address.Port)
	return c, err
}
