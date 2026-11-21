# BL-023 GUI candidate evidence

## Status

Local synthetic candidate verification is implemented. Deployed acceptance,
production promotion, signing, and rollback drills remain blocked because no
acceptance/production target or approval is configured.

## Hermetic evidence

| Check | Status | Evidence |
| --- | --- | --- |
| Angular unit tests | Passed | `npm --prefix frontend run test:unit` |
| Angular production build | Passed | `npm --prefix frontend run build` |
| Chromium E2E | Passed | `npm --prefix frontend run test:e2e` |
| Synthetic tagged acceptance | Passed | `make acceptance` |
| Candidate identity unit test | Passed | `python3 scripts/test_gui_candidate_identity.py` |
| Documentation and contracts | Passed | `make docs` |

The browser suite synthetic-mocks the local `/api/v1` contract and verifies
profile identity, metadata/discovery, bounded run, schedule request stopping,
replay reconciliation, and cleaning preview/apply confirmation. It does not
provision a deployed environment or claim deployed acceptance.

## Candidate identity

Use the exact frontend and backend artifact pair:

```sh
make build
npm --prefix frontend ci
npm --prefix frontend run test:unit
npm --prefix frontend run build
python3 scripts/gui_candidate_identity.py \
  --binary bin/mimir \
  --frontend-dist frontend/dist/mimir/browser \
  --output dist/gui-candidate-identity.json
```

The identity file binds the Git source revision, binary SHA-256, frontend
directory file manifest/SHA-256, and health-reported contract versions.
Candidate production eligibility requires a clean worktree.

## Explicitly blocked

- A configured acceptance environment and human approval.
- Exact-artifact promotion to acceptance.
- Acceptance accessibility/security sign-off by an independent operator.
- Acceptance recovery and rollback rehearsal.
- Signing/provenance and production promotion.

No production smoke or promotion is claimed by this evidence.
