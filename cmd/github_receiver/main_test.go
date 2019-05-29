package main

import (
	"flag"
	"io/ioutil"
	"sync"
	"testing"

	"github.com/m-lab/go/prometheusx"
	"github.com/m-lab/go/prometheusx/promtest"
)

func TestMetrics(t *testing.T) {
	receiverDuration.WithLabelValues("x")
	promtest.LintMetrics(t)
}

func Test_main(t *testing.T) {
	tests := []struct {
		name      string
		authfile  string
		authtoken string
		repo      string
		inmemory  bool
	}{
		{
			name:      "okay-default",
			repo:      "fake-repo",
			authtoken: "token",
			inmemory:  false,
		},
		{
			name:     "okay-inmemory",
			authfile: "fake-token",
			repo:     "fake-repo",
			inmemory: true,
		},
		{
			name: "missing-flags-usage",
		},
	}
	flag.CommandLine.SetOutput(ioutil.Discard)
	osExit = func(int) {}
	for _, tt := range tests {
		*authtoken = tt.authtoken
		authtokenFile = []byte(tt.authfile)
		*githubOrg = "fake-org"
		*githubRepo = tt.repo
		*enableInMemory = tt.inmemory
		// Guarantee no port conflicts between tests of main.
		*prometheusx.ListenAddress = ":0"
		*receiverAddr = ":0"
		t.Run(tt.name, func(t *testing.T) {
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				main()
				wg.Done()
			}()
			cancelCtx()
			wg.Wait()
		})
	}
}