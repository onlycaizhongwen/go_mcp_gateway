package registry

import "context"

type Discovery interface {
	ListHealthy(ctx context.Context, ref ServiceRef) ([]Instance, error)
}

type Registrar interface {
	Register(ctx context.Context, instance Instance) error
	Deregister(ctx context.Context, instance Instance) error
}

type Client interface {
	Discovery
	Registrar
}
