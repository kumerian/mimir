import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnInit, inject, signal } from '@angular/core';
import {
  ApiService,
  ActivationResult,
  CleaningRequest,
  CleaningResult,
  DiscoveryResult,
  ProfileHealth,
  ReplayResult,
  RevisionSummary,
  RunResult,
  ScheduleResult,
  StopResult,
  ValidationReport,
} from './api.service';

type View = 'metadata' | 'runs' | 'cleaning';

interface FailureView {
  category: string;
  diagnostic: string;
  operation?: string;
  stage?: string;
  retryable?: boolean;
  commitCertainty?: string;
  reconciliation?: string;
  httpStatus?: number;
}

const INITIAL_METADATA = `{
  "schemaVersion": "mimir.metadata/v1",
  "revision": "synthetic-orders-v1",
  "status": "draft",
  "sources": [
    {
      "id": "synthetic-orders",
      "classification": "internal",
      "objects": [
        {
          "id": "orders",
          "path": "orders.jsonl",
          "columns": [
            { "id": "order_id", "type": "integer" },
            { "id": "customer_name", "type": "string" }
          ]
        }
      ]
    }
  ]
}`;

@Component({
  selector: 'mimir-root',
  standalone: true,
  template: `
    <main class="shell">
      <header class="topbar">
        <a class="brand" href="/" aria-label="Mimir home">
          <span class="brand-mark" aria-hidden="true">M</span>
          <span>Mimir</span>
        </a>
        <span class="local-badge"><span aria-hidden="true"></span> Local workspace</span>
      </header>

      <section class="intro">
        <p class="eyebrow">LOCAL REFERENCE PROFILE</p>
        <h1>Operator workspace</h1>
        <p class="lede">Manage metadata and run only the operations available in this local profile.</p>
      </section>

      <section class="status-card" aria-labelledby="connection-title">
        <div class="status-icon" [class.online]="health()?.status === 'ready'" aria-hidden="true">
          {{ health()?.status === 'ready' ? '✓' : '!' }}
        </div>
        <div class="status-copy">
          <div class="status-heading">
            <div>
              <p class="eyebrow">BACKEND IDENTITY</p>
              <h2 id="connection-title">Local API connection</h2>
            </div>
            <button type="button" class="secondary-button" [disabled]="healthBusy()" (click)="loadHealth()">Refresh</button>
          </div>
          @if (health(); as profile) {
            <p class="connection-state"><span class="dot"></span> Connected</p>
            <dl class="identity-grid">
              <div><dt>Profile</dt><dd>{{ profile.profile }}</dd></div>
              <div><dt>Version</dt><dd>{{ profile.version }}</dd></div>
              <div><dt>Commit</dt><dd>{{ profile.commit }}</dd></div>
              <div><dt>API contract</dt><dd>{{ contractVersion(profile.contracts) }}</dd></div>
            </dl>
            <p class="health-note">Identity only—not a check of filesystem, source, destination, database, or environment readiness.</p>
          } @else if (healthError(); as failure) {
            <div class="inline-error" role="alert">
              <strong>{{ failure.category }}</strong>
              <span>{{ failure.diagnostic }}</span>
              @if (failure.httpStatus) { <span>HTTP {{ failure.httpStatus }}</span> }
            </div>
          } @else {
            <p class="muted">{{ healthBusy() ? 'Checking the Go backend…' : 'Backend identity has not been loaded.' }}</p>
          }
        </div>
      </section>

      <nav class="workspace-nav" aria-label="Workspace sections">
        <button type="button" [class.active]="view() === 'metadata'" [attr.aria-current]="view() === 'metadata' ? 'page' : null" (click)="setView('metadata')">Metadata & discovery</button>
        <button type="button" [class.active]="view() === 'runs'" [attr.aria-current]="view() === 'runs' ? 'page' : null" (click)="setView('runs')">Runs & schedules</button>
        <button type="button" [class.active]="view() === 'cleaning'" [attr.aria-current]="view() === 'cleaning' ? 'page' : null" (click)="setView('cleaning')">Explicit cleaning</button>
      </nav>

      @if (operationError(); as failure) {
        <section class="failure-card" role="alert" aria-live="assertive">
          <div class="failure-title">
            <div><p class="eyebrow">OPERATION FAILED</p><h2>{{ failure.category }}</h2></div>
            @if (failure.httpStatus) { <span class="status-code">HTTP {{ failure.httpStatus }}</span> }
          </div>
          <p>{{ failure.diagnostic }}</p>
          <dl class="failure-details">
            @if (failure.operation) { <div><dt>Operation</dt><dd>{{ failure.operation }}</dd></div> }
            @if (failure.stage) { <div><dt>Stage</dt><dd>{{ failure.stage }}</dd></div> }
            @if (failure.commitCertainty) { <div><dt>Commit certainty</dt><dd>{{ failure.commitCertainty }}</dd></div> }
            @if (failure.reconciliation) { <div><dt>Recovery guidance</dt><dd>{{ failure.reconciliation }}</dd></div> }
            @if (failure.retryable !== undefined) { <div><dt>Retryable</dt><dd>{{ failure.retryable ? 'Yes, as reported by the contract' : 'No' }}</dd></div> }
          </dl>
          <p class="health-note">No automatic retry was made. Follow the local recovery guidance before repeating an uncertain operation.</p>
        </section>
      }

      @if (busy(); as operation) {
        <p class="busy-note" role="status" aria-live="polite">{{ operation }} in progress…</p>
      }

      @if (view() === 'metadata') {
        <section class="view-intro">
          <p class="eyebrow">CONTROL PLANE</p>
          <h2>Metadata and schema discovery</h2>
          <p>Validation is authoritative at the backend. Drafts are immutable; discovery proposes changes but never activates them.</p>
        </section>

        <section class="panel-grid">
          <article class="panel">
            <div class="panel-heading"><div><p class="eyebrow">METADATA V1 · MAX 1 MIB</p><h3>Validate and author a draft</h3></div></div>
            <label for="metadata-json">Metadata JSON</label>
            <textarea id="metadata-json" class="code-input metadata-input" spellcheck="false" [value]="metadataText" (input)="updateMetadata($event)" aria-describedby="metadata-limit"></textarea>
            <p id="metadata-limit" class="field-help">Strict metadata v1 JSON only. Unsupported transformation or quality rules fail closed.</p>
            <div class="button-row">
              <button type="button" [disabled]="isBusy()" (click)="validateMetadata()">Validate metadata</button>
              <button type="button" class="secondary-button" [disabled]="!canAuthor() || isBusy()" (click)="authorMetadata()">Author valid draft</button>
            </div>
            @if (report(); as validation) {
              <div class="result-card" aria-live="polite">
                <div class="result-heading">
                  <strong [class.good]="validation.valid" [class.bad]="!validation.valid">{{ validation.valid ? 'Valid metadata' : 'Invalid metadata' }}</strong>
                  <span class="pill">{{ validation.compatibilityDecision }}</span>
                </div>
                <p class="digest">Input digest <code>{{ validation.inputDigest }}</code></p>
                <p class="field-help">runEligible: false · report storage: {{ validation.securityImpact }}</p>
                @if (!isCurrentValidation()) {
                  <p class="warning-note">The editor changed after this report. Validate the current input before authoring.</p>
                }
                @if (validation.diagnostics.length) {
                  <ul class="diagnostics">
                    @for (diagnostic of validation.diagnostics; track $index) {
                      <li><code>{{ diagnostic.code }}</code><span>{{ diagnostic.path }}</span><p>{{ diagnostic.message }}</p></li>
                    }
                  </ul>
                } @else {
                  <p class="field-help">No diagnostics returned.</p>
                }
              </div>
            }
            @if (authoredRevision(); as authored) {
              <div class="result-card">
                <strong>Draft authored</strong>
                <p class="digest"><code>{{ authored.revision }}</code> · {{ authored.status }}</p>
                <p class="digest">Digest <code>{{ authored.digest }}</code></p>
              </div>
            }
          </article>

          <article class="panel">
            <div class="panel-heading"><div><p class="eyebrow">IMMUTABLE IDENTITIES</p><h3>Validation reports and revisions</h3></div></div>
            <label for="report-digest">Retrieve report by exact input digest</label>
            <div class="input-action">
              <input id="report-digest" type="text" inputmode="text" autocomplete="off" maxlength="64" placeholder="64-character SHA-256 digest" [value]="reportDigest" (input)="reportDigest = valueOf($event)">
              <button type="button" class="secondary-button" [disabled]="!isDigest(reportDigest) || isBusy()" (click)="retrieveReport()">Retrieve</button>
            </div>
            <div class="subsection-heading">
              <div><h4>Revision identities</h4><p class="field-help">Only revision, status, and digest are available.</p></div>
              <button type="button" class="secondary-button" [disabled]="isBusy()" (click)="loadRevisions()">Refresh list</button>
            </div>
            @if (revisions().length) {
              <div class="table-wrap">
                <table>
                  <thead><tr><th>Revision</th><th>Status</th><th>Digest</th></tr></thead>
                  <tbody>
                    @for (revision of revisions(); track revision.revision) {
                      <tr><td><code>{{ revision.revision }}</code></td><td><span class="pill">{{ revision.status }}</span></td><td class="table-digest"><code>{{ revision.digest }}</code></td></tr>
                    }
                  </tbody>
                </table>
              </div>
            } @else {
              <p class="empty-state">No revisions loaded. Refresh to list returned identities.</p>
            }
            <div class="subsection-heading activation-heading">
              <div><h4>Explicit activation</h4><p class="field-help">Activation changes which revision future operations use.</p></div>
            </div>
            <label for="activate-revision">Draft revision ID</label>
            <input id="activate-revision" type="text" maxlength="128" [value]="activationRevision" (input)="activationRevision = valueOf($event)">
            @if (requiresExpectedActive()) {
              <label for="expected-active">Expected current active revision <span class="required-note">required to replace active metadata</span></label>
              <input id="expected-active" type="text" maxlength="128" [value]="expectedActiveRevision" (input)="expectedActiveRevision = valueOf($event)" [placeholder]="activeRevision() ?? ''">
            }
            <button type="button" class="activation-button" [disabled]="!canActivate() || isBusy()" (click)="activateMetadata()">Activate selected draft</button>
            @if (activationResult(); as activated) {
              <p class="success-note" role="status">Active revision: <code>{{ activated.revision }}</code> · {{ activated.status }} · {{ activated.digest }}</p>
            }
          </article>
        </section>

        <section class="panel full-panel">
          <div class="panel-heading"><div><p class="eyebrow">ADVISORY ONLY</p><h3>Discover the active local JSONL object</h3></div></div>
          <p class="field-help">Discovery reads the active revision and a bounded source snapshot. It does not update metadata. No proposal history is available.</p>
          <div class="form-grid">
            <div>
              <label for="discovery-source">Source root path</label>
              <input id="discovery-source" type="text" autocomplete="off" [value]="discoverySourceRoot" (input)="discoverySourceRoot = valueOf($event)" placeholder="/private/path/to/source">
            </div>
            <div>
              <label for="discovery-disposition">Proposed failure disposition</label>
              <select id="discovery-disposition" [value]="discoveryDisposition" (change)="discoveryDisposition = valueOf($event)">
                <option value="reject">reject</option><option value="hold">hold</option><option value="quarantine">quarantine</option><option value="continue">continue</option>
              </select>
            </div>
          </div>
          <button type="button" [disabled]="!discoverySourceRoot.trim() || isBusy()" (click)="discover()">Discover schema proposal</button>
          @if (discoveryResult(); as discovery) {
            <div class="result-card advisory-result" aria-live="polite">
              <div class="result-heading"><strong>Advisory proposal · not activated</strong><span class="pill">executionReady: false</span></div>
              <p>Proposal <code>{{ discovery.proposalId }}</code> · active revision <code>{{ discovery.proposal.activeRevision }}</code></p>
              <p>Source digest <code>{{ discovery.proposal.observation.sourceDigest }}</code> · compatible: {{ discovery.proposal.report.compatible }}</p>
              <h4>Proposed columns</h4>
              @if (discovery.proposal.proposedColumns.length) {
                <div class="table-wrap"><table><thead><tr><th>Column</th><th>Type</th><th>Classification</th></tr></thead><tbody>
                  @for (column of discovery.proposal.proposedColumns; track column.id) {
                    <tr><td><code>{{ column.id }}</code></td><td>{{ column.type }}</td><td>{{ column.classification ?? 'Not returned' }}</td></tr>
                  }
                </tbody></table></div>
              } @else { <p class="empty-state">No proposed columns returned.</p> }
              <h4>Schema changes</h4>
              @if (discovery.proposal.report.changes.length) {
                <ul class="change-list">@for (change of discovery.proposal.report.changes; track $index) { <li><code>{{ change.field }}</code><span>{{ change.kind }}{{ change.before ? ' · ' + change.before : '' }}{{ change.after ? ' → ' + change.after : '' }}</span></li> }</ul>
              } @else { <p class="field-help">No schema changes returned.</p> }
            </div>
          }
        </section>
      }

      @if (view() === 'runs') {
        <section class="view-intro">
          <p class="eyebrow">EXECUTION</p>
          <h2>Run, schedule, and replay</h2>
          <p>Each action is explicit. Schedule evaluation creates requests only; a run is a separate operation.</p>
        </section>

        <section class="panel-grid">
          <article class="panel">
            <div class="panel-heading"><div><p class="eyebrow">ONE SOURCE SNAPSHOT</p><h3>Run ingestion</h3></div></div>
            <p class="field-help">One JSONL object, at most 16 MiB and 1,000 records. The result excludes source values and history.</p>
            <label for="run-source">Source root path</label>
            <input id="run-source" type="text" autocomplete="off" [value]="runSourceRoot" (input)="runSourceRoot = valueOf($event)" placeholder="/private/path/to/source">
            <button type="button" [disabled]="!runSourceRoot.trim() || isBusy()" (click)="runIngestion()">Run one bounded ingestion</button>
            @if (runResult(); as result) {
              <div class="result-card">
                <strong>Run result</strong>
                <dl class="result-details"><div><dt>Run ID</dt><dd>{{ result.runId }}</dd></div><div><dt>Metadata revision</dt><dd>{{ result.metadataRevision }}</dd></div><div><dt>Records</dt><dd>{{ result.recordCount }}</dd></div><div><dt>Already committed</dt><dd>{{ result.alreadyCommitted ? 'Yes (reconciled)' : 'No' }}</dd></div></dl>
              </div>
            }
          </article>

          <article class="panel">
            <div class="panel-heading"><div><p class="eyebrow">REQUESTS ONLY · NO DISPATCHER</p><h3>Evaluate schedules</h3></div></div>
            <label for="schedule-at">Evaluation time <span class="field-help">(optional; default is current time)</span></label>
            <input id="schedule-at" type="datetime-local" [value]="scheduleAt" (input)="scheduleAt = valueOf($event)">
            <button type="button" [disabled]="isBusy()" (click)="evaluateSchedules()">Evaluate fixed-interval schedules</button>
            @if (scheduleResult(); as result) {
              <div class="result-card">
                <div class="result-heading"><strong>Schedule decisions</strong><span class="pill">{{ result.requests.length }} undelivered request(s)</span></div>
                @if (result.decisions.length) {
                  <div class="table-wrap"><table><thead><tr><th>Source / object</th><th>Eligible</th><th>Decision</th></tr></thead><tbody>
                    @for (decision of result.decisions; track $index) {
                      <tr><td><code>{{ decision.sourceId }}/{{ decision.objectId }}</code></td><td>{{ decision.eligible ? 'Yes' : 'No' }}</td><td>{{ decision.reason }}</td></tr>
                    }
                  </tbody></table></div>
                } @else { <p class="empty-state">No schedule decisions returned.</p> }
                @if (result.requests.length) {
                  <h4>Undelivered requests</h4>
                  <ul class="change-list">@for (request of result.requests; track request.requestId) { <li><code>{{ request.requestId }}</code><span>{{ request.sourceId }}/{{ request.objectId }} · {{ request.scheduledAt }} · revision {{ request.metadataRevision }}</span></li> }</ul>
                }
              </div>
            }
            <div class="stop-box">
              <h4>Stop pending schedule requests</h4>
              <p class="field-help">This acknowledges pending requests only. It does not cancel or stop an ingestion run.</p>
              <label for="stop-reason">Reason</label>
              <select id="stop-reason" [value]="stopReason" (change)="updateStopReason($event)">
                <option value="shutdown">shutdown</option><option value="cancelled">cancelled</option>
              </select>
              <button type="button" class="secondary-button" [disabled]="isBusy()" (click)="stopScheduleRequests()">Stop pending requests</button>
              @if (stopResult(); as stopped) { <p class="success-note" role="status">{{ stopped.status }}. Run state was not changed.</p> }
            </div>
          </article>
        </section>

        <section class="panel full-panel">
          <div class="panel-heading"><div><p class="eyebrow">PINNED RECONCILIATION ONLY</p><h3>Replay an existing run</h3></div></div>
          <p class="field-help">Replay verifies the original source snapshot, pinned metadata, capabilities, and rules, then reconciles the original idempotent batch. It does not produce a new output.</p>
          <div class="form-grid">
            <div><label for="replay-source">Source root path</label><input id="replay-source" type="text" autocomplete="off" [value]="replaySourceRoot" (input)="replaySourceRoot = valueOf($event)"></div>
            <div><label for="replay-run-id">Original run ID</label><input id="replay-run-id" type="text" maxlength="64" [value]="replayRunId" (input)="replayRunId = valueOf($event)" placeholder="64-character run ID"></div>
            <div><label for="replay-reason">Reason</label><select id="replay-reason" [value]="replayReason" (change)="updateReplayReason($event)"><option value="operator_request">operator request</option><option value="recovery">recovery</option><option value="incident">incident</option></select></div>
          </div>
          <button type="button" [disabled]="!replaySourceRoot.trim() || !isDigest(replayRunId) || isBusy()" (click)="replay()">Reconcile original run</button>
          @if (replayResult(); as result) {
            <div class="result-card"><strong>Original batch reconciled</strong><p>Replay plan <code>{{ result.replayPlanId }}</code> · original run <code>{{ result.originalRunId }}</code></p><p>Result run <code>{{ result.result.runId }}</code> · revision <code>{{ result.result.metadataRevision }}</code> · {{ result.result.recordCount }} record(s) · already committed: {{ result.result.alreadyCommitted }}</p></div>
          }
        </section>
      }

      @if (view() === 'cleaning') {
        <section class="view-intro">
          <p class="eyebrow">EXPLICIT DATA CHANGE</p>
          <h2>Preview before applying a correction</h2>
          <p>Cleaning is separate from validation and ingestion. It never replaces the input record.</p>
        </section>
        <section class="panel full-panel cleaning-panel">
          <div class="sensitive-warning" role="note"><strong>Sensitive local data</strong><span>Record content stays in this page's memory only. Do not use secrets or data you are not authorized to process. Clear the editor and result when finished.</span></div>
          <div class="form-grid">
            <div>
              <label for="clean-classification">Classification</label>
              <select id="clean-classification" [value]="cleaningClassification" (change)="updateCleaningClassification($event)"><option value="public">public</option><option value="internal">internal</option></select>
            </div>
            <div>
              <label for="correction-id">Correction ID</label>
              <input id="correction-id" type="text" maxlength="128" [value]="correctionId" (input)="updateCorrectionId($event)">
            </div>
            <div>
              <label for="clean-operation">Supported operation · version v1</label>
              <select id="clean-operation" [value]="cleaningOperation" (change)="updateCleaningOperation($event)"><option value="trim">trim</option><option value="lowercase">lowercase</option><option value="uppercase">uppercase</option><option value="replace_null">replace_null</option></select>
            </div>
            <div>
              <label for="clean-field">Field</label>
              <input id="clean-field" type="text" [value]="cleaningField" (input)="updateCleaningField($event)">
            </div>
          </div>
          <label for="clean-record">One JSON-object record · max 256 KiB</label>
          <textarea id="clean-record" class="code-input record-input" spellcheck="false" [value]="cleaningRecordText" (input)="updateCleaningRecord($event)" placeholder='{"name":"  Example  ","optional":null}'></textarea>
          @if (cleaningOperation === 'replace_null') {
            <label for="replacement-value">Replacement value as JSON</label>
            <input
              id="replacement-value"
              type="text"
              [value]="replacementValueText"
              (input)="updateReplacementValue($event)"
              placeholder="Replacement JSON value"
            />
          }
          <div class="button-row">
            <button type="button" [disabled]="!canPreviewCleaning() || isBusy()" (click)="previewCleaning()">Preview correction</button>
            <button type="button" class="secondary-button" [disabled]="isBusy()" (click)="clearCleaning()">Clear sensitive input and result</button>
          </div>
          @if (cleaningResult(); as result) {
            <div class="result-card sensitive-result" aria-live="polite">
              <div class="result-heading"><strong>{{ result.preview ? 'Preview only · nothing persisted' : 'Separate cleaned output applied' }}</strong><span class="pill">{{ result.classification }}</span></div>
              <p class="field-help">The original record remains unchanged. This record-bearing output is sensitive local data.</p>
              <h4>{{ result.preview ? 'Preview record' : 'Cleaned output record' }}</h4>
              <pre class="result-json">{{ pretty(result.record) }}</pre>
              <h4>Audit identity</h4>
              <dl class="result-details"><div><dt>Correction</dt><dd>{{ result.audit.correctionId }} · {{ result.audit.correctionVersion }}</dd></div><div><dt>Output digest</dt><dd>{{ result.audit.outputDigest }}</dd></div><div><dt>Occurred</dt><dd>{{ result.audit.occurredAt }}</dd></div><div><dt>Already applied</dt><dd>{{ result.alreadyApplied ? 'Yes (reconciled)' : 'No' }}</dd></div></dl>
              @if (result.preview) {
                <label class="confirm-row"><input type="checkbox" [checked]="applyConfirmed" (change)="applyConfirmed = checkedOf($event)"> I reviewed this preview and explicitly authorize writing separate cleaned output.</label>
                <button type="button" class="danger-button" [disabled]="!canApplyCleaning() || isBusy()" (click)="applyCleaning()">Apply this correction</button>
              }
            </div>
          }
        </section>
      }

      <footer>
        <span>Mimir local reference profile · single-user local use</span>
        <span>No hosted identity, run history, dispatcher, or record browsing</span>
      </footer>
    </main>
  `,
})
export class AppComponent implements OnInit {
  private readonly api = inject(ApiService);
  readonly health = signal<ProfileHealth | null>(null);
  readonly healthError = signal<FailureView | null>(null);
  readonly healthBusy = signal(false);
  readonly view = signal<View>('metadata');
  readonly busy = signal<string | null>(null);
  readonly operationError = signal<FailureView | null>(null);

  readonly report = signal<ValidationReport | null>(null);
  readonly revisions = signal<RevisionSummary[]>([]);
  readonly authoredRevision = signal<RevisionSummary | null>(null);
  readonly activationResult = signal<ActivationResult | null>(null);
  readonly discoveryResult = signal<DiscoveryResult | null>(null);
  readonly runResult = signal<RunResult | null>(null);
  readonly scheduleResult = signal<ScheduleResult | null>(null);
  readonly stopResult = signal<StopResult | null>(null);
  readonly replayResult = signal<ReplayResult | null>(null);
  readonly cleaningResult = signal<CleaningResult | null>(null);

  metadataText = INITIAL_METADATA;
  reportDigest = '';
  activationRevision = '';
  expectedActiveRevision = '';
  discoverySourceRoot = '';
  discoveryDisposition = 'reject';
  runSourceRoot = '';
  scheduleAt = '';
  stopReason: 'cancelled' | 'shutdown' = 'shutdown';
  replaySourceRoot = '';
  replayRunId = '';
  replayReason: 'operator_request' | 'recovery' | 'incident' = 'operator_request';
  cleaningClassification: 'public' | 'internal' = 'internal';
  cleaningOperation: 'trim' | 'lowercase' | 'uppercase' | 'replace_null' = 'trim';
  correctionId = '';
  cleaningField = '';
  cleaningRecordText = '';
  replacementValueText = '';
  applyConfirmed = false;

  private lastValidatedText: string | null = null;
  private previewRequestKey: string | null = null;

  ngOnInit(): void {
    this.loadHealth();
  }

  setView(view: View): void {
    if (view !== 'cleaning') this.clearCleaning();
    this.view.set(view);
    this.operationError.set(null);
  }

  loadHealth(): void {
    this.healthBusy.set(true);
    this.healthError.set(null);
    this.api.health().subscribe({
      next: (profile) => {
        this.health.set(profile);
        this.healthBusy.set(false);
      },
      error: (error: unknown) => {
        this.healthError.set(this.failureView(error));
        this.healthBusy.set(false);
      },
    });
  }

  contractVersion(contracts: string[]): string {
    return contracts.find((contract) => contract.startsWith('mimir.gui-api/')) ?? 'Unknown';
  }

  isBusy(): boolean {
    return this.busy() !== null;
  }

  updateMetadata(event: Event): void {
    this.metadataText = this.valueOf(event);
    this.authoredRevision.set(null);
  }

  validateMetadata(): void {
    const submittedText = this.metadataText;
    const metadata = this.parseJson(submittedText);
    if (metadata === undefined) return;
    this.lastValidatedText = null;
    this.execute('Validate metadata', this.api.validate(metadata), (report) => {
      this.report.set(report);
      this.lastValidatedText = submittedText;
      this.authoredRevision.set(null);
    });
  }

  isCurrentValidation(): boolean {
    return this.lastValidatedText !== null && this.lastValidatedText === this.metadataText;
  }

  canAuthor(): boolean {
    return this.report()?.valid === true && this.isCurrentValidation();
  }

  authorMetadata(): void {
    if (!this.canAuthor()) return;
    const metadata = this.parseJson(this.metadataText);
    if (metadata === undefined) return;
    this.execute('Author metadata', this.api.author(metadata), (summary) => {
      this.authoredRevision.set(summary);
      this.revisions.update((items) => [
        ...items.filter((item) => item.revision !== summary.revision || item.status !== summary.status),
        summary,
      ]);
    });
  }

  isDigest(value: string): boolean {
    return /^[a-fA-F0-9]{64}$/.test(value);
  }

  retrieveReport(): void {
    if (!this.isDigest(this.reportDigest)) return;
    this.execute('Retrieve validation report', this.api.validationReport(this.reportDigest), (report) => {
      this.report.set(report);
      this.lastValidatedText = null;
    });
  }

  loadRevisions(): void {
    this.execute('List metadata revisions', this.api.revisions(), (items) => this.revisions.set(items));
  }

  activeRevision(): string | null {
    return this.revisions().find((item) => item.status.toLowerCase() === 'active')?.revision ?? null;
  }

  requiresExpectedActive(): boolean {
    const active = this.activeRevision();
    return active !== null && this.activationRevision.trim() !== active;
  }

  canActivate(): boolean {
    return this.activationRevision.trim().length > 0
      && (!this.requiresExpectedActive() || this.expectedActiveRevision.trim().length > 0);
  }

  activateMetadata(): void {
    if (!this.canActivate()) return;
    const request: { revision: string; expectedActive?: string } = {
      revision: this.activationRevision.trim(),
    };
    if (this.requiresExpectedActive()) request.expectedActive = this.expectedActiveRevision.trim();
    this.execute('Activate metadata', this.api.activate(request), (result) => {
      this.activationResult.set(result);
    });
  }

  discover(): void {
    const sourceRoot = this.discoverySourceRoot.trim();
    if (!sourceRoot) return;
    this.execute('Discover schema', this.api.discover({
      sourceRoot,
      disposition: this.discoveryDisposition,
    }), (result) => this.discoveryResult.set(result));
  }

  runIngestion(): void {
    const sourceRoot = this.runSourceRoot.trim();
    if (!sourceRoot) return;
    this.execute('Run ingestion', this.api.run({ sourceRoot }), (result) => this.runResult.set(result));
  }

  evaluateSchedules(): void {
    const request: { at?: string } = {};
    if (this.scheduleAt) {
      const evaluatedAt = new Date(this.scheduleAt);
      if (Number.isNaN(evaluatedAt.getTime())) {
        this.localError('invalid_input', 'Enter a valid schedule evaluation time.');
        return;
      }
      request.at = evaluatedAt.toISOString();
    }
    this.execute('Evaluate schedules', this.api.evaluateSchedules(request), (result) => this.scheduleResult.set(result));
  }

  stopScheduleRequests(): void {
    this.execute('Stop pending schedule requests', this.api.stopScheduleRequests(this.stopReason), (result) => {
      this.stopResult.set(result);
      this.scheduleResult.set(null);
    });
  }

  updateStopReason(event: Event): void {
    this.stopReason = this.valueOf(event) === 'cancelled' ? 'cancelled' : 'shutdown';
  }

  updateReplayReason(event: Event): void {
    const value = this.valueOf(event);
    if (value === 'recovery' || value === 'incident' || value === 'operator_request') this.replayReason = value;
  }

  replay(): void {
    const sourceRoot = this.replaySourceRoot.trim();
    const runId = this.replayRunId.trim();
    if (!sourceRoot || !this.isDigest(runId)) return;
    this.execute('Reconcile original run', this.api.replay({
      sourceRoot,
      runId,
      reason: this.replayReason,
    }), (result) => this.replayResult.set(result));
  }

  previewCleaning(): void {
    const request = this.cleaningRequest();
    if (!request) return;
    this.cleaningResult.set(null);
    this.previewRequestKey = null;
    this.applyConfirmed = false;
    this.execute('Preview cleaning', this.api.previewCleaning(request), (result) => {
      this.cleaningResult.set(result);
      if (result.preview) this.previewRequestKey = this.pretty(request);
    });
  }

  canPreviewCleaning(): boolean {
    return this.correctionId.trim().length > 0
      && this.cleaningField.trim().length > 0
      && this.cleaningRecordText.trim().length > 0;
  }

  canApplyCleaning(): boolean {
    const result = this.cleaningResult();
    if (!this.applyConfirmed || result?.preview !== true || this.previewRequestKey === null) return false;
    const request = this.cleaningRequest();
    return request !== null && this.previewRequestKey === this.pretty(request);
  }

  applyCleaning(): void {
    if (!this.canApplyCleaning()) return;
    const request = this.cleaningRequest();
    if (!request) return;
    this.execute('Apply cleaning', this.api.applyCleaning(request), (result) => {
      this.cleaningResult.set(result);
      this.applyConfirmed = false;
    });
  }

  clearCleaning(): void {
    this.cleaningRecordText = '';
    this.replacementValueText = '';
    this.correctionId = '';
    this.cleaningField = '';
    this.cleaningResult.set(null);
    this.previewRequestKey = null;
    this.applyConfirmed = false;
  }

  updateCleaningRecord(event: Event): void {
    this.cleaningRecordText = this.valueOf(event);
    this.invalidateCleaningPreview();
  }

  updateCorrectionId(event: Event): void {
    this.correctionId = this.valueOf(event);
    this.invalidateCleaningPreview();
  }

  updateCleaningField(event: Event): void {
    this.cleaningField = this.valueOf(event);
    this.invalidateCleaningPreview();
  }

  updateReplacementValue(event: Event): void {
    this.replacementValueText = this.valueOf(event);
    this.invalidateCleaningPreview();
  }

  updateCleaningClassification(event: Event): void {
    this.cleaningClassification = this.valueOf(event) === 'public' ? 'public' : 'internal';
    this.invalidateCleaningPreview();
  }

  updateCleaningOperation(event: Event): void {
    const value = this.valueOf(event);
    if (value === 'trim' || value === 'lowercase' || value === 'uppercase' || value === 'replace_null') {
      this.cleaningOperation = value;
      this.invalidateCleaningPreview();
    }
  }

  valueOf(event: Event): string {
    return (event.target as HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement).value;
  }

  checkedOf(event: Event): boolean {
    return (event.target as HTMLInputElement).checked;
  }

  pretty(value: unknown): string {
    return JSON.stringify(value, null, 2) ?? '';
  }

  private cleaningRequest(): CleaningRequest | null {
    const record = this.parseJson(this.cleaningRecordText);
    if (record === undefined) return null;
    if (!this.isJsonObject(record)) {
      this.localError('invalid_input', 'The cleaning input must be one JSON object.');
      return null;
    }
    const request: CleaningRequest = {
      record,
      classification: this.cleaningClassification,
      correction: {
        id: this.correctionId.trim(),
        version: 'v1',
        operation: this.cleaningOperation,
        field: this.cleaningField.trim(),
      },
    };
    if (this.cleaningOperation === 'replace_null') {
      const value = this.parseJson(this.replacementValueText);
      if (value === undefined) return null;
      request.correction.value = value;
    }
    return request;
  }

  private invalidateCleaningPreview(): void {
    this.cleaningResult.set(null);
    this.previewRequestKey = null;
    this.applyConfirmed = false;
  }

  private parseJson(value: string): unknown | undefined {
    try {
      return JSON.parse(value) as unknown;
    } catch {
      this.localError('invalid_input', 'Enter syntactically valid JSON. Input contents are not included in this error.');
      return undefined;
    }
  }

  private isJsonObject(value: unknown): value is Record<string, unknown> {
    return typeof value === 'object' && value !== null && !Array.isArray(value);
  }

  private localError(category: string, diagnostic: string): void {
    this.operationError.set({ category, diagnostic });
  }

  private execute<T>(name: string, request: import('rxjs').Observable<T>, onSuccess: (value: T) => void): void {
    if (this.isBusy()) return;
    this.operationError.set(null);
    this.busy.set(name);
    request.subscribe({
      next: (value) => {
        onSuccess(value);
        this.busy.set(null);
      },
      error: (error: unknown) => {
        this.operationError.set(this.failureView(error));
        this.busy.set(null);
      },
    });
  }

  private failureView(error: unknown): FailureView {
    if (!(error instanceof HttpErrorResponse)) {
      return { category: 'request_failed', diagnostic: 'The operation failed before a safe API response was received.' };
    }
    const body = error.error;
    if (this.isJsonObject(body) && body['contractVersion'] === 'mimir.failure/v1') {
      const category = typeof body['category'] === 'string' ? body['category'] : 'request_failed';
      const diagnostic = typeof body['safeDiagnostic'] === 'string'
        ? body['safeDiagnostic']
        : 'The API returned a failure without a safe diagnostic.';
      return {
        category,
        diagnostic,
        operation: typeof body['operation'] === 'string' ? body['operation'] : undefined,
        stage: typeof body['stage'] === 'string' ? body['stage'] : undefined,
        retryable: typeof body['retryable'] === 'boolean' ? body['retryable'] : undefined,
        commitCertainty: typeof body['commitCertainty'] === 'string' ? body['commitCertainty'] : undefined,
        reconciliation: typeof body['reconciliation'] === 'string' ? body['reconciliation'] : undefined,
        httpStatus: error.status || undefined,
      };
    }
    return error.status
      ? { category: 'http_error', diagnostic: 'The backend returned an error without a recognized safe failure envelope.', httpStatus: error.status }
      : { category: 'unavailable', diagnostic: 'The backend is unavailable. Check that the local services are running.' };
  }
}
