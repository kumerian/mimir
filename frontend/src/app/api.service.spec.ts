import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { ApiService, CleaningRequest, hasUnsafeJsonInteger } from './api.service';

describe('ApiService', () => {
  let api: ApiService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.resetTestingModule();
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), ApiService],
    });
    api = TestBed.inject(ApiService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    if (http) http.verify();
    TestBed.resetTestingModule();
  });

  it('adds the version header and maps the health contract', () => {
    const request$ = api.health();
    request$.subscribe();
    const request = http.expectOne('/api/v1/health');
    expect(request.request.headers.get('Mimir-Contract-Version')).toBe('mimir.gui-api/v1');
    request.flush({
      status: 'ready',
      version: 'test',
      commit: 'test',
      contracts: ['mimir.gui-api/v1'],
      profile: 'local-reference-v1',
    });
  });

  it('sends metadata JSON unchanged with an application/json content type', () => {
    const metadata = '{"name":"Example","name":"Duplicate"}';
    api.validate(metadata).subscribe();
    const validate = http.expectOne('/api/v1/metadata/validate');
    expect(validate.request.method).toBe('POST');
    expect(validate.request.headers.get('Content-Type')).toBe('application/json');
    expect(validate.request.body).toBe(metadata);
    validate.flush({
      contractVersion: 'mimir.metadata-validation/v1',
      inputDigest: 'a'.repeat(64),
      valid: false,
      runEligible: false,
      securityImpact: 'input_not_retained',
      compatibilityDecision: 'rejected_v1',
      diagnostics: [],
    });

    api.author(metadata).subscribe();
    const author = http.expectOne('/api/v1/metadata/revisions');
    expect(author.request.method).toBe('POST');
    expect(author.request.headers.get('Content-Type')).toBe('application/json');
    expect(author.request.body).toBe(metadata);
    author.flush({ revision: 'draft-v1', status: 'draft', digest: 'a'.repeat(64) });
  });

  it('detects unsafe JSON integers outside strings', () => {
    expect(hasUnsafeJsonInteger('{"id":9007199254740993}')).toBe(true);
    expect(hasUnsafeJsonInteger('{"id":"9007199254740993"}')).toBe(false);
  });

  it('uses separate preview and apply cleaning endpoints', () => {
    const request: CleaningRequest = {
      record: '{"name":"  Example  "}',
      classification: 'internal',
      correction: {
        id: 'trim-name',
        version: 'v1',
        operation: 'trim',
        field: 'name',
      },
    };

    api.previewCleaning(request).subscribe();
    const preview = http.expectOne({
      method: 'POST',
      url: '/api/v1/cleaning/preview',
    });
    expect(preview.request.headers.get('Content-Type')).toBe('application/json');
    expect(preview.request.body).toBe(
      '{"record":{"name":"  Example  "},"classification":"internal","correction":{"id":"trim-name","version":"v1","operation":"trim","field":"name"}}',
    );
    preview.flush({
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

    api.applyCleaning(request).subscribe();
    const applied = http.expectOne({
      method: 'POST',
      url: '/api/v1/cleaning/apply',
    });
    expect(applied.request.headers.get('Content-Type')).toBe('application/json');
    expect(applied.request.body).toBe(
      '{"record":{"name":"  Example  "},"classification":"internal","correction":{"id":"trim-name","version":"v1","operation":"trim","field":"name"}}',
    );
    applied.flush({
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
      preview: false,
      alreadyApplied: false,
    });
  });

  it('does not add run history or unsupported request routes', () => {
    api.run({ sourceRoot: '/private/source' }).subscribe();
    const request = http.expectOne('/api/v1/runs');
    expect(request.request.method).toBe('POST');
    expect(request.request.body).toEqual({ sourceRoot: '/private/source' });
    request.flush({
      runId: '1111111111111111111111111111111111111111111111111111111111111111',
      metadataRevision: 'synthetic-orders-v1',
      recordCount: 1,
      alreadyCommitted: false,
    });
  });
});
