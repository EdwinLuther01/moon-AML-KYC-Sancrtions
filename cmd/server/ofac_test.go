// Copyright 2019 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/moov-io/base"
	"github.com/moov-io/base/docker"
	ofac "github.com/moov-io/ofac/client"

	"github.com/go-kit/kit/log"
	"github.com/ory/dockertest"
)

type testOFACClient struct {
	sdn *ofac.Sdn

	// error to be returned instead of field from above
	err error
}

func (c *testOFACClient) Ping() error {
	return c.err
}

func (c *testOFACClient) Search(_ context.Context, name string, _ string) (*ofac.Sdn, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.sdn, nil
}

type ofacDeployment struct {
	res    *dockertest.Resource
	client OFACClient
}

func (d *ofacDeployment) close(t *testing.T) {
	if err := d.res.Close(); err != nil {
		t.Error(err)
	}
}

func spawnOFAC(t *testing.T) *ofacDeployment {
	// no t.Helper() call so we know where it failed

	if testing.Short() {
		t.Skip("-short flag enabled")
	}
	if !docker.Enabled() {
		t.Skip("Docker not enabled")
	}

	// Spawn OFAC docker image
	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatal(err)
	}
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "moov/ofac",
		Tag:        "v0.7.0",
		Cmd:        []string{"-http.addr=:8080"},
	})
	if err != nil {
		t.Fatal(err)
	}

	client := newOFACClient(log.NewNopLogger(), fmt.Sprintf("http://localhost:%s", resource.GetPort("8080/tcp")))
	err = pool.Retry(func() error {
		return client.Ping()
	})
	if err != nil {
		t.Fatal(err)
	}
	return &ofacDeployment{resource, client}
}

func TestOFAC__client(t *testing.T) {
	endpoint := ""
	if client := newOFACClient(log.NewNopLogger(), endpoint); client == nil {
		t.Fatal("expected non-nil client")
	}

	// Spawn an OFAC Docker image and ping against it
	deployment := spawnOFAC(t)
	if err := deployment.client.Ping(); err != nil {
		t.Fatal(err)
	}
	deployment.close(t) // close only if successful
}

func TestOFAC__search(t *testing.T) {
	ctx := context.TODO()

	deployment := spawnOFAC(t)

	// Search query that matches an SDN higher than an AltName
	sdn, err := deployment.client.Search(ctx, "Nicolas Maduro", base.ID())
	if err != nil || sdn == nil {
		t.Fatalf("sdn=%v err=%v", sdn, err)
	}
	if sdn.EntityID != "22790" {
		t.Errorf("SDN=%s %#v", sdn.EntityID, sdn)
	}

	// Search query that matches an AltName higher than SDN
	sdn, err = deployment.client.Search(ctx, "Osama bin Muhammad bin Awad BIN LADIN", base.ID())
	if err != nil || sdn == nil {
		t.Fatalf("sdn=%v err=%v", sdn, err)
	}
	if sdn.EntityID != "6365" {
		t.Errorf("SDN=%s %#v", sdn.EntityID, sdn)
	}

	deployment.close(t) // close only if successful
}

func TestOFAC_ping(t *testing.T) {
	client := &testOFACClient{}

	// Ping tests
	if err := client.Ping(); err != nil {
		t.Error("expected no error")
	}

	// set error and verify we get it
	client.err = errors.New("ping error")
	if err := client.Ping(); err == nil {
		t.Error("expected error")
	} else {
		if !strings.Contains(err.Error(), "ping error") {
			t.Errorf("unknown error: %v", err)
		}
	}
}
