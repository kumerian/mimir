# Working process

This is an initial, lightweight working process for Mimir. It uses the repository's Markdown backlog and existing development and test workflows. It does not establish a sprint cadence or require a particular set of Scrum events.

## Flow of work

1. **Capture and clarify.** Record work in the appropriate backlog area. Make the intended outcome and acceptance criteria clear enough to check. Identify dependencies, risks, and relevant product behavior before starting.
2. **Check readiness.** The people doing and reviewing the work use the [Definition of Ready](definition-of-ready.md) as a checklist. If an important question prevents a sound implementation, clarify the work before treating it as ready.
3. **Implement and verify.** Make the smallest change that satisfies the acceptance criteria. Run the relevant tests and checks, and update directly related documentation. Keep incomplete work visible rather than describing it as finished.
4. **Review and integrate.** When using a pull request, make the change and its verification evidence understandable to reviewers, and address or explicitly resolve review feedback. The repository's CI runs on pull requests; the [Definition of Done](definition-of-done.md) describes the expected check results.
5. **Confirm completion.** Check every applicable Definition of Done criterion. If any criterion is unmet, use the [Definition of Undone](definition-of-undone.md) and continue or re-plan the remaining work.

The backlog currently separates process, loading, scheduling, user-interface, logging, and secrets work. Keep dependencies and unfinished work visible in that backlog; this document does not impose ordering or priorities on its items.

## Collaboration

The people contributing to a change share responsibility for clarifying its outcome, raising risks, verifying behavior, and keeping its status accurate. This document does not assign individual roles or require a specific approval count. The team may add those agreements if and when they are established.
