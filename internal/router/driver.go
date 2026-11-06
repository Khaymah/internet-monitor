package router

import (
	"context"
	"errors"
)

var ErrPasswordRequired = errors.New("the router needs a password before it can be read")

type Creds struct {
	User     string
	Password string
}

type Driver interface {
	Name() string
	Detect(ctx context.Context, p *Probe) (bool, error)
	Read(ctx context.Context, p *Probe, c Creds) (WebStats, error)
}

type AnonymousReader interface {
	Anonymous() bool
}

type ExperimentalDriver interface {
	Experimental() bool
}

func IsExperimental(d Driver) bool {
	e, ok := d.(ExperimentalDriver)
	return ok && e.Experimental()
}

func canReadAnonymously(d Driver) bool {
	a, ok := d.(AnonymousReader)
	return ok && a.Anonymous()
}

func Drivers() []Driver {
	return []Driver{huaweiHG{}, zteNew{}, tr064{}}
}

func Detect(ctx context.Context, p *Probe, drivers []Driver, anonymousOnly bool) (Driver, error) {
	var errs []error
	for _, d := range drivers {
		if anonymousOnly && !canReadAnonymously(d) {
			continue
		}
		ok, err := d.Detect(ctx, p)
		if ok {
			return d, nil
		}
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 && p.Unreachable() {
		return nil, errors.Join(errs...)
	}
	return nil, ErrWebUnknown
}
