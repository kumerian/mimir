import { HttpErrorResponse } from '@angular/common/http';
import { Component, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { forkJoin } from 'rxjs';
import { ENTITIES, Entity, Field, MetadataApi, MetadataRecord, Resource } from './metadata-api';

@Component({
  selector: 'app-metadata-editor',
  imports: [FormsModule, RouterLink],
  templateUrl: './metadata-editor.html',
  styleUrl: './metadata-editor.css',
})
export class MetadataEditor {
  private readonly api = inject(MetadataApi);
  private readonly destroyRef = inject(DestroyRef);
  readonly entities = ENTITIES;
  readonly selected = signal(ENTITIES[0]);
  readonly records = signal<Partial<Record<Resource, MetadataRecord[]>>>({});
  readonly busy = signal(false);
  readonly loaded = signal(false);
  readonly error = signal('');
  readonly message = signal('');
  readonly editingId = signal<string | null>(null);
  readonly formOpen = signal(false);
  readonly pendingDelete = signal<MetadataRecord | null>(null);
  values: MetadataRecord = {};

  constructor() {
    this.refresh();
  }

  refresh() {
    if (this.busy()) return;
    this.busy.set(true);
    this.loaded.set(false);
    this.error.set('');
    forkJoin({
      'source-systems': this.api.list('source-systems'),
      authentications: this.api.list('authentications'),
      'api-parameters': this.api.list('api-parameters'),
      interfaces: this.api.list('interfaces'),
      objects: this.api.list('objects'),
      columns: this.api.list('columns'),
    })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (records) => {
          this.records.set(records);
          this.loaded.set(true);
          this.busy.set(false);
        },
        error: (error: HttpErrorResponse) => {
          this.error.set(this.describe(error));
          this.busy.set(false);
        },
      });
  }

  select(entity: Entity) {
    this.selected.set(entity);
    this.cancel();
    if (this.loaded()) this.error.set('');
    this.message.set('');
  }

  edit(record?: MetadataRecord) {
    this.values = record ? { ...record } : {};
    const key = record?.[this.selected().key];
    this.editingId.set(typeof key === 'string' ? key : null);
    this.formOpen.set(true);
    this.pendingDelete.set(null);
    this.error.set('');
    this.message.set('');
  }

  cancel() {
    this.formOpen.set(false);
    this.editingId.set(null);
    this.pendingDelete.set(null);
    this.values = {};
  }

  options(field: Field) {
    return field.reference ? (this.records()[field.reference] ?? []) : [];
  }

  recordLabel(record: MetadataRecord) {
    const resource = this.selected().resource;
    if (resource === 'interfaces') return this.interfaceLabel(record);
    if (resource === 'objects') return this.objectLabel(record);
    if (resource === 'columns') {
      const objectId = record['objectId'];
      const columnName = record['columnName'];
      const objectRecord = this.findRecord('objects', 'objectId', objectId);
      return objectRecord
        ? `${this.objectLabel(objectRecord)} / ${columnName}`
        : `${String(objectId ?? '')} / ${String(columnName ?? '')}`;
    }
    if (resource === 'api-parameters') {
      const interfaceRecord = this.findRecord('interfaces', 'interfaceId', record['interfaceId']);
      const parameterId = record['parameterId'];
      const label = interfaceRecord
        ? this.interfaceLabel(interfaceRecord)
        : String(record['interfaceId'] ?? '');
      return `${typeof parameterId === 'string' ? parameterId : ''} / ${label}`;
    }
    const value = record[this.selected().key];
    return typeof value === 'string' ? value : '';
  }

  referenceLabel(field: Field, record: MetadataRecord) {
    if (field.reference === 'interfaces') return this.interfaceLabel(record);
    if (field.reference === 'objects') return this.objectLabel(record);
    const key = field.referenceKey ? record[field.referenceKey] : null;
    return typeof key === 'string' ? key : '';
  }

  valid() {
    return this.selected().fields.every((field) => {
      const value = this.values[field.name];
      if (field.optional && (value == null || value === '')) return true;
      if (field.kind === 'boolean') return typeof value === 'boolean';
      if (field.kind === 'number') return typeof value === 'number';
      return (
        typeof value === 'string' &&
        value.trim().length > 0 &&
        value.length <= (field.maxLength ?? 255)
      );
    });
  }

  save() {
    if (this.busy() || !this.valid()) return;
    const value: MetadataRecord = {};
    for (const field of this.selected().fields) {
      value[field.name] =
        field.optional && (this.values[field.name] == null || this.values[field.name] === '')
          ? null
          : this.values[field.name];
    }
    this.busy.set(true);
    this.error.set('');
    const id = this.editingId();
    const request =
      id === null
        ? this.api.create(this.selected().resource, value)
        : this.api.update(this.selected().resource, id, { ...value, [this.selected().key]: id });
    request.pipe(takeUntilDestroyed(this.destroyRef)).subscribe({
      next: () => this.completed('Record saved.'),
      error: (error: HttpErrorResponse) => this.failed(error),
    });
  }

  requestDelete(record: MetadataRecord) {
    this.cancel();
    this.error.set('');
    this.message.set('');
    this.pendingDelete.set(record);
  }

  confirmDelete() {
    const record = this.pendingDelete();
    const id = record?.[this.selected().key];
    if (this.busy() || typeof id !== 'string') return;
    this.busy.set(true);
    this.error.set('');
    this.api
      .delete(this.selected().resource, id)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: () => this.completed('Record deleted.'),
        error: (error: HttpErrorResponse) => this.failed(error),
      });
  }

  private completed(message: string) {
    this.cancel();
    this.message.set(message);
    this.busy.set(false);
    this.refresh();
  }

  private failed(error: HttpErrorResponse) {
    this.error.set(this.describe(error));
    this.busy.set(false);
  }

  private describe(error: HttpErrorResponse) {
    if (error.status === 409)
      return 'Conflict: the name already exists under this parent, a referenced record is missing, or the record is still in use. Refresh and check its relationships.';
    if (error.status === 400)
      return 'Invalid data. Check required fields, field lengths, and the unchanged primary key.';
    if (error.status === 404) return 'This record no longer exists. Cancel and refresh.';
    return 'Cannot reach the backend or complete this operation. Your changes have not been confirmed; please retry.';
  }

  private interfaceLabel(record: MetadataRecord) {
    const sourceName = record['sourceName'];
    const interfaceName = record['interfaceName'];
    return `${typeof sourceName === 'string' ? sourceName : ''} / ${typeof interfaceName === 'string' ? interfaceName : ''}`;
  }

  private objectLabel(record: MetadataRecord) {
    const interfaceRecord = this.findRecord('interfaces', 'interfaceId', record['interfaceId']);
    const objectName = record['objectName'];
    const label = interfaceRecord ? this.interfaceLabel(interfaceRecord) : String(record['interfaceId'] ?? '');
    return `${label} / ${typeof objectName === 'string' ? objectName : ''}`;
  }

  private findRecord(resource: Resource, key: string, value: MetadataRecord[string]) {
    return (this.records()[resource] ?? []).find((record) => record[key] === value) ?? null;
  }
}
