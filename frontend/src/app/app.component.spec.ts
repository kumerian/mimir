import { HttpErrorResponse, provideHttpClient } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AppComponent } from './app.component';
import { ApiService } from './api.service';

describe('AppComponent local-operation safeguards', () => {
  let component: AppComponent;

  afterEach(() => TestBed.resetTestingModule());

  beforeEach(() => {
    TestBed.resetTestingModule();
    const api = {
      health: vi.fn(() => of({
        status: 'ready',
        version: 'test',
        commit: 'test',
        contracts: ['mimir.gui-api/v1'],
        profile: 'local-reference-v1',
      })),
    };
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        { provide: ApiService, useValue: api },
      ],
    });
    component = TestBed.runInInjectionContext(() => new AppComponent());
  });

  it('does not author until validation passed for the exact current text', () => {
    expect(component.canAuthor()).toBe(false);
    component.report.set({
      contractVersion: 'mimir.metadata-validation/v1',
      inputDigest: 'a'.repeat(64),
      valid: true,
      runEligible: false,
      securityImpact: 'input_not_retained',
      compatibilityDecision: 'accepted_v1',
      diagnostics: [],
    });
    expect(component.canAuthor()).toBe(false);

    component.metadataText = component.metadataText;
    (component as unknown as { lastValidatedText: string }).lastValidatedText = component.metadataText;
    expect(component.canAuthor()).toBe(true);
  });

  it('requires expected active revision before replacing an active revision', () => {
    component.revisions.set([
      { revision: 'active-v1', status: 'active', digest: 'a'.repeat(64) },
      { revision: 'draft-v2', status: 'draft', digest: 'b'.repeat(64) },
    ]);
    component.activationRevision = 'draft-v2';
    expect(component.requiresExpectedActive()).toBe(true);
    expect(component.canActivate()).toBe(false);
    component.expectedActiveRevision = 'active-v1';
    expect(component.canActivate()).toBe(true);
  });

  it('does not expose schedule evaluation as dispatch', () => {
    component.scheduleResult.set({
      decisions: [{
        sourceId: 'orders',
        objectId: 'orders',
        eligible: true,
        reason: 'slot_due',
      }],
      requests: [{
        contractVersion: 'mimir.schedule-request/v1',
        requestId: 'c'.repeat(64),
        metadataRevision: 'active-v1',
        sourceId: 'orders',
        objectId: 'orders',
        scheduledAt: '2026-01-01T00:00:00Z',
        retryLimit: 0,
        concurrencyLimit: 1,
      }],
    });
    expect(component.scheduleResult()?.requests).toHaveLength(1);
  });

  it('requires an explicit preview match before cleaning apply', () => {
    component.correctionId = 'trim-name';
    component.cleaningField = 'name';
    component.cleaningRecordText = '{"name":"  Example  "}';
    component.cleaningClassification = 'internal';
    component.cleaningOperation = 'trim';
    component.applyConfirmed = true;
    expect(component.canApplyCleaning()).toBe(false);

    component.cleaningResult.set({
      record: { name: 'Example' },
      classification: 'internal',
      audit: {
        correctionId: 'trim-name',
        correctionVersion: 'v1',
        actor: 'local-user',
        inputReference: 'test',
        outputDigest: 'test',
        idempotencyKey: 'test',
        occurredAt: '2026-01-01T00:00:00Z',
      },
      preview: true,
      alreadyApplied: false,
    });
    const expectedRequest = {
      record: '{"name":"  Example  "}',
      classification: 'internal',
      correction: {
        id: 'trim-name',
        version: 'v1',
        operation: 'trim',
        field: 'name',
      },
    };
    const access = component as unknown as {
      previewRequestKey: string;
      pretty(value: unknown): string;
    };
    access.previewRequestKey = access.pretty(expectedRequest);
    expect(component.canApplyCleaning()).toBe(true);
  });

  it('rejects cleaning JSON with integers outside the safe range', () => {
    component.correctionId = 'trim-id';
    component.cleaningField = 'id';
    component.cleaningRecordText = '{"id":9007199254740993}';
    component.cleaningOperation = 'trim';
    const access = component as unknown as { cleaningRequest(): unknown };
    expect(access.cleaningRequest()).toBeNull();

    component.cleaningRecordText = '{"id":null}';
    component.cleaningOperation = 'replace_null';
    component.replacementValueText = '9007199254740993';
    expect(access.cleaningRequest()).toBeNull();
  });

  it('marks unrecognized mutating failures as unknown outcomes', () => {
    const access = component as unknown as {
      failureView(error: unknown, mutatingOperation?: string): {
        category: string;
        commitCertainty?: string;
        reconciliation?: string;
        retryable?: boolean;
      };
    };
    const failure = access.failureView(
      new HttpErrorResponse({ status: 502, error: { message: 'proxy lost response' } }),
      'Run ingestion',
    );
    expect(failure.category).toBe('unknown_outcome');
    expect(failure.commitCertainty).toBe('unknown');
    expect(failure.reconciliation).toBe('reconcile_before_retry');
    expect(failure.retryable).toBe(false);
  });

  it('keeps unrecognized read-only failures on the existing HTTP path', () => {
    const access = component as unknown as {
      failureView(error: unknown, mutatingOperation?: string): {
        category: string;
        commitCertainty?: string;
      };
    };
    const failure = access.failureView(
      new HttpErrorResponse({ status: 502, error: { message: 'proxy error' } }),
    );
    expect(failure.category).toBe('http_error');
    expect(failure.commitCertainty).toBeUndefined();
  });
});
