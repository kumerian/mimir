import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { App } from './app';

describe('App', () => {
  let http: HttpTestingController;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [App],
      providers: [provideHttpClient(), provideHttpClientTesting()],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('shows loading, then displays the database status from the backend', async () => {
    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();
    const element = fixture.nativeElement as HTMLElement;
    expect(element.querySelector('[role="status"]')?.textContent).toContain('Connecting');

    http.expectOne('/api/status').flush({ application: 'mimir', database: 'connected' });
    await fixture.whenStable();

    expect(element.querySelector('h1')?.textContent).toBe('Mimir');
    expect(element.querySelector('[role="status"]')?.textContent).toBe('mimir: database connected');
    expect(element.querySelector('.error')).toBeNull();
  });

  it('displays an explicit error when the backend request fails', async () => {
    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();

    http.expectOne('/api/status').flush('Unavailable', {
      status: 503,
      statusText: 'Service Unavailable',
    });
    await fixture.whenStable();

    const element = fixture.nativeElement as HTMLElement;
    expect(element.querySelector('.error')?.textContent).toContain('Cannot reach the backend');
  });
});
