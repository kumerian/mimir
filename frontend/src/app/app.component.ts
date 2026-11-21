import { HttpClient, HttpErrorResponse, HttpHeaders } from '@angular/common/http';
import { Component, OnInit, inject, signal } from '@angular/core';

interface ProfileHealth {
  status: string;
  version: string;
  commit: string;
  contracts: string[];
  profile: string;
}

@Component({
  selector: 'mimir-root',
  standalone: true,
  template: `
    <main class="shell">
      <header class="topbar">
        <a class="brand" href="/" aria-label="Mimir home">
          <img src="mimir_logo_32x32.png" width="32" height="32" alt="">
          <span>Mimir</span>
        </a>
        <span class="local-badge"><span aria-hidden="true"></span> Local workspace</span>
      </header>

      <section class="intro">
        <p class="eyebrow">DATA INGESTION WORKSPACE</p>
        <h1>Welcome to Mimir</h1>
        <p class="lede">A local-first workspace for discovering, validating, and processing data.</p>
      </section>

      <section class="status-card" aria-labelledby="connection-title">
        <div class="status-icon" [class.online]="health()?.status === 'ready'" aria-hidden="true">
          {{ health()?.status === 'ready' ? '✓' : '!' }}
        </div>
        <div class="status-copy">
          <h2 id="connection-title">Backend connection</h2>
          @if (health(); as profile) {
            <p class="connection-state"><span class="dot"></span> Connected</p>
            <dl>
              <div><dt>Profile</dt><dd>{{ profile.profile }}</dd></div>
              <div><dt>Version</dt><dd>{{ profile.version }}</dd></div>
              <div><dt>API contract</dt><dd>{{ contractVersion(profile.contracts) }}</dd></div>
            </dl>
            <p class="health-note">Profile health reports identity only; it does not check MariaDB or source readiness.</p>
          } @else if (error()) {
            <p class="connection-error">{{ error() }}</p>
            <button type="button" (click)="loadHealth()">Retry connection</button>
          } @else {
            <p class="muted">Checking the Go backend…</p>
          }
        </div>
      </section>

      <section class="capabilities" aria-labelledby="capabilities-title">
        <div>
          <p class="eyebrow">WORKSPACE</p>
          <h2 id="capabilities-title">Available through the local API</h2>
          <p class="muted">The interface is being built around the existing backend capabilities.</p>
        </div>
        <ul>
          <li><span>01</span> Metadata validation and revision management</li>
          <li><span>02</span> Source discovery and data runs</li>
          <li><span>03</span> Schedules, replay, and cleaning</li>
        </ul>
      </section>

      <footer>
        <span>Mimir local reference profile</span>
        <span>Angular UI · Go API · MariaDB metadata</span>
      </footer>
    </main>
  `,
})
export class AppComponent implements OnInit {
  private readonly http = inject(HttpClient);
  readonly health = signal<ProfileHealth | null>(null);
  readonly error = signal<string | null>(null);

  ngOnInit(): void {
    this.loadHealth();
  }

  loadHealth(): void {
    this.error.set(null);
    this.health.set(null);
    const headers = new HttpHeaders({ 'Mimir-Contract-Version': 'mimir.gui-api/v1' });
    this.http.get<ProfileHealth>('/api/v1/health', { headers }).subscribe({
      next: (profile) => this.health.set(profile),
      error: (error: HttpErrorResponse) => {
        this.error.set(error.status
          ? `The Go backend returned HTTP ${error.status}.`
          : 'The Go backend is unavailable. Check that the local services are running.');
      },
    });
  }

  contractVersion(contracts: string[]): string {
    return contracts.find((contract) => contract.startsWith('mimir.gui-api/')) ?? 'Unknown';
  }
}
