package main

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
)

type staticProvider struct {
	env []string
}

func (provider staticProvider) ResolveSecrets(context.Context, []string) ([]string, error) {
	return provider.env, nil
}

func TestRunDoesNotLogArgumentsOrResolvedEnvironment(t *testing.T) {
	const secret = "do-not-log-this-secret"

	logger := log.StandardLogger()
	originalOutput := logger.Out
	originalLevel := logger.Level
	originalFormatter := logger.Formatter
	t.Cleanup(func() {
		log.SetOutput(originalOutput)
		log.SetLevel(originalLevel)
		log.SetFormatter(originalFormatter)
	})

	var output bytes.Buffer
	log.SetOutput(&output)
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.JSONFormatter{})

	pid, err := run(
		context.Background(),
		staticProvider{env: []string{"API_TOKEN=" + secret}},
		true,
		false,
		[]string{"true", "--token=" + secret},
	)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		t.Fatalf("os.FindProcess() error = %v", err)
	}
	if _, err = process.Wait(); err != nil {
		t.Fatalf("process.Wait() error = %v", err)
	}

	logs := output.String()
	if strings.Contains(logs, secret) {
		t.Fatalf("run() logged a secret: %s", logs)
	}
	if strings.Contains(logs, `"env"`) || strings.Contains(logs, `"args"`) {
		t.Fatalf("run() logged sensitive fields: %s", logs)
	}
	if !strings.Contains(logs, `"command":"true"`) {
		t.Fatalf("run() did not log the safe command field: %s", logs)
	}
}
