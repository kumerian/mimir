import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { AuditLog } from './audit-log';

describe('AuditLog', () => {
  let http: HttpTestingController;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AuditLog],
      providers: [provideHttpClient(), provideHttpClientTesting()],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('displays audit details returned by the backend', async () => {
    const fixture = TestBed.createComponent(AuditLog);
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Loading audit history');
    http.expectOne('/api/audit').flush([
      {
        id: 1,
        occurredAt: '2026-01-01T10:00:00Z',
        operation: 'UPDATE',
        entityType: 'SourceSystem',
        recordId: 'crm',
        details: '{"sourceFunctionalName":{"before":"CRM","after":"Updated"}}',
      },
    ]);
    await fixture.whenStable();
    const text = fixture.nativeElement.textContent;
    expect(text).toContain('UPDATE');
    expect(text).toContain('SourceSystem');
    expect(text).toContain('crm');
    expect(text).toContain('Updated');
  });

  it('reports load failures and allows retrying', async () => {
    const fixture = TestBed.createComponent(AuditLog);
    http.expectOne('/api/audit').flush('Unavailable', {
      status: 503,
      statusText: 'Service Unavailable',
    });
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('[role="alert"]').textContent).toContain(
      'Cannot load metadata audit history',
    );
    fixture.componentInstance.refresh();
    http.expectOne('/api/audit').flush([]);
    await fixture.whenStable();
    expect(fixture.nativeElement.textContent).toContain('No metadata changes');
  });
});
