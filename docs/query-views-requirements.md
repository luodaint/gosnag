# Query Views — Requirements Document

## 1. Overview

Add a read-only query feature in GoSnag so users can run ad hoc queries against controlled PostgreSQL views and build lightweight reports.

The goal is to provide a New Relic-style query workflow for operational reporting, without exposing unrestricted SQL against application databases.

## 2. Goals

- let users query predefined PostgreSQL views from the GoSnag UI
- support operational reporting and quick investigation
- allow saving and re-running useful queries
- keep the experience safe, fast, and read-only

## 3. Non-Goals

- unrestricted SQL against production application databases
- access to base tables by default
- support for mutating SQL
- full BI/dashboarding in the first iteration

## 4. Product Direction

The feature should be built around a controlled analytical surface:

- PostgreSQL only in the first version
- queries run only against approved views
- read-only execution
- row limits and execution timeouts

This should feel like a lightweight query console for support, product, and engineering teams.

## 5. Core Requirements

**REQ-QV-001**: A project MUST be able to enable or disable query views.

**REQ-QV-002**: The system MUST support a project-level PostgreSQL connection dedicated to query execution.

**REQ-QV-003**: Queries MUST be limited to `SELECT` statements.

**REQ-QV-004**: Queries MUST run only against approved views, not arbitrary tables.

**REQ-QV-005**: The UI MUST provide a query editor, tabular results, and basic error feedback.

**REQ-QV-006**: Users SHOULD be able to save named queries for later reuse.

**REQ-QV-007**: The system SHOULD support exporting query results in a simple format such as CSV.

## 6. Safety Requirements

**REQ-QV-020**: The connection used for this feature MUST be read-only.

**REQ-QV-021**: The backend MUST enforce query timeouts and result limits.

**REQ-QV-022**: Credentials MUST be treated as sensitive configuration and MUST NOT be returned to the frontend in clear text.

**REQ-QV-023**: The first release SHOULD be limited to admin users.

## 7. Suggested First Scope

The first release should focus on:

- one PostgreSQL connection per project
- a whitelist of views
- a simple query editor
- saved queries
- CSV export

That is enough to validate the feature before expanding into richer reporting or dashboards.
