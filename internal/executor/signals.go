package executor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
)

type signalCancellation struct {
	signal os.Signal
}

func (cancellation signalCancellation) Error() string {
	return fmt.Sprintf("received signal %s", cancellation.signal)
}

func NotifyContext(
	parent context.Context,
	signals ...os.Signal,
) (context.Context, context.CancelFunc) {
	ctx, cancelCause := context.WithCancelCause(parent)
	received := make(chan os.Signal, 1)
	signal.Notify(received, signals...)
	stopped := make(chan struct{})
	var stopOnce sync.Once

	go func() {
		select {
		case receivedSignal := <-received:
			cancelCause(signalCancellation{signal: receivedSignal})
		case <-stopped:
		case <-ctx.Done():
		}
	}()

	return ctx, func() {
		stopOnce.Do(func() {
			signal.Stop(received)
			close(stopped)
			cancelCause(context.Canceled)
		})
	}
}

func ContextSignal(ctx context.Context) (os.Signal, bool) {
	var cancellation signalCancellation
	if !errors.As(context.Cause(ctx), &cancellation) {
		return nil, false
	}

	return cancellation.signal, true
}
