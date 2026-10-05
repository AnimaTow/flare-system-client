package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	clientContext "github.com/flare-foundation/flare-system-client/client/context"
	"github.com/flare-foundation/flare-system-client/client/runner"
	"github.com/flare-foundation/flare-system-client/client/shared"
	globalConfig "github.com/flare-foundation/flare-system-client/config"

	"github.com/flare-foundation/go-flare-common/pkg/database"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
)

// errSignal is the cancel cause of a requested shutdown, the only stop that exits 0.
var errSignal = errors.New("shutdown signal")

func main() {
	logger.Info("Starting Flare System client")

	clientCtx, err := clientContext.BuildContext()
	if err != nil {
		logger.Fatalf("building context: %v", err)
	}

	logger.Set(clientCtx.Config().Logger)
	// after Set: Logger() is a snapshot, and DB retry errors are dropped until a logger is set
	database.SetErrorLogger(logger.Logger())

	for _, w := range clientCtx.Config().SubmitterWarnings() {
		logger.Warnf("submitter config: %s", w)
	}

	for _, w := range clientCtx.Config().GasOverrideWarnings() {
		logger.Warnf("gas config: %s", w)
	}

	// txs are signed with the configured chain_id — a mismatch fails every send
	verifyCtx, cancelVerify := context.WithTimeout(context.Background(), 10*time.Second)
	err = clientCtx.Config().Chain.VerifyChainID(verifyCtx)
	cancelVerify()
	if errors.Is(err, globalConfig.ErrChainIDMismatch) {
		logger.Fatalf("chain config: %v", err)
	} else if err != nil {
		logger.Warnf("chain config: could not verify chain_id against the node: %v", err)
	}

	// Prometheus metrics
	shared.InitMetricsServer(&clientCtx.Config().Metrics)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancelCause(context.Background())
	go func() {
		sig := <-signalChan
		logger.Infof("Received %v signal, attempting graceful shutdown", sig)
		cancel(errSignal)
	}()

	wg := runner.Start(ctx, cancel, clientCtx)
	wg.Wait()
	// the first cause wins, so errors returned while draining after a signal keep exit 0
	if cause := context.Cause(ctx); cause != nil && !errors.Is(cause, errSignal) {
		logger.Fatalf("Stopped Flare System client: %v", cause)
	}
	logger.Info("Stopped Flare System client")
}
