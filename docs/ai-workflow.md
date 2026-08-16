# AI Workflow Optimization Guide

This document helps AI coding tools such as Codex, Cursor, Claude Code, Copilot, Continue, Windsurf, and Trae work more intelligently, quickly, accurately, and token-efficiently within TradeMind. The core goal is: **read less irrelevant content, form a minimal context package first, then implement, verify, and capture experience in small steps**.

## Scope

- Day-to-day vibe coding, bug fixes, feature development, refactoring, documentation improvements, and PR preparation.
- Applies to all AI agents and AI editors; tool-specific rules live only in their own configuration, while long-term general rules live in `AGENTS.md` and this document.
- Does not replace [ai-coding-rules.md](ai-coding-rules.md), [module-map.md](module-map.md), or [task-checklist.md](task-checklist.md), but explains how to use them more context-efficiently.

## Working Principles

1. **Identify the task type before reading files**: do not read the entire repository up front.
2. **Read entry points first, then details**: `AGENTS.md`, this document, `docs/module-map.md`, related module documentation, related code.
3. **Facts before solutions**: use search and existing code to confirm scripts, routes, fields, configuration, queue names, and Provider interfaces.
4. **Small changes, small verifications**: prioritize completing one verifiable closed loop before expanding to related content.
5. **Preserve human decision-making authority**: proceed carefully and confirm when product scope, external platforms, secrets, production data, destructive operations, or costly AI calls are involved.
6. **Write experience back to the repository**: recurring issues, architectural decisions, and tooling conventions must be captured in the appropriate document, not left only in a single conversation.

## Minimal Context Package

When starting a task, an AI should first assemble a context package of no more than 10 items:

| Item | Description |
| --- | --- |
| Task goal | What the user actually wants, not the tool's first impression |
| Current branch and changes | `git status --short --branch`, to avoid overwriting the user's changes |
| Change type | Backend, Admin, Collector, Provider, API, configuration, Docker, CI, documentation |
| Related entry points | Files that must be checked, found via `docs/module-map.md` |
| Existing implementation | Use `rg` to find the handler/service/provider/page/type/test, rather than guessing |
| Constraints | MVP scope, Provider abstraction, security, queues, need for human confirmation, etc. |
| Plan | 2-5 actionable steps |
| Verification method | The minimal check from `docs/task-checklist.md` |
| Risks | Points that are unconfirmed, unverified, or need human judgment |
| Experience capture | Whether this task needs updates to rules, pitfalls, PROGRESS, or module documentation |

The context package should only retain "information that will affect the next decision." Logs, candidate files, failed hypotheses, and temporary output that have been read but are no longer needed should not carry forward into subsequent turns.

## Context Engineering Budget

Manage context for every task in the order "budget → retrieve → compress → execute" to avoid the task getting heavier as it progresses.

| Stage | Token target | Allowed into context | Should not enter context |
| --- | --- | --- | --- |
| Task kickoff | ≤ 1k | User goal, branch state, change type, constraints | Full README, full directory tree |
| Locating files | ≤ 2k | `rg` results, relevant file paths, key symbols | Irrelevant search results, large repeated code blocks |
| Reading implementation | ≤ 6k | Relevant functions, types, config snippets, error summaries | Entire modules, full build logs |
| Modify and verify | ≤ 4k | Diff summary, key lines from failed tests, verification commands | Full text of successful logs, unrelated warnings |
| Deliver and capture | ≤ 2k | Change summary, verification results, remaining risks, reusable experience | A blow-by-blow account of the process |

Execution rules:

- Use `rg --files`, `rg -n`, and `git diff --stat` to find entry points first, then read locally.
- Keep each fact only once; if it's already been written into a plan or document, don't re-expand it repeatedly in conversation.
- Read long files by heading, function name, or line number; only read the full file when the structure is unknown.
- For test and build output, record only the first root cause, relevant file/line numbers, the command, and the final status.
- When the context starts growing large, first produce a 5-8 item "current fact summary" before continuing to the next step.

## Task Routing

| Task type | Read first | Common sync targets |
| --- | --- | --- |
| Backend API / DTO | `docs/api.md`, the corresponding handler/service/model, frontend services/types | `docs/api.md`, Admin pages, tests |
| Provider | `docs/provider.md`, `docs/provider-template.md`, `backend/internal/providers` | Settings page, connection test, masked display, Provider documentation |
| Admin page | `admin/config/routes.ts`, page, services, types, UI rules | README capability description, related docs |
| Collector | `collector/`, `docs/collector-1688-pitfalls.md`, collection API | Backend DTO, draft mapping, `docs/api.md` |
| Environment variables | `.env.example`, `docs/env.md`, config code | Docker, development, and deployment documentation |
| Docker / CI | Workflow, compose, Dockerfile, `docs/docker-deployment.md` | README, CONTRIBUTING, PR template |
| Documentation / rules | `docs/README.md`, `AGENTS.md`, `.cursor/rules/README.md` | README / README.en navigation, related rules |

## Token-Saving Strategies

- Prefer `rg --files`, `rg -n`, `git diff --stat`, and partial `Get-Content` / `sed` reads; don't paste large files whole.
- Read directories and symbols first, then implementation; read recently relevant files first, then expand scope.
- For large documents, read only the relevant section; conclusions needed long-term should be written into a plan or document instead of re-read repeatedly.
- Don't dump build logs, test logs, and API responses verbatim into context — keep only the error summary, file/line numbers, and key commands.
- Keep rule files short and strong, with detailed explanations in `docs/`; Cursor `.mdc` files should only keep high-frequency constraints and links.
- Maintain only one current plan per task; update it step by step as you go, avoiding long repeated restatements.

## Automatic Prompt Optimization

An AI agent should first compress the user's raw request into an "execution prompt" before starting to search and modify. The execution prompt is not a long explanation written for the user, but a short instruction used for the current task.

### Standard Rewrite Template

```text
Goal: describe the acceptable result in one sentence.
Task type: backend / Admin / Collector / Provider / API / configuration / Docker / CI / documentation.
Scope boundaries: what must be done; what is explicitly out of scope.
Required reading: AGENTS.md, docs/ai-workflow.md, docs/module-map.md, task-related files.
Fact confirmation: fields, routes, commands, or configuration that need to be confirmed via rg / git status / partial reads.
Implementation strategy: 2-5 steps, prioritizing reuse of existing layering, Providers, components, and documentation structure.
Verification: choose the minimal check command per docs/task-checklist.md.
Capture: determine whether docs / pitfalls / rules / PROGRESS need to be updated.
```

### Rewrite Rules

- For vague requests, first fill in the acceptance result: e.g. "optimize the AI workflow" should become "update documentation and rules so the AI can automatically do prompt rewriting, context budgeting, and writing back experience."
- Break large requests down into an MVP closed loop first: prioritize completing the smallest verifiable result, and don't default to expanding into a full ERP.
- If the user specifies a path, command, or platform, verify it exists first; if it doesn't, note the assumption and search for a nearby implementation.
- Generate confirmation points for high-risk actions: production data, secrets, paid AI calls, external platform write operations, destructive commands.
- Keep the final prompt short; don't copy long documents — reference long rules via links.

### Requirement Clarification Priority

Only ask follow-up questions when a safe inference cannot be made. First try to confirm the following from the repository itself:

1. Routes, scripts, environment variables, Provider names, task types.
2. Existing pages, services, types, tests, and documentation entry points.
3. Whether the same feature, same bug, or same rule already exists.

Cases where a follow-up question is required:

- The goal would change the product boundary or introduce heavyweight capability.
- Mutually exclusive implementation paths exist for the same requirement, and both would affect user data.
- Real secrets, accounts, production data, or external platform write access are needed.

## Standard Execution Flow

1. **Align on the goal**: confirm whether the task belongs to one of the current two main lines of work, avoiding a default expansion into a full ERP.
2. **Scan context**: check the branch, uncommitted changes, module mapping, and related files.
3. **Form a plan**: list the scope of impact, files to edit, and verification commands.
4. **Implement the change**: keep it small and focused, prioritizing reuse of existing layering and Provider abstractions.
5. **Sync documentation**: update related documentation and entry points per `docs/module-map.md`.
6. **Run verification**: perform the minimum necessary checks per `docs/task-checklist.md`.
7. **Capture experience**: write reusable conclusions to the correct location.
8. **Final summary**: explain what was changed, what was verified, why anything was left unverified, and remaining risks.

## Quality Gates

To reduce rework, an AI agent performs a lightweight self-check before editing, before verifying, and before delivery.

| Timing | Self-check question |
| --- | --- |
| Before editing | Is this change within the user's goal? Has `docs/module-map.md` been checked? Would it overwrite the user's existing changes? |
| Before verifying | Have API / types / configuration / documentation entry points been synced? Is formatting or a build needed? |
| Before delivery | Are the verification results stated? Is there a reason for anything unverified? Is there reusable experience that needs to be written back? |

If a check fails, fix it before continuing; if it can't be fixed due to environment limitations, state that explicitly in the final summary.

## Self-Improvement Mechanism

AI should not interpret "growth" as secretly saving private memory locally; TradeMind's growth should happen through auditable repository documentation. Self-improvement follows the closed loop "observe → generalize → write back → reuse next time."

| Trigger scenario | Write-back location |
| --- | --- |
| A category of bug occurs a second time | The corresponding pitfalls document or module documentation |
| A new cross-tool long-term rule | `AGENTS.md`, `docs/ai-coding-rules.md`, synced to `.cursor/rules/` if needed |
| Cursor-specific execution constraints | `.cursor/rules/*.mdc` and `.cursor/rules/README.md` |
| Phase facts, completed capabilities, known issues | `docs/PROGRESS.md` |
| API / Provider / queue / configuration contract changes | `docs/api.md`, `docs/provider.md`, `docs/env.md` |
| Prompt, AI call chain, or quality gate changes | Prompt templates, AI provider documentation, related task documentation |
| PR process, check commands, or branching strategy changes | `docs/branching.md`, `CONTRIBUTING.md`, PR template |

### Criteria for Writing Back Experience

Write back rather than only explaining in chat when any of the following apply:

- The same type of issue has occurred a second time.
- A reusable step was formed this time to save tokens or improve accuracy.
- A module has a new quality gate, regression command, or prohibited practice.
- A prompt template, Provider contract, AI task input/output, or token-logging approach has changed.
- A conflict has appeared between documents or rules that needs a priority order.

Writing back experience must satisfy:

- Do not record real secrets, cookies, tokens, customer data, production data, or private conversations.
- Do not elevate a one-off personal preference into a global rule.
- New rules must be short, actionable, and reduce future misjudgment or token consumption.
- If a rule only applies to a specific directory or tech stack, prefer writing it as a scoped rule rather than polluting all tasks.

### Self-Improvement Output Format

When capturing experience, prefer the following format for easier retrieval by future AI runs:

```text
Trigger: the situation in which this experience applies.
Rule: one actionable constraint.
Verification: what command, file, or page confirms it.
Location: the directory, module, or task type the rule applies to.
```

## Multi-Tool Collaboration Conventions

- **Codex / Claude Code / other agents**: read `AGENTS.md`, this document, and task-related documentation first, then execute the change.
- **Cursor**: relies mainly on `.cursor/rules/*.mdc`, jumping to `docs/` when detail is needed; don't copy this entire document into every rule.
- **Copilot / Continue / Windsurf / Trae**: treat `AGENTS.md` and this document as the project-description entry point, reading the minimal relevant documentation per task type.
- **Human developers**: can use the "minimal context package" as a template for issues, PRs, or handoff notes.

## Common Prompt Template

When starting a task with any AI tool, it's recommended to include:

```text
Goal:
Scope of impact:
Must follow:
- Read AGENTS.md, docs/ai-workflow.md, docs/module-map.md first
- Stay within MVP scope; do not introduce heavyweight ERP capability
- Rewrite the request into a short execution prompt first, then read files per the minimal context package
- Verify per docs/task-checklist.md after making changes
Expected output:
- Change summary
- Verification results
- Reasons for anything unverified, and remaining risks
```

## Admin Copy and UI Standards

For changes to the Admin side (pages, components, styles), in addition to `docs/module-map.md`, read `.agents/skills/frontend-design/SKILL.md` first; that skill is the single complete source for Admin UI design standards, shared-component conventions, layout rules, responsive acceptance criteria, and the AI implementation workflow. The table below lists only commonly used supporting resources and does not replace the primary standard.

| Resource | Purpose |
| --- | --- |
| `.agents/skills/frontend-design/SKILL.md` | Primary Admin UI standard: five-tier viewports, `browser_route`, root-node horizontal overflow, readonly, rowKey, Header / Content baseline, API / payload protection |
| `docs/ui-copywriting.md` | User-facing copy glossary, prohibited items, `pnpm check:ui-copy` |
| `admin/src/constants/copywriting.ts` | Page titles, descriptions, unified terminology for products/platforms/tasks/inventory |
| `admin/src/constants/layoutTokens.ts` | Page padding, card spacing, form grid spacing |
| `admin/src/constants/errorMessages.ts` | Error code → user-facing message (including suggested actions) |
| `admin/src/constants/status.ts` | Status copy and Tag colors |
| `admin/src/constants/userFriendly.ts` | Generic labels (spec, storage, runtime, access method, etc.) |
| `admin/src/components/ui/` | PageContainer, SectionCard, FormGrid, EmptyState, TechnicalDetails, TaskJsonBlock, etc. |

### Copy Principles (summary)

1. **User-facing, not developer-facing**: the main UI should not surface raw terms like Provider, Worker, runtime, Storage, Stale, Endpoint; see the full glossary in **`docs/ui-copywriting.md`**.
2. **Help text** should only answer: what it's for, what to enter, and what happens if entered incorrectly.
3. **Technical information** (error codes, request IDs, raw JSON) belongs in a collapsed "Technical Details" section, collapsed by default; JSON in the task detail drawer uses `TechnicalDetails` + `TaskJsonBlock`.
4. **Empty states** must include: a title, the reason, and a suggested action (an optional button).
5. **Buttons** use "verb + object" (e.g. "Save Settings", "Test Connection"), avoiding bare "OK" or "Submit".
6. **After changing user-facing copy**, run `pnpm check:ui-copy --strict` (same command as CI); sync new high-frequency terms to `userFriendly.ts` and `docs/ui-copywriting.md`.

### Layout Principles (summary)

Layout, horizontal baseline, page container, five-tier viewports, Modal / Drawer / Popconfirm, and write-request side-effect acceptance criteria are governed by `.agents/skills/frontend-design/SKILL.md`. When modifying an Admin page: first check whether an existing `PAGE_COPY` / shared component can be reused, to avoid hand-writing styles and duplicating terminology on every page; prefer `TmPageContainer` for the page container, and dashboard-type pages may use `layoutTokens.dashboardMaxWidth` per the primary standard.

### Established Patterns

| Scenario | Approach |
| --- | --- |
| Task detail drawer | Business fields in `Descriptions`; `input` / `output` / raw JSON in `TechnicalDetails` + `TaskJsonBlock` |
| Publishing / inventory / listing descriptions | User-readable description shown directly; API parameter names and preset key names collapsed in `TechnicalDetails` |
| Store authorization form | OAuth main flow exposed; secret overrides, tokens, seller IDs, etc. collapsed into `TechnicalDetails` |
| Collection rules / Prompt JSON | The entire editing area is wrapped in `TechnicalDetails`, with a hint that it "usually doesn't need to be changed" |
| Status tags | Prefer `constants/status.ts` or `commonStatusLabel()`; avoid rendering raw English enum values directly |
| Platform / task type | `platformLabel()`, `aiTaskTypeLabel()`, `taskTypeLabel()`, etc. |
| Error display | Main UI uses `formatUserErrorMessage()`; the raw `errorCode` appears only in the technical details section |

### Common Helper Functions (`copywriting.ts` / `status.ts`)

- `commonStatusLabel` / `readinessLevelLabel` / `publishModeLabel`
- `collectTaskEventLabel` / `collectTaskStatusTransition`
- `AI_FIELD_COPY` (e.g. AI-optimized title / description)

When adding a new page or drawer, check the table above and the implementation of the `PublishTasks` and `DraftDetail` publishing tabs before writing the UI.

## Completion Criteria

Before an AI collaboration task is considered complete, confirm at least:

- The code, configuration, and documentation relevant to the current task have been read.
- The user's existing changes have not been overwritten.
- Related content has been checked per `docs/module-map.md`.
- Verification has been performed or explained per `docs/task-checklist.md`.
- Any new long-term experience has been written to the appropriate document, not left only in chat.

## Production Maintenance Phase

The historical F1-F9 planning, freeze audits, and enhancement plans have been removed from the current working tree; query Git history when needed. Current tasks should be carried out via the following entry points:

- Confirm scope from the [Documentation Center](README.md), [Module Reference Index](module-map.md), and [Current Maintenance Status](PROGRESS.md).
- Automated regression is run by GitHub Actions; product processes are signed off per the [P10 Manual Acceptance Checklist](PRODUCTION_MANUAL_ACCEPTANCE_CHECKLIST.md).
- When demo data is needed, generate it temporarily per the [Demo Data Seeding Guide](DEMO_SEEDING_GUIDE.md); do not commit run output.
- Maintenance prioritizes stability, security issues, and necessary feature fixes; it does not restore phase gates, one-off reports, or freeze evidence.
