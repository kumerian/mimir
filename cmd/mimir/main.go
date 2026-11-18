package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kumerian/mimir/internal/cleaning"
	"github.com/kumerian/mimir/internal/local"
	"github.com/kumerian/mimir/internal/metadata"
	"github.com/kumerian/mimir/internal/scheduler"
)

var (
	buildVersion = "dev"
	buildCommit  = "unknown"
)

type health struct {
	Status    string   `json:"status"`
	Version   string   `json:"version"`
	Commit    string   `json:"commit"`
	Contracts []string `json:"contracts"`
	Profile   string   `json:"profile"`
}

func main() {
	os.Exit(execute(os.Args[1:], os.Stdout, os.Stderr))
}

func execute(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: mimir <health|validate|validation-report|author|list-metadata|activate|discover|schedule|schedule-stop|run|clean|replay> [options]")
		return 2
	}
	switch args[0] {
	case "health":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "health does not accept configuration arguments")
			return 2
		}
		if err := local.CheckProfile(); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return writeJSON(stdout, health{
			Status: "ready", Version: buildVersion, Commit: buildCommit,
			Contracts: []string{
				metadata.ContractVersion, metadata.ValidationReportVersion,
				scheduler.ContractVersion, scheduler.AcknowledgementContractVersion,
				"mimir.failure/v1",
			},
			Profile: "local-reference-v1",
		})
	case "validate":
		flags := flag.NewFlagSet("validate", flag.ContinueOnError)
		flags.SetOutput(stderr)
		path := flags.String("metadata", "", "path to a metadata JSON document")
		stateDir := flags.String("state-dir", "", "optional private directory for immutable report persistence")
		if err := flags.Parse(args[1:]); err != nil || *path == "" {
			return 2
		}
		file, err := os.Open(*path)
		if err != nil {
			fmt.Fprintln(stderr, "metadata file is unavailable")
			return 1
		}
		raw, readErr := io.ReadAll(io.LimitReader(file, (1<<20)+1))
		closeErr := file.Close()
		if readErr != nil || closeErr != nil {
			fmt.Fprintln(stderr, "metadata file could not be read")
			return 1
		}
		report := metadata.Validate(raw)
		if *stateDir != "" {
			report, err = local.ValidateMetadata(context.Background(), raw, *stateDir)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
		}
		if code := writeJSON(stdout, report); code != 0 {
			return code
		}
		if !report.Valid {
			return 1
		}
		return 0
	case "validation-report":
		flags := flag.NewFlagSet("validation-report", flag.ContinueOnError)
		flags.SetOutput(stderr)
		stateDir := flags.String("state-dir", "", "private metadata state directory")
		inputDigest := flags.String("input-digest", "", "SHA-256 identity of the exact validated input")
		if err := flags.Parse(args[1:]); err != nil || *stateDir == "" || *inputDigest == "" {
			return 2
		}
		report, err := local.LoadMetadataValidationReport(context.Background(), *stateDir, *inputDigest)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return writeJSON(stdout, report)
	case "author":
		flags := flag.NewFlagSet("author", flag.ContinueOnError)
		flags.SetOutput(stderr)
		candidate := flags.String("metadata", "", "path to a draft metadata JSON document")
		stateDir := flags.String("state-dir", "", "local metadata, state, data, and audit directory")
		allowLocal := flags.Bool("allow-local", false, "authorize the explicit local-user reference profile")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		result, err := local.Author(context.Background(), *candidate, *stateDir, "local-user", *allowLocal)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return writeJSON(stdout, result)
	case "list-metadata":
		flags := flag.NewFlagSet("list-metadata", flag.ContinueOnError)
		flags.SetOutput(stderr)
		stateDir := flags.String("state-dir", "", "local metadata state directory")
		allowLocal := flags.Bool("allow-local", false, "authorize the explicit local-user reference profile")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		result, err := local.ListMetadata(*stateDir, "local-user", *allowLocal)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return writeJSON(stdout, result)
	case "activate":
		flags := flag.NewFlagSet("activate", flag.ContinueOnError)
		flags.SetOutput(stderr)
		candidate := flags.String("metadata", "", "path to a draft metadata JSON document")
		revision := flags.String("revision", "", "revision ID of a previously authored draft")
		stateDir := flags.String("state-dir", "", "local metadata, state, data, and audit directory")
		expectedActive := flags.String("expected-active", "", "expected current active revision (required when superseding)")
		allowLocal := flags.Bool("allow-local", false, "authorize the explicit local-user reference profile")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		var result local.ActivationResult
		var err error
		if *revision != "" {
			result, err = local.ActivateDraft(context.Background(), *revision, *stateDir, "local-user", *allowLocal, *expectedActive)
		} else {
			result, err = local.Activate(context.Background(), *candidate, *stateDir, "local-user", *allowLocal, *expectedActive)
		}
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return writeJSON(stdout, result)
	case "discover":
		flags := flag.NewFlagSet("discover", flag.ContinueOnError)
		flags.SetOutput(stderr)
		sourceRoot := flags.String("source-root", "", "directory containing the bounded source object")
		stateDir := flags.String("state-dir", "", "local metadata, state, data, and audit directory")
		disposition := flags.String("disposition", "hold", "explicit schema-drift disposition")
		allowLocal := flags.Bool("allow-local", false, "authorize the explicit local-user reference profile")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		result, err := local.Discover(ctx, local.DiscoveryOptions{
			SourceRoot: *sourceRoot, StateDir: *stateDir, Actor: "local-user",
			AllowLocal: *allowLocal, Disposition: *disposition,
		})
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return writeJSON(stdout, result)
	case "schedule":
		flags := flag.NewFlagSet("schedule", flag.ContinueOnError)
		flags.SetOutput(stderr)
		stateDir := flags.String("state-dir", "", "local metadata, run, state, data, and audit directory")
		at := flags.String("at", "", "fixed RFC3339 evaluation time (defaults to current UTC time)")
		allowLocal := flags.Bool("allow-local", false, "authorize the explicit local-user reference profile")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		var clock time.Time
		if *at != "" {
			var err error
			clock, err = time.Parse(time.RFC3339, *at)
			if err != nil {
				fmt.Fprintln(stderr, "--at must be an RFC3339 timestamp")
				return 2
			}
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		result, err := local.EvaluateSchedules(ctx, local.ScheduleOptions{
			StateDir: *stateDir, Actor: "local-user", AllowLocal: *allowLocal, At: clock,
		})
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return writeJSON(stdout, result)
	case "schedule-stop":
		flags := flag.NewFlagSet("schedule-stop", flag.ContinueOnError)
		flags.SetOutput(stderr)
		stateDir := flags.String("state-dir", "", "local metadata, run, state, data, and audit directory")
		reason := flags.String("reason", "shutdown", "cancelled or shutdown")
		allowLocal := flags.Bool("allow-local", false, "authorize the explicit local-user reference profile")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		if err := local.StopPendingScheduleRequests(context.Background(), *stateDir, "local-user", *allowLocal, *reason); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return writeJSON(stdout, struct {
			Status string `json:"status"`
		}{Status: "pending_requests_stopped"})
	case "run":
		flags := flag.NewFlagSet("run", flag.ContinueOnError)
		flags.SetOutput(stderr)
		sourceRoot := flags.String("source-root", "", "directory containing the bounded source object")
		stateDir := flags.String("state-dir", "", "local state, data, and audit directory")
		allowLocal := flags.Bool("allow-local", false, "authorize the explicit local-user reference profile")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		result, err := local.Run(ctx, local.Options{
			SourceRoot: *sourceRoot,
			StateDir:   *stateDir,
			Actor:      "local-user",
			AllowLocal: *allowLocal,
		})
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return writeJSON(stdout, result)
	case "clean":
		flags := flag.NewFlagSet("clean", flag.ContinueOnError)
		flags.SetOutput(stderr)
		recordPath := flags.String("record", "", "path to one JSON-object record")
		stateDir := flags.String("state-dir", "", "local state, cleaned-data, and audit directory")
		classification := flags.String("classification", "", "explicit public or internal classification")
		correctionID := flags.String("correction-id", "", "versioned correction identifier")
		correctionVersion := flags.String("correction-version", "", "correction version")
		operation := flags.String("operation", "", "trim, lowercase, uppercase, or replace_null")
		field := flags.String("field", "", "record field to correct")
		value := flags.String("value-json", "", "JSON replacement value for replace_null")
		preview := flags.Bool("preview", false, "show the deterministic output without persisting it")
		allowLocal := flags.Bool("allow-local", false, "authorize the explicit local-user reference profile")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		var replacement any
		if *value != "" {
			decoder := json.NewDecoder(strings.NewReader(*value))
			decoder.UseNumber()
			if err := decoder.Decode(&replacement); err != nil {
				fmt.Fprintln(stderr, "replacement value must be valid JSON")
				return 2
			}
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		result, err := local.CleanRecord(ctx, local.CleaningOptions{
			StateDir: *stateDir, RecordPath: *recordPath, Actor: "local-user",
			Classification: *classification, AllowLocal: *allowLocal, Preview: *preview,
			Correction: cleaning.Correction{
				ID: *correctionID, Version: *correctionVersion, Operation: *operation, Field: *field, Value: replacement,
			},
		})
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return writeJSON(stdout, result)
	case "replay":
		flags := flag.NewFlagSet("replay", flag.ContinueOnError)
		flags.SetOutput(stderr)
		sourceRoot := flags.String("source-root", "", "directory containing the bounded source object")
		stateDir := flags.String("state-dir", "", "local metadata, run, data, and audit directory")
		runID := flags.String("run-id", "", "original immutable run identity")
		reason := flags.String("reason", "", "operator_request, recovery, or incident")
		allowLocal := flags.Bool("allow-local", false, "authorize the explicit local-user reference profile")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		result, err := local.Replay(ctx, local.Options{
			SourceRoot: *sourceRoot, StateDir: *stateDir, Actor: "local-user", AllowLocal: *allowLocal,
		}, *runID, *reason)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return writeJSON(stdout, result)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
}

func writeJSON(writer io.Writer, value any) int {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(true)
	if err := encoder.Encode(value); err != nil {
		return 1
	}
	return 0
}
