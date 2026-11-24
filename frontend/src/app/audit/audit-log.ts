import { HttpClient } from '@angular/common/http';
import { Component, inject, signal } from '@angular/core';

interface AuditEntry {
  id: number;
  occurredAt: string;
  operation: string;
  entityType: string;
  recordId: string;
  details: string;
}

@Component({
  selector: 'app-audit-log',
  templateUrl: './audit-log.html',
  styleUrl: './audit-log.css',
})
export class AuditLog {
  private readonly http = inject(HttpClient);
  readonly entries = signal<AuditEntry[]>([]);
  readonly busy = signal(false);
  readonly error = signal('');

  constructor() {
    this.refresh();
  }

  refresh() {
    if (this.busy()) return;
    this.busy.set(true);
    this.error.set('');
    this.http.get<AuditEntry[]>('/api/audit').subscribe({
      next: (entries) => {
        this.entries.set(entries);
        this.busy.set(false);
      },
      error: () => {
        this.error.set('Cannot load metadata audit history. Please try again later.');
        this.busy.set(false);
      },
    });
  }
}
