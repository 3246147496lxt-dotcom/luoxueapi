package lifecycle

import "context"

// Component is a long-lived process component managed by Supervisor.
// Implementations must make Stop safe to call more than once.
type Component interface {
	Name() string
	Start(context.Context) error
	Stop(context.Context) error
}

// ComponentFuncs adapts lifecycle functions without making service packages
// depend on the process supervisor.
type ComponentFuncs struct {
	ComponentName string
	StartFunc     func(context.Context) error
	StopFunc      func(context.Context) error
}

func (c ComponentFuncs) Name() string { return c.ComponentName }

func (c ComponentFuncs) Start(ctx context.Context) error {
	if c.StartFunc == nil {
		return nil
	}
	return c.StartFunc(ctx)
}

func (c ComponentFuncs) Stop(ctx context.Context) error {
	if c.StopFunc == nil {
		return nil
	}
	return c.StopFunc(ctx)
}
