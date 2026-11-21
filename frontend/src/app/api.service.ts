import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';

const CONTRACT_VERSION = 'mimir.gui-api/v1';

export interface ProfileHealth {
  status: string;
  version: string;
  commit: string;
  contracts: string[];
  profile: string;
}

export interface ValidationReport {
  contractVersion: string;
  inputDigest: string;
  valid: boolean;
  runEligible: false;
  securityImpact: string;
  compatibilityDecision: string;
  diagnostics: Array<{ code: string; path: string; message: string }>;
}

export interface RevisionSummary {
  revision: string;
  status: string;
  digest: string;
}

export interface ActivationResult extends RevisionSummary {}

export interface DiscoveryResult {
  proposalId: string;
  proposal: {
    activeRevision: string;
    observation: {
      provider: string;
      providerVersion: string;
      scope: string;
      sourceDigest: string;
      observedAt: string;
      columns: Array<{ id: string; type: string; classification?: string }>;
    };
    report: {
      compatible: boolean;
      disposition: string;
      changes: Array<{ field: string; kind: string; before?: string; after?: string }>;
    };
    proposedColumns: Array<{ id: string; type: string; classification?: string }>;
    executionReady: false;
  };
}

export interface RunResult {
  runId: string;
  metadataRevision: string;
  recordCount: number;
  alreadyCommitted: boolean;
}

export interface ScheduleResult {
  decisions: Array<{
    sourceId: string;
    objectId: string;
    eligible: boolean;
    reason: string;
    request?: ScheduleRunRequest;
  }>;
  requests: ScheduleRunRequest[];
}

export interface ScheduleRunRequest {
  contractVersion: string;
  requestId: string;
  metadataRevision: string;
  sourceId: string;
  objectId: string;
  scheduledAt: string;
  retryLimit: number;
  concurrencyLimit: number;
}

export interface StopResult {
  status: string;
}

export interface ReplayResult {
  replayPlanId: string;
  originalRunId: string;
  result: RunResult;
}

export interface CleaningRequest {
  record: string;
  classification: 'public' | 'internal';
  correction: {
    id: string;
    version: 'v1';
    operation: 'trim' | 'lowercase' | 'uppercase' | 'replace_null';
    field: string;
    value?: string;
  };
}

export interface CleaningResult {
  record: Record<string, unknown>;
  classification: 'public' | 'internal';
  audit: {
    correctionId: string;
    correctionVersion: string;
    actor: string;
    inputReference: string;
    outputDigest: string;
    idempotencyKey: string;
    occurredAt: string;
  };
  preview: boolean;
  alreadyApplied: boolean;
}

@Injectable({ providedIn: 'root' })
export class ApiService {
  private readonly http = inject(HttpClient);
  private readonly headers = new HttpHeaders({
    'Mimir-Contract-Version': CONTRACT_VERSION,
  });
  private readonly jsonHeaders = new HttpHeaders({
    'Mimir-Contract-Version': CONTRACT_VERSION,
    'Content-Type': 'application/json',
  });

  health(): Observable<ProfileHealth> {
    return this.http.get<ProfileHealth>('/api/v1/health', { headers: this.headers });
  }

  validate(metadata: string): Observable<ValidationReport> {
    return this.http.post<ValidationReport>('/api/v1/metadata/validate', metadata, { headers: this.jsonHeaders });
  }

  validationReport(digest: string): Observable<ValidationReport> {
    return this.http.get<ValidationReport>(`/api/v1/metadata/validation-reports/${encodeURIComponent(digest)}`, {
      headers: this.headers,
    });
  }

  revisions(): Observable<RevisionSummary[]> {
    return this.http.get<RevisionSummary[]>('/api/v1/metadata/revisions', { headers: this.headers });
  }

  author(metadata: string): Observable<RevisionSummary> {
    return this.http.post<RevisionSummary>('/api/v1/metadata/revisions', metadata, { headers: this.jsonHeaders });
  }

  activate(request: { revision: string; expectedActive?: string }): Observable<ActivationResult> {
    return this.http.post<ActivationResult>('/api/v1/metadata/activate', request, { headers: this.headers });
  }

  discover(request: { sourceRoot: string; disposition: string }): Observable<DiscoveryResult> {
    return this.http.post<DiscoveryResult>('/api/v1/discovery', request, { headers: this.headers });
  }

  run(request: { sourceRoot: string }): Observable<RunResult> {
    return this.http.post<RunResult>('/api/v1/runs', request, { headers: this.headers });
  }

  evaluateSchedules(request: { at?: string }): Observable<ScheduleResult> {
    return this.http.post<ScheduleResult>('/api/v1/schedules/evaluate', request, { headers: this.headers });
  }

  stopScheduleRequests(reason: 'cancelled' | 'shutdown'): Observable<StopResult> {
    return this.http.post<StopResult>('/api/v1/schedules/stop', { reason }, { headers: this.headers });
  }

  replay(request: {
    sourceRoot: string;
    runId: string;
    reason: 'operator_request' | 'recovery' | 'incident';
  }): Observable<ReplayResult> {
    return this.http.post<ReplayResult>('/api/v1/replays', request, { headers: this.headers });
  }

  previewCleaning(request: CleaningRequest): Observable<CleaningResult> {
    return this.http.post<CleaningResult>('/api/v1/cleaning/preview', this.cleaningBody(request), { headers: this.jsonHeaders });
  }

  applyCleaning(request: CleaningRequest): Observable<CleaningResult> {
    return this.http.post<CleaningResult>('/api/v1/cleaning/apply', this.cleaningBody(request), { headers: this.jsonHeaders });
  }

  private cleaningBody(request: CleaningRequest): string {
    const correctionFields = [
      `"id":${JSON.stringify(request.correction.id)}`,
      `"version":${JSON.stringify(request.correction.version)}`,
      `"operation":${JSON.stringify(request.correction.operation)}`,
      `"field":${JSON.stringify(request.correction.field)}`,
    ];
    if (request.correction.value !== undefined) {
      correctionFields.push(`"value":${request.correction.value}`);
    }
    return `{"record":${request.record},"classification":${JSON.stringify(request.classification)},"correction":{${correctionFields.join(',')}}}`;
  }
}
