<!--
Sync Impact Report
Version: 0.0.0 → 1.0.0
Modified principles: None → I. Code Quality & Simplicity; II. Testing Discipline (Non-Negotiable); III. User Experience Consistency; IV. Performance & Efficiency; V. Observability & Reliability
Added sections: Quality Gates & Non-Functional Standards; Development Workflow & Review Process
Removed sections: None
Templates requiring updates: ✅ .specify/templates/plan-template.md; ✅ .specify/templates/spec-template.md; ✅ .specify/templates/tasks-template.md; ⚠ .specify/templates/commands (not present in repo)
Follow-up TODOs: TODO(RATIFICATION_DATE): original adoption date not yet documented
-->
# GoKnut Constitution

## Core Principles

### I. Code Quality & Simplicity
Code MUST remain small, cohesive, and readable; unnecessary abstractions or dead code are removed promptly. Linting, formatting, and idiomatic naming are mandatory and block merges. Every change updates documentation and inline usage examples where behavior shifts.

### II. Testing Discipline (Non-Negotiable)
Automated tests MUST accompany every behavior change: unit coverage for new logic, integration/contract tests for interfaces and data boundaries, and regression tests for fixed defects. Tests are written before or alongside code and MUST fail first. No change may merge with failing tests or missing critical coverage justified in the plan.

### III. User Experience Consistency
UI and UX changes MUST use the shared design system, consistent interaction patterns, and vetted copy. Accessibility is mandatory: WCAG 2.1 AA expectations for keyboard, screen reader, and contrast are validated. Every user-facing change documents and verifies loading, empty, and error states; acceptance evidence (screenshots or recordings) is attached when feasible.

### IV. Performance & Efficiency
Each feature declares and validates performance budgets. Default expectations: backend p95 latency ≤250ms (p99 ≤500ms) for primary flows, frontend critical render/interaction ≤2s on baseline hardware, and changes avoid +100MB sustained memory growth unless justified. Profiling or load checks are required for new hot paths or endpoints, and observed impacts are recorded in the plan/spec.

### V. Observability & Reliability
Behavior MUST be observable: structured logs at decision points, metrics for throughput/latency/error rates, and traces for cross-service calls when applicable. Error handling favors fail-fast with actionable messages. Reliability goals are tracked via SLOs/error budgets, and incidents feed back into tests and documentation.

## Quality Gates & Non-Functional Standards

Work must satisfy constitutional gates before implementation and at review: (1) quality: lint/format clean, small cohesive changes, docs updated; (2) testing: new/changed behavior has automated coverage with failing-first evidence; (3) UX: design system usage, accessibility validation, and state handling documented; (4) performance: stated budgets with validation or planned measurement; (5) observability: logs/metrics/traces defined for new paths and failure modes.

## Development Workflow & Review Process

Planning artifacts (plan/spec) MUST capture quality gates, budgets, and observability hooks for the feature. Implementation follows TDD/BDD where applicable, keeping changes small and independently releasable. Code review verifies every principle, requests evidence for UX and performance validation, and rejects unmeasured risk. Releases include a brief change summary with impacts, validation performed, and rollback signals.

## Governance

This constitution governs all development practices and supersedes conflicting guidance. Amendments require documented rationale, version bump per semantic rules, and explicit communication in the repo. Compliance is reviewed at each PR and during periodic audits; violations require remediation plans before release. Runtime guidance files and templates MUST stay synchronized with the current constitution.

**Version**: 1.0.0 | **Ratified**: TODO(RATIFICATION_DATE) | **Last Amended**: 2025-12-06
