import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { ENTITIES, MetadataRecord } from './metadata-api';
import { MetadataEditor } from './metadata-editor';

describe('MetadataEditor', () => {
  let http: HttpTestingController;
  const source = {
    sourceName: 'crm',
    sourceFunctionalName: 'CRM',
    sourceType: 'database',
    sourceOwner: 'Owner',
    sourceApplication: 'CRM',
    sourceLocation: 'https://crm',
    isActive: true,
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [MetadataEditor],
      providers: [provideHttpClient(), provideHttpClientTesting()],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
  });
  afterEach(() => http.verify());

  function load(records: Partial<Record<string, MetadataRecord[]>> = {}) {
    for (const entity of ENTITIES) {
      http.expectOne(`/api/metadata/${entity.resource}`).flush(records[entity.resource] ?? []);
    }
  }

  it('renders all entity views and creates every model through the API', async () => {
    const fixture = TestBed.createComponent(MetadataEditor);
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Working');
    load();
    await fixture.whenStable();
    for (const entity of ENTITIES) {
      fixture.componentInstance.select(entity);
      fixture.componentInstance.edit();
      fixture.componentInstance.values = Object.fromEntries(
        entity.fields.map((field) => [
          field.name,
          field.optional
            ? null
            : field.kind === 'boolean'
              ? true
              : field.kind === 'number'
                ? 1
                : field.maxLength
                  ? 'x'.repeat(field.maxLength)
                  : `${field.name}-value`,
        ]),
      );
      fixture.detectChanges();
      expect(fixture.nativeElement.querySelectorAll('input, select').length).toBe(
        entity.fields.length,
      );
      fixture.nativeElement
        .querySelector('form')
        .dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
      const request = http.expectOne(`/api/metadata/${entity.resource}`);
      expect(request.request.method).toBe('POST');
      expect(Object.keys(request.request.body).sort()).toEqual(
        entity.fields.map((field) => field.name).sort(),
      );
      if (entity.resource === 'interfaces')
        expect(request.request.body.authenticationId).toBeNull();
      request.flush(request.request.body);
      load();
      await fixture.whenStable();
      expect(fixture.nativeElement.textContent).toContain('Record saved.');
      expect(fixture.componentInstance.formOpen()).toBe(false);
    }
  });

  it('uses reference selectors populated from parent records', async () => {
    const fixture = TestBed.createComponent(MetadataEditor);
    load({ 'source-systems': [source] });
    fixture.componentInstance.select(ENTITIES.find((entity) => entity.resource === 'interfaces')!);
    fixture.componentInstance.edit();
    fixture.detectChanges();
    await fixture.whenStable();
    const element = fixture.nativeElement as HTMLElement;
    expect(element.querySelector('#sourceName')?.textContent).toContain('crm');
    expect(element.querySelector('#authenticationId')?.textContent).toContain('None');
    expect(element.textContent).toContain('production secrets');
  });

  it('masks authentication credentials in lists and uses password inputs', async () => {
    const fixture = TestBed.createComponent(MetadataEditor);
    const authentication = {
      authenticationId: 'auth',
      authenticationType: 'basic',
      authenticationLocation: 'local',
      authenticationUser: 'user',
      authenticationCredential: 'dev-credential',
      isActive: true,
    };
    const sourceInterface = {
      interfaceId: 'interface-id',
      interfaceName: 'interface',
      sourceName: 'crm',
    };
    load({ authentications: [authentication], interfaces: [sourceInterface] });
    fixture.detectChanges();
    fixture.componentInstance.select(ENTITIES.find((entity) => entity.resource === 'authentications')!);
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('tbody').textContent).toContain('********');
    expect(fixture.nativeElement.querySelector('tbody').textContent).not.toContain('dev-credential');
    fixture.nativeElement.querySelector('[aria-label="Edit auth"]').click();
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('#authenticationCredential').type).toBe('password');
    fixture.componentInstance.select(ENTITIES.find((entity) => entity.resource === 'api-parameters')!);
    fixture.componentInstance.edit();
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('#interfaceId')?.textContent).toContain('interface');
    expect(fixture.nativeElement.querySelector('#interfaceId')?.textContent).toContain('crm');
  });

  it('distinguishes same-named scoped records in lists and parent selectors', () => {
    const fixture = TestBed.createComponent(MetadataEditor);
    const first = { interfaceId: 'i1', interfaceName: 'shared', sourceName: 'source-a' };
    const second = { interfaceId: 'i2', interfaceName: 'shared', sourceName: 'source-b' };
    const firstObject = { objectId: 'o1', objectName: 'shared', interfaceId: 'i1' };
    const secondObject = { objectId: 'o2', objectName: 'shared', interfaceId: 'i2' };
    load({ interfaces: [first, second], objects: [firstObject, secondObject] });
    const editor = fixture.componentInstance;
    editor.select(ENTITIES.find((entity) => entity.resource === 'objects')!);
    expect(editor.recordLabel(firstObject)).toBe('source-a / shared / shared');
    expect(editor.recordLabel(secondObject)).toBe('source-b / shared / shared');
    editor.select(ENTITIES.find((entity) => entity.resource === 'columns')!);
    const field = ENTITIES.find((entity) => entity.resource === 'columns')!.fields.find(
      (candidate) => candidate.name === 'objectId',
    )!;
    expect(editor.referenceLabel(field, firstObject)).toBe('source-a / shared / shared');
    expect(editor.referenceLabel(field, secondObject)).toBe('source-b / shared / shared');
    editor.select(ENTITIES.find((entity) => entity.resource === 'columns')!);
    expect(editor.recordLabel({ columnId: 'c1', columnName: 'key', objectId: 'o1' })).toBe(
      'source-a / shared / shared / key',
    );
  });

  it('edits and deletes the chosen same-named interface by its stable id', async () => {
    const fixture = TestBed.createComponent(MetadataEditor);
    const first = {
      interfaceId: 'i1',
      interfaceName: 'shared',
      sourceName: 'source-a',
      authenticationId: null,
      interfaceFunctionalName: 'First',
      interfaceType: 'jdbc',
      interfaceLocation: 'jdbc:first',
      decimalSeparator: '.',
      thousandsSeparator: ',',
      delimiter: null,
      hasHeaders: true,
      isActive: true,
    };
    const second = { ...first, interfaceId: 'i2', sourceName: 'source-b', interfaceFunctionalName: 'Second' };
    load({ interfaces: [first, second] });
    const editor = fixture.componentInstance;
    editor.select(ENTITIES.find((entity) => entity.resource === 'interfaces')!);
    editor.edit(second);
    editor.values['interfaceFunctionalName'] = 'Updated';
    editor.save();

    const update = http.expectOne('/api/metadata/interfaces?id=i2');
    expect(update.request.method).toBe('PUT');
    expect(update.request.body.interfaceId).toBe('i2');
    update.flush({ ...second, interfaceFunctionalName: 'Updated' });
    load({ interfaces: [first, { ...second, interfaceFunctionalName: 'Updated' }] });
    await fixture.whenStable();

    editor.requestDelete({ ...second, interfaceFunctionalName: 'Updated' });
    editor.confirmDelete();
    const deletion = http.expectOne('/api/metadata/interfaces?id=i2');
    expect(deletion.request.method).toBe('DELETE');
    deletion.flush(null);
    load({ interfaces: [first] });
    await fixture.whenStable();
    expect(editor.records()['interfaces']).toEqual([first]);
  });

  it('edits without changing the key and encodes identifiers in requests', async () => {
    const fixture = TestBed.createComponent(MetadataEditor);
    const record = { ...source, sourceName: 'crm/table & finance' };
    load({ 'source-systems': [record] });
    fixture.detectChanges();
    const editButton = fixture.nativeElement.querySelector('tbody button');
    expect(editButton?.getAttribute('aria-label')).toBe('Edit crm/table & finance');
    editButton.click();
    fixture.detectChanges();
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('#sourceName').readOnly).toBe(true);
    fixture.componentInstance.values['sourceFunctionalName'] = 'Updated';
    fixture.nativeElement
      .querySelector('form')
      .dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    const request = http.expectOne('/api/metadata/source-systems?id=crm%2Ftable%20%26%20finance');
    expect(request.request.method).toBe('PUT');
    expect(request.request.body.sourceFunctionalName).toBe('Updated');
    request.flush({ ...record, sourceFunctionalName: 'Updated' });
    load({ 'source-systems': [{ ...record, sourceFunctionalName: 'Updated' }] });
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('tbody').textContent).toContain('Updated');
  });

  it('validates blank and overlength fields and retains a failed edit for retry', async () => {
    const fixture = TestBed.createComponent(MetadataEditor);
    load();
    fixture.componentInstance.edit();
    fixture.componentInstance.values = { ...source, sourceName: ' ' };
    expect(fixture.componentInstance.valid()).toBe(false);
    fixture.componentInstance.values = { ...source, sourceName: 'x'.repeat(256) };
    expect(fixture.componentInstance.valid()).toBe(false);
    fixture.componentInstance.values = { ...source };
    fixture.componentInstance.save();
    fixture.componentInstance.save();
    const request = http.expectOne('/api/metadata/source-systems');
    request.flush({}, { status: 409, statusText: 'Conflict' });
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('[role="alert"]').textContent).toContain('Conflict');
    expect(fixture.componentInstance.formOpen()).toBe(true);
    expect(fixture.componentInstance.values).toEqual(source);
    fixture.componentInstance.save();
    http.expectOne('/api/metadata/source-systems').flush(source);
    load();
    await fixture.whenStable();
    expect(fixture.componentInstance.formOpen()).toBe(false);
  });

  it('requires delete confirmation, reports referenced-record conflicts, and deletes on retry', async () => {
    const fixture = TestBed.createComponent(MetadataEditor);
    load({ 'source-systems': [source] });
    fixture.detectChanges();
    fixture.nativeElement.querySelector('[aria-label="Delete crm"]').click();
    http.expectNone('/api/metadata/source-systems?id=crm');
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Delete "crm"?');
    fixture.componentInstance.confirmDelete();
    const request = http.expectOne('/api/metadata/source-systems?id=crm');
    expect(request.request.method).toBe('DELETE');
    request.flush({}, { status: 409, statusText: 'Conflict' });
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.error').textContent).toContain('still in use');
    fixture.componentInstance.confirmDelete();
    http.expectOne('/api/metadata/source-systems?id=crm').flush(null);
    load();
    await fixture.whenStable();
    expect(fixture.nativeElement.textContent).toContain('Record deleted.');
    expect(fixture.nativeElement.textContent).toContain('No records yet');
  });

  it('cancels edits and deletion without writing data', () => {
    const fixture = TestBed.createComponent(MetadataEditor);
    load();
    fixture.componentInstance.edit(source);
    fixture.componentInstance.cancel();
    expect(fixture.componentInstance.formOpen()).toBe(false);
    fixture.componentInstance.requestDelete(source);
    fixture.componentInstance.cancel();
    expect(fixture.componentInstance.pendingDelete()).toBeNull();
    http.expectNone((request) => request.method !== 'GET');
  });

  it('shows loading errors and can refresh successfully', async () => {
    const fixture = TestBed.createComponent(MetadataEditor);
    const requests = ENTITIES.map((entity) => http.expectOne(`/api/metadata/${entity.resource}`));
    requests[0].flush('Unavailable', { status: 503, statusText: 'Service Unavailable' });
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('[role="alert"]').textContent).toContain(
      'Cannot reach',
    );
    expect(fixture.componentInstance.loaded()).toBe(false);
    fixture.componentInstance.select(ENTITIES[1]);
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('[role="alert"]').textContent).toContain(
      'Cannot reach',
    );
    fixture.componentInstance.refresh();
    load();
    await fixture.whenStable();
    expect(fixture.componentInstance.loaded()).toBe(true);
    expect(fixture.nativeElement.querySelector('[role="alert"]')).toBeNull();
  });

  it('keeps successful-save confirmation when reloading fails, then recovers without another write', () => {
    const fixture = TestBed.createComponent(MetadataEditor);
    load();
    fixture.componentInstance.edit();
    fixture.componentInstance.values = { ...source };
    fixture.componentInstance.save();
    http.expectOne('/api/metadata/source-systems').flush(source);
    const requests = ENTITIES.map((entity) => http.expectOne(`/api/metadata/${entity.resource}`));
    requests[0].flush('Unavailable', { status: 503, statusText: 'Service Unavailable' });
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Record saved.');
    expect(fixture.nativeElement.querySelector('[role="alert"]').textContent).toContain(
      'Cannot reach',
    );
    expect(fixture.componentInstance.formOpen()).toBe(false);
    fixture.componentInstance.refresh();
    load({ 'source-systems': [source] });
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('tbody').textContent).toContain('crm');
  });

  it.each([400, 404, 500])('shows explicit errors for failed saves (%s)', (status) => {
    const fixture = TestBed.createComponent(MetadataEditor);
    load();
    fixture.componentInstance.edit(source);
    fixture.componentInstance.save();
    http
      .expectOne('/api/metadata/source-systems?id=crm')
      .flush({}, { status, statusText: 'Error' });
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('[role="alert"]')).not.toBeNull();
    expect(fixture.componentInstance.values).toEqual(source);
  });
});
