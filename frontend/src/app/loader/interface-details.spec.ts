import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { ActivatedRoute, provideRouter } from '@angular/router';
import { InterfaceDetails } from './interface-details';

describe('InterfaceDetails', () => {
  let http: HttpTestingController;
  const sourceInterface = {
    interfaceId: 'interface-id',
    interfaceName: 'sample',
    sourceName: 'source',
    authenticationId: null,
    interfaceFunctionalName: 'Sample',
    interfaceType: 'CSV',
    interfaceLocation: '/data/sample.csv',
    decimalSeparator: '.',
    thousandsSeparator: ',',
    delimiter: ';',
    hasHeaders: true,
    isActive: true,
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [InterfaceDetails],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([]),
        {
          provide: ActivatedRoute,
          useValue: { snapshot: { paramMap: { get: () => 'interface-id' } } },
        },
      ],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  function loadDetails() {
    http.expectOne('/api/metadata/interfaces?id=interface-id').flush(sourceInterface);
    http.expectOne('/api/metadata/objects').flush([]);
    http.expectOne('/api/metadata/columns').flush([]);
  }

  it('only enables actions for a saved CSV Interface and disables buttons until requests finish', () => {
    const fixture = TestBed.createComponent(InterfaceDetails);
    fixture.detectChanges();
    loadDetails();
    fixture.detectChanges();

    const deriveButton = [...fixture.nativeElement.querySelectorAll('button')].find(
      (button: HTMLButtonElement) => button.textContent.includes('Create/update'),
    ) as HTMLButtonElement;
    expect(deriveButton.disabled).toBe(false);
    deriveButton.click();
    fixture.detectChanges();

    const derive = http.expectOne('/api/loader/interfaces/interface-id/metadata');
    expect(derive.request.method).toBe('POST');
    expect(deriveButton.disabled).toBe(true);
    derive.flush({
      object: { objectId: 'object-id', objectName: 'sample', interfaceId: 'interface-id' },
      columns: [{ columnId: 'column-id', columnName: 'name', objectId: 'object-id' }],
      rowsScanned: 2,
      removedColumns: 0,
      updated: false,
    });
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Metadata saved: 1 Column from 2 data rows.');
    http.expectOne('/api/metadata/interfaces?id=interface-id').flush(sourceInterface);
    http.expectOne('/api/metadata/objects').flush([
      { objectId: 'object-id', objectName: 'sample', interfaceId: 'interface-id' },
    ]);
    http.expectOne('/api/metadata/columns').flush([
      { columnId: 'column-id', columnName: 'name', objectId: 'object-id' },
    ]);
    fixture.detectChanges();

    const loadButton = [...fixture.nativeElement.querySelectorAll('button')].find(
      (button: HTMLButtonElement) => button.textContent.includes('Load CSV data'),
    ) as HTMLButtonElement;
    loadButton.click();
    fixture.detectChanges();
    const load = http.expectOne('/api/loader/interfaces/interface-id/load');
    expect(loadButton.disabled).toBe(true);
    load.flush({ objectName: 'sample', tableName: 'object-stable-id', rowsLoaded: 2 });
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Loaded 2 data rows into sample');
    http.expectOne('/api/metadata/interfaces?id=interface-id').flush(sourceInterface);
    http.expectOne('/api/metadata/objects').flush([
      { objectId: 'object-id', objectName: 'sample', interfaceId: 'interface-id' },
    ]);
    http.expectOne('/api/metadata/columns').flush([
      { columnId: 'column-id', columnName: 'name', objectId: 'object-id' },
    ]);
  });

  it('shows backend errors without a success message and omits CSV actions for other types', () => {
    const fixture = TestBed.createComponent(InterfaceDetails);
    fixture.detectChanges();
    loadDetails();
    fixture.detectChanges();

    const deriveButton = [...fixture.nativeElement.querySelectorAll('button')].find(
      (button: HTMLButtonElement) => button.textContent.includes('Create/update'),
    ) as HTMLButtonElement;
    deriveButton.click();
    http.expectOne('/api/loader/interfaces/interface-id/metadata').flush(
      { detail: 'CSV headers must be unique.' },
      { status: 400, statusText: 'Bad Request' },
    );
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('[role="alert"]').textContent).toContain('unique');
    expect(fixture.nativeElement.textContent).not.toContain('Metadata saved');

    fixture.componentInstance.sourceInterface.set({ ...sourceInterface, interfaceType: 'jdbc' });
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('supports only CSV');
    expect(
      [...fixture.nativeElement.querySelectorAll('button')].some((button: HTMLButtonElement) =>
        button.textContent.includes('Load CSV data'),
      ),
    ).toBe(false);
  });
});
