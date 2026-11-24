import { HttpErrorResponse } from '@angular/common/http';
import { Component, computed, DestroyRef, inject, signal } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { forkJoin } from 'rxjs';
import {
  MetadataApi,
  SourceColumn,
  SourceObject,
  SourceSystemInterface,
} from '../metadata/metadata-api';

@Component({
  selector: 'app-interface-details',
  imports: [RouterLink],
  templateUrl: './interface-details.html',
  styleUrl: './interface-details.css',
})
export class InterfaceDetails {
  private readonly api = inject(MetadataApi);
  private readonly destroyRef = inject(DestroyRef);
  private readonly interfaceId = inject(ActivatedRoute).snapshot.paramMap.get('id') ?? '';
  readonly sourceInterface = signal<SourceSystemInterface | null>(null);
  readonly objects = signal<SourceObject[]>([]);
  readonly columns = signal<SourceColumn[]>([]);
  readonly loading = signal(true);
  readonly operation = signal<'metadata' | 'load' | null>(null);
  readonly error = signal('');
  readonly message = signal('');
  readonly supportsCsv = computed(() => {
    const type = this.sourceInterface()?.interfaceType.replace(/[\s_-]/g, '').toLowerCase();
    return type === 'csv' || type === 'flatfile';
  });

  constructor() {
    this.refresh();
  }

  refresh() {
    if (!this.interfaceId) {
      this.error.set('A saved Interface is required.');
      this.loading.set(false);
      return;
    }
    this.loading.set(true);
    this.error.set('');
    forkJoin({
      sourceInterface: this.api.get('interfaces', this.interfaceId),
      objects: this.api.list('objects'),
      columns: this.api.list('columns'),
    })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: ({ sourceInterface, objects, columns }) => {
          this.sourceInterface.set(sourceInterface);
          const interfaceObjects = objects.filter((object) => object.interfaceId === this.interfaceId);
          this.objects.set(interfaceObjects);
          const objectIds = new Set(interfaceObjects.map((object) => object.objectId));
          this.columns.set(columns.filter((column) => objectIds.has(column.objectId)));
          this.loading.set(false);
        },
        error: (error: HttpErrorResponse) => {
          this.error.set(this.describe(error));
          this.loading.set(false);
        },
      });
  }

  deriveMetadata() {
    if (this.operation() || !this.supportsCsv()) return;
    this.operation.set('metadata');
    this.error.set('');
    this.message.set('');
    this.api
      .deriveCsvMetadata(this.interfaceId)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          const columnLabel = result.columns.length === 1 ? 'Column' : 'Columns';
          this.message.set(
            `Metadata saved: ${result.columns.length} ${columnLabel} from ${result.rowsScanned} data rows.` +
              (result.removedColumns ? ` Removed ${result.removedColumns} obsolete Columns.` : ''),
          );
          this.operation.set(null);
          this.refresh();
        },
        error: (error: HttpErrorResponse) => this.failed(error),
      });
  }

  loadData() {
    if (this.operation() || !this.supportsCsv()) return;
    this.operation.set('load');
    this.error.set('');
    this.message.set('');
    this.api
      .loadCsv(this.interfaceId)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          this.message.set(
            `Loaded ${result.rowsLoaded} data rows into ${result.objectName} (${result.tableName}).`,
          );
          this.operation.set(null);
          this.refresh();
        },
        error: (error: HttpErrorResponse) => this.failed(error),
      });
  }

  columnsFor(objectId: string) {
    return this.columns()
      .filter((column) => column.objectId === objectId)
      .sort((left, right) => left.columnOrdinalPosition - right.columnOrdinalPosition);
  }

  private failed(error: HttpErrorResponse) {
    this.error.set(this.describe(error));
    this.operation.set(null);
  }

  private describe(error: HttpErrorResponse) {
    const detail = error.error?.detail;
    return typeof detail === 'string'
      ? detail
      : 'The backend could not complete this Interface operation. No success was confirmed.';
  }
}
