import { expect, test } from '@playwright/test';

test.beforeEach(async ({ page }) => {
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    let body: unknown = {};

    if (path.endsWith('/health')) {
      body = {
        status: 'ready',
        version: 'e2e',
        commit: 'synthetic',
        contracts: ['mimir.gui-api/v1'],
        profile: 'local-reference-v1',
      };
    } else if (path.endsWith('/metadata/validate')) {
      body = {
        contractVersion: 'mimir.metadata-validation/v1',
        inputDigest: 'a'.repeat(64),
        valid: true,
        runEligible: false,
        securityImpact: 'input_not_retained',
        compatibilityDecision: 'accepted_v1',
        diagnostics: [],
      };
    } else if (path.endsWith('/metadata/revisions') && request.method() === 'GET') {
      body = [
        { revision: 'active-v1', status: 'active', digest: 'b'.repeat(64) },
        { revision: 'synthetic-orders-v1', status: 'draft', digest: 'c'.repeat(64) },
      ];
    } else if (path.endsWith('/metadata/revisions')) {
      body = { revision: 'synthetic-orders-v1', status: 'draft', digest: 'c'.repeat(64) };
    } else if (path.endsWith('/metadata/activate')) {
      body = { revision: 'synthetic-orders-v1', status: 'active', digest: 'c'.repeat(64) };
    } else if (path.endsWith('/discovery')) {
      body = {
        proposalId: 'proposal-e2e',
        proposal: {
          activeRevision: 'synthetic-orders-v1',
          observation: {
            contractVersion: 'mimir.discovery/v1',
            provider: 'local-jsonl',
            providerVersion: 'v1',
            scope: 'local-reference',
            sourceDigest: 'd'.repeat(64),
            observedAt: '2026-01-01T00:00:00Z',
            columns: [{ id: 'order_id', type: 'integer', classification: 'internal' }],
          },
          report: {
            contractVersion: 'mimir.discovery/v1',
            compatible: true,
            disposition: 'continue',
            changes: [],
          },
          proposedColumns: [{ id: 'order_id', type: 'integer', classification: 'internal' }],
          executionReady: false,
        },
      };
    } else if (path.endsWith('/runs')) {
      body = {
        runId: 'e'.repeat(64),
        metadataRevision: 'synthetic-orders-v1',
        recordCount: 1,
        alreadyCommitted: false,
      };
    } else if (path.endsWith('/schedules/evaluate')) {
      body = {
        decisions: [{ sourceId: 'orders', objectId: 'orders', eligible: true, reason: 'slot_due' }],
        requests: [{
          contractVersion: 'mimir.schedule-request/v1',
          requestId: 'f'.repeat(64),
          metadataRevision: 'synthetic-orders-v1',
          sourceId: 'orders',
          objectId: 'orders',
          scheduledAt: '2026-01-01T00:00:00Z',
          retryLimit: 0,
          concurrencyLimit: 1,
        }],
      };
    } else if (path.endsWith('/schedules/stop')) {
      body = { status: 'pending_requests_stopped' };
    } else if (path.endsWith('/replays')) {
      body = {
        replayPlanId: 'replay-e2e',
        originalRunId: 'e'.repeat(64),
        result: {
          runId: 'e'.repeat(64),
          metadataRevision: 'synthetic-orders-v1',
          recordCount: 1,
          alreadyCommitted: true,
        },
      };
    } else if (path.endsWith('/cleaning/preview')) {
      body = {
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
      };
    } else if (path.endsWith('/cleaning/apply')) {
      body = {
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
      };
    } else {
      body = {
        contractVersion: 'mimir.failure/v1',
        category: 'unsupported_capability',
        operation: 'browser-test',
        retryable: false,
        commitCertainty: 'not_committed',
        safeDiagnostic: 'This synthetic route is unsupported.',
      };
    }

    await route.fulfill({
      status: path.endsWith('/health') ? 200 : 200,
      contentType: 'application/json',
      body: JSON.stringify(body),
    });
  });
});

test('metadata discovery and health identity journey', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByRole('heading', { name: 'Local API connection' })).toBeVisible();
  await expect(page.getByText('Identity only—not a check')).toBeVisible();

  await page.getByRole('button', { name: 'Validate metadata' }).click();
  await expect(page.getByText('Valid metadata')).toBeVisible();
  await expect(page.getByText('runEligible: false')).toBeVisible();

  await page.getByRole('button', { name: 'Author valid draft' }).click();
  await expect(page.getByText('Draft authored')).toBeVisible();
  await page.getByRole('button', { name: 'Refresh list' }).click();
  await expect(page.getByText('active-v1')).toBeVisible();

  await page.locator('#discovery-source').fill('/private/source');
  await page.getByRole('button', { name: 'Discover schema proposal' }).click();
  await expect(page.getByText('Advisory proposal · not activated')).toBeVisible();
  await expect(page.getByText('executionReady: false')).toBeVisible();
});

test('bounded run schedule replay and request stop journey', async ({ page }) => {
  await page.goto('/');
  await page.getByRole('button', { name: 'Runs & schedules' }).click();

  await page.locator('#run-source').fill('/private/source');
  await page.getByRole('button', { name: 'Run one bounded ingestion' }).click();
  await expect(page.getByText('Run result')).toBeVisible();
  await expect(page.getByText('Already committed')).toBeVisible();

  await page.getByRole('button', { name: 'Evaluate fixed-interval schedules' }).click();
  await expect(page.getByText('undelivered request(s)', { exact: false })).toBeVisible();

  await page.getByRole('button', { name: 'Stop pending requests' }).click();
  await expect(page.getByText('pending_requests_stopped')).toBeVisible();
  await expect(page.getByText('Run state was not changed')).toBeVisible();

  await page.locator('#replay-source').fill('/private/source');
  await page.getByLabel('Original run ID').fill('e'.repeat(64));
  await page.getByRole('button', { name: 'Reconcile original run' }).click();
  await expect(page.getByText('Original batch reconciled')).toBeVisible();
});

test('cleaning requires preview and explicit apply confirmation', async ({ page }) => {
  await page.goto('/');
  await page.getByRole('button', { name: 'Explicit cleaning' }).click();

  await page.getByLabel('Correction ID').fill('trim-name');
  await page.getByLabel('Field').fill('name');
  await page.getByLabel('One JSON-object record · max 256 KiB').fill('{"name":"  Example  "}');
  await page.getByRole('button', { name: 'Preview correction' }).click();
  await expect(page.getByText('Preview only · nothing persisted')).toBeVisible();

  const applyButton = page.getByRole('button', { name: 'Apply this correction' });
  await expect(applyButton).toBeDisabled();
  await page.getByLabel('I reviewed this preview').check();
  await expect(applyButton).toBeEnabled();
  await applyButton.click();
  await expect(page.getByText('Separate cleaned output applied')).toBeVisible();
  await expect(page.getByText('Sensitive local data', { exact: true })).toBeVisible();
});
