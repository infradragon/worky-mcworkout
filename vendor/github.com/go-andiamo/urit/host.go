package urit

type HostOption interface {
	GetAddress() string
}

type Host interface {
	HostOption
}

func NewHost(address string) Host {
	return &host{
		address: address,
	}
}

type host struct {
	address string
}

func (h *host) GetAddress() string {
	return h.address
}
