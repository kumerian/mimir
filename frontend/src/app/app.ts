import { Component, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { AuditLog } from './audit/audit-log';
import { MetadataEditor } from './metadata/metadata-editor';

@Component({
  selector: 'app-root',
  imports: [MetadataEditor, AuditLog],
  templateUrl: './app.html',
  styleUrl: './app.css',
})
export class App {
  protected readonly status = signal('Connecting to the backend...');
  protected readonly error = signal(false);

  constructor() {
    inject(HttpClient)
      .get<{ application: string; database: string }>('/api/status')
      .pipe(takeUntilDestroyed())
      .subscribe({
        next: (response) =>
          this.status.set(`${response.application}: database ${response.database}`),
        error: () => {
          this.error.set(true);
          this.status.set('Cannot reach the backend or database. Please try again later.');
        },
      });
  }
}
