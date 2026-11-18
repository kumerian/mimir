# Local reference profile operations

## Data ownership and layout

The `--state-dir` belongs to the local operator and contains separate logical
authorities:

```text
state-dir/
  metadata/active.jsonl
  metadata/drafts/<revision>.json
  metadata/revisions/<sha256>.json
  metadata/validation-reports/<input-sha256>.json
  metadata/proposals/<proposal-id>.json
  plans/<run-id>.json
  plans/replays/<replay-plan-id>.json
  state/runs.jsonl
  state/checkpoints.jsonl
  state/schedule-requests.jsonl
  state/schedule-request-acks.jsonl
  data/destination.jsonl
  data/quarantine.jsonl
  data/cleaned.jsonl
  audit/events.jsonl
  telemetry/events.jsonl
  LOCK
```

Files are created with owner-only permissions where the host filesystem
supports them. The destination, quarantine, and cleaned data files contain
record values and must be treated according to their classification. Plans
and audit omit secret material. `LOCK` serializes local access. The run-state journal validates attempt
transitions and owner tokens under a journal lock, so a newer attempt fences
stale local writes. Neither mechanism provides distributed worker fencing.
Audit events are required, append-only, and deduplicated; telemetry is a
separate best-effort allowlisted journal and never carries record values.

Quarantine has no automatic expiry or per-record deletion API. The operator
retains it with the local state directory until disposing of that entire
directory under local data-handling rules.

The application refuses state directories or journals with permissions broader
than the local user. Metadata revision identifiers are immutable: activating
different content under a retained revision identifier fails with a conflict.
Complete journal entries with duplicate JSON keys or unknown fields fail
closed.

## Validation and startup

`mimir health` checks the compiled-in source and destination capability
registration and reports the binary and contract identity. It does not claim
that a state directory, source, destination, or production dependency is ready.
Before a run, validate and author/activate the candidate with an explicit local
grant. Run startup rejects changed local source/destination versions or
required guarantees, a missing source directory, and an unavailable or
non-private state directory before reading a source or creating a plan.
Verify source access and state directory permissions separately. Discovery
proposals are inert until explicitly authored and activated. Schedule
evaluation stores requests; it never dispatches runs.
Requests interrupted during schedule evaluation are marked stopped in the
separate acknowledgement journal and audited before cancellation returns.
Operators can explicitly stop all outstanding local requests with
`mimir schedule-stop --state-dir <dir> --reason shutdown --allow-local`.
Acknowledgements are immutable and idempotent; they do not create or alter run
execution state. There is no local dispatcher or resume mechanism for stopped
requests.

The `validate` command uses the embedded, pinned Draft 2020-12 validator and
semantic parser. Use `validate --metadata <file> --state-dir <dir>` to retain
an immutable report under `metadata/validation-reports/`; query it with
`validation-report --state-dir <dir> --input-digest <sha256>`. The report
contains only a digest, safe diagnostics, compatibility/security decisions,
and `runEligible: false`. Reports are immutable and may be repeated only for
identical report content. Invalid authoring persists this report before
returning a failure and does not create a revision. Protect and back up these
reports with the metadata directory; they do not contain the source document.

## Recovery

1. Stop other local processes using the state directory.
2. Preserve a complete copy of the directory before intervention.
3. If an unterminated final journal line exists, the next application open
   removes only that tail; complete malformed entries cause a fail-closed
   error.
4. If destination commit exists but checkpoint or commit audit is absent, rerun
   with the same active revision and unchanged source file. The adapter
   reconciles by stable batch identity before appending.
5. A checkpoint without its matching destination commit is inconsistent state.
   The application refuses to recreate the destination or report success.
   Preserve and restore the complete state directory or escalate for operator
   recovery; do not delete the checkpoint or synthesize destination evidence.
6. If the `LOCK` remains, verify the recorded process is no longer active
   before removing it. Never clear a lock based only on elapsed time.
7. If reconciliation remains ambiguous, preserve the directory and escalate;
   do not retry with edited state or changed source contents.
8. For quality quarantine, verify the destination, quarantine, audit, run, and
   checkpoint evidence together. A partial set is not success; rerun only with
   the unchanged source and active revision so the batch identities can be
   reconciled.
9. Cleaning writes a separate idempotent output and audit event. If output is
   present but the audit event is absent, repeat the same correction request
   to reconcile it; never edit or replace the original record.
10. The local automated fault matrix covers interruptions before and after
    plan audit, attempt start, destination commit, quarantine commit, checkpoint,
    run audit, and terminal state. These tests establish only the documented
    local JSONL behavior; they do not prove distributed leases, source
    acknowledgement, or network-filesystem recovery.
11. Replay fault injection covers before/after immutable replay-plan write,
    before/after replay preparation audit, and before/after reconciliation of
    the original idempotent batch. Restart must retain one destination commit,
    one checkpoint, and one replay-preparation audit identity.

## Backup, restore, rollback, and uninstall

The profile has no automated backup. Stop the process and copy the entire
state directory to a protected local backup. Restore the directory as one
consistent set; never restore only the checkpoint or destination journal.
The local automated offline whole-directory drill is
`go test ./internal/local -run '^TestOfflineWholeDirectoryBackupRestorePreservesReplay$'`.
It verifies restoration and deduplication on the current host only.
Reinstall/rollback the binary only when its metadata, plan, and journal
contracts remain compatible. Unsupported state versions fail closed and need
an explicit forward migration; no destructive migration is implemented.

To uninstall local sample data, stop the process, preserve any required
evidence, and remove only the explicitly configured synthetic state and source
directories. Do not use production or personal data in this profile.

## Troubleshooting

| Symptom | Meaning and action |
| --- | --- |
| `access_denied` | The explicit local-user grant or classification policy failed. Review the redacted denial event; do not broaden classification to bypass it. |
| `unsupported_capability` | The plan lacks a required local source/destination guarantee. No adapter call or publication should follow. |
| `timeout` | A bounded operation exceeded its context deadline. Retry only when the operation's stable identity and commit outcome make reconciliation safe. |
| `ambiguous_commit` | Destination write result is uncertain. Rerun unchanged inputs to reconcile; do not blindly retry edited data. |
| `journal ... malformed` | A complete journal entry is corrupt. Preserve the directory and escalate; the application does not discard complete evidence. |
| `metadata.validation.load` | The report digest is invalid, missing, or its immutable file failed integrity/permission checks. Confirm the exact original input digest and state-directory integrity. |
| `run.config` | The source root is unavailable or the state directory is missing/not private. Correct the local paths/permissions before retrying; preflight does not create run state or read source records. |
| State directory is owned | Another process has `LOCK`, or a prior process exited unexpectedly. Verify process ownership before clearing it. |
| Readiness is `ready` but a run fails | `health` checks built-in capability registration, not filesystem or full run readiness. Validate metadata, source path, and local permissions. |

Production deployment, identity, secret stores, encryption, retention periods,
external telemetry, and automated rollback are not implemented by this local
reference profile.
