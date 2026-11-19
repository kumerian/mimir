package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"syscall"
	"time"

	"github.com/kumerian/mimir/internal/adapters/mariadb"
	"github.com/kumerian/mimir/internal/httpapi"
	"github.com/kumerian/mimir/internal/local"
)

func serveHTTP(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(stderr)
	listenAddress := flags.String("listen", "127.0.0.1:8787", "loopback IP and TCP port")
	stateDirectory := flags.String("state-dir", "", "fixed private local state directory")
	allowLocal := flags.Bool("allow-local", false, "authorize the explicit local-user reference profile")
	containerNetwork := flags.Bool("container-network", false, "allow the explicit private-container 0.0.0.0 listener and MariaDB metadata profile")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return 2
	}
	if !*allowLocal {
		fmt.Fprintln(stderr, "serve requires --allow-local")
		return 2
	}
	if !filepath.IsAbs(*stateDirectory) {
		fmt.Fprintln(stderr, "serve requires an absolute --state-dir")
		return 2
	}
	if err := httpapi.ValidateListenAddressMode(*listenAddress, *containerNetwork); err != nil {
		fmt.Fprintln(stderr, "serve listen address is not allowed for the selected mode")
		return 2
	}
	if err := local.CheckProfile(); err != nil {
		fmt.Fprintln(stderr, "local reference profile is unavailable")
		return 1
	}
	contracts := append(contractVersions(), httpapi.ContractVersion)
	sort.Strings(contracts)
	profile := httpapi.ProfileHealth{
		Status: "ready", Version: buildVersion, Commit: buildCommit,
		Contracts: contracts, Profile: "local-reference-v1",
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var (
		operations *httpapi.LocalOperations
		readiness  func(context.Context) error
		store      *mariadb.Store
		err        error
	)
	if *containerNetwork {
		startupCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
		store, err = mariadb.Open(startupCtx, mariadb.Config{
			Address:           os.Getenv("MIMIR_DB_ADDRESS"),
			RuntimeUser:       os.Getenv("MIMIR_DB_RUNTIME_USER"),
			RuntimePassword:   os.Getenv("MIMIR_DB_RUNTIME_PASSWORD"),
			MigrationUser:     os.Getenv("MIMIR_DB_MIGRATION_USER"),
			MigrationPassword: os.Getenv("MIMIR_DB_MIGRATION_PASSWORD"),
		})
		cancel()
		if err != nil {
			fmt.Fprintln(stderr, "metadata database is not ready or its configuration is invalid")
			return 1
		}
		defer store.Close()
		repository, reports, proposals := store.MetadataStores()
		operations, err = httpapi.NewContainerLocalOperationsWithStores(
			*stateDirectory, *allowLocal, profile, repository, reports, proposals, "/data/sources",
		)
		readiness = store.Ping
	} else {
		operations, err = httpapi.NewLocalOperations(*stateDirectory, *allowLocal, profile)
	}
	if err != nil {
		fmt.Fprintln(stderr, "local API configuration is invalid")
		return 2
	}
	listener, err := net.Listen("tcp", *listenAddress)
	if err != nil {
		fmt.Fprintln(stderr, "local API listener could not be opened")
		return 1
	}
	server := &http.Server{
		Handler:           httpapi.NewHandlerWithReadiness(operations, readiness),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	fmt.Fprintf(stdout, "serving local API on %s\n", *listenAddress)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()
	select {
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintln(stderr, "local API server stopped unexpectedly")
			return 1
		}
		return 0
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			fmt.Fprintln(stderr, "local API shutdown exceeded its grace period")
			return 1
		}
		if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintln(stderr, "local API server stopped unexpectedly")
			return 1
		}
		return 0
	}
}
