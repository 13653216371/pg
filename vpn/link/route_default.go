//go:build !linux && !windows && !darwin

package link

import (
	"context"
	"errors"
)

func RouteSubscribe(ctx context.Context, ch chan<- RouteUpdate) error {
	return errors.ErrUnsupported
}