package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	boshapp "github.com/cloudfoundry/bosh-agent/app"
	"github.com/cloudfoundry/bosh-agent/infrastructure/agentlogger"
	"github.com/cloudfoundry/bosh-utils/logger"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

const mainLogTag = "main"

func runAgent(logger logger.Logger) chan error {
	errCh := make(chan error, 1)
	go func() {
		defer logger.HandlePanic("Main")

		logger.Debug(mainLogTag, "Starting agent")

		fs := boshsys.NewOsFileSystem(logger)
		app := boshapp.New(logger, fs)

		err := app.Setup(os.Args)
		if err != nil {
			logger.Error(mainLogTag, "App setup %s", err.Error())
			errCh <- err
			return
		}

		err = app.Run()
		if err != nil {
			logger.Error(mainLogTag, "App run %s", err.Error())
			errCh <- err
			return
		}
	}()
	return errCh
}

func startAgent(logger logger.Logger) error {
	sigCh := make(chan os.Signal, 8)
	signal.Notify(sigCh, os.Interrupt, os.Kill)
	errCh := runAgent(logger)
	for {
		select {
		case sig := <-sigCh:
			return fmt.Errorf("received signal (%s): stopping now", sig)
		case err := <-errCh:
			return err
		}
	}
}

func main() {
	asyncLog := boshlog.NewAsyncWriterLogger(boshlog.LevelDebug, os.Stdout, os.Stderr)
	logger := newSignalableLogger(asyncLog)

	exitCode := 0
	if err := startAgent(logger); err != nil {
		logger.Error(mainLogTag, "Agent exited with error: %s", err)
		exitCode = 1
	}
	logger.FlushTimeout(time.Minute)
	os.Exit(exitCode)
}

func newSignalableLogger(logger logger.Logger) logger.Logger {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGSEGV)
	signalableLogger, _ := agentlogger.NewSignalableLogger(logger, c)
	return signalableLogger
}
