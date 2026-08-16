---
name: frontend-design
description: The single, complete source of truth for TradeMind Admin UI design standards, shared component conventions, layout rules, responsive acceptance criteria, and the AI implementation workflow
license: Complete terms in LICENSE.txt
---

# TradeMind Admin UI Design and AI Implementation Standard

This file is the single, complete source of truth for the TradeMind Admin UI design and implementation standard. Any AI tool (Cursor, Claude Code, Codex, Copilot, Continue, Windsurf, Trae, or any other agent) working on Admin frontend tasks must follow this standard; other entry points should only reference this file and must not duplicate a second complete UI standard. Besides UI and E2E acceptance, Admin UI implementation must also follow `.agents/skills/code-quality/SKILL.md`.

Large page decomposition, shared UI boundaries, cross-page module design, shared types/services, and the evolution of DraftDetail follow `.agents/skills/modular-architecture/SKILL.md`; this standard only defines UI visuals, layout, responsiveness, and interaction acceptance.

## 1. Automatic Application Rules

This standard applies not only when the user explicitly asks for "UI design," "UI optimization," "responsive acceptance," or "use the frontend-design skill." It automatically applies to any task involving Admin frontend pages, components, styles, layout, responsiveness, interactions, state display, copy, accessibility, or visual bugs — the task requester does not need to specify this separately.

Whenever any of the following semantics appear, the task must automatically be judged as UI-related and this standard must be read. Typical user descriptions include, but are not limited to: "fix this page display issue," "add a new Admin page," "the table overflows on mobile," "the button gets clicked twice," "tweak this button," "tweak this table," "optimize the page."

- Page visuals: the page doesn't look good, the page needs optimizing, style adjustments are needed, content isn't aligned, inconsistent spacing, unreasonable font size, inconsistent colors, unclear card hierarchy, the page is too cramped, the page is too sparse, the content area is too narrow, abnormal page width, unreasonable button hierarchy, unclear status display.
- Page layout: Header and Content misaligned, Breadcrumb misaligned with body content, padding/margin issues, grid layout issues, Flex/Grid issues, horizontal scroll on the page root, a table breaking out of the page, Tabs overflowing, Card width issues, layout glitches after the sidebar expands/collapses, misaligned fixed bars/toolbars/pagination.
- Responsiveness and mobile: mobile adaptation, tablet adaptation, small-screen display issues, a Modal overflowing the screen, abnormal Drawer width, buttons truncated on mobile, a table unusable on mobile, long text breaking the layout, misalignment after a viewport change.
- Components and interaction: adding or modifying Button, Form, Table/ProTable, Card, Tabs, Modal, Drawer, Popconfirm, Tooltip, Select, Upload, Pagination, EmptyState, Alert, Tag, Toolbar, PageContainer, Dashboard cards, or status panels.
- Page states: loading, empty, error, readonly, disabled, submitting, refreshing, success, failure, partial data, no permission, no data, request failure.
- User interaction issues: click has no effect, double-click, duplicate submission, Modal won't close, Drawer state is wrong on reopen, wrong form default values, form validation display issues, unclear button disabled state, wrong Tab switch state, wrong deep-link targeting, wrong page state after browser refresh, keyboard interaction issues, focus state issues.
- UI copy: button copy, page titles, page descriptions, status copy, empty-state copy, error messages, risk warnings, the semantics of "create draft" vs. "publish," the semantics of "task created" vs. "task completed," actionable suggestions for the user.
- Accessibility: aria-selected, aria-label, role, keyboard navigation, focus, form error association, unnamed icon buttons, Tab accessibility, Modal/Drawer accessibility.

When working in the following directories or files, you should proactively assess whether UI is involved; as long as page rendering, component structure, `className`, styles, interaction, or state display is involved, this standard must be automatically applied:

- `admin/src/pages/**`
- `admin/src/components/**`
- `admin/src/layouts/**`
- `admin/src/app.tsx`
- `admin/src/global.less`
- `**/*.tsx`
- `**/*.jsx`
- `**/*.less`
- `**/*.css`
- `**/*.scss`

You must not skip this standard just because the user describes the task as "fixing a bug," "a small issue," "a tweak," "changing a button," "changing a table," "fixing a display issue," or "optimizing a page." Do not ask the user whether they want this standard applied — it is not optional.

A purely frontend task may skip the full UI workflow only after confirming it has zero effect on the DOM, `className`, styles, user interaction, page state, loading/error/empty, and responsiveness. Scenarios that may exclude the full UI workflow include: pure TypeScript type fixes, pure service wrapping, pure API parameter fixes, pure data transformation, pure utility functions, pure build configuration, pure test configuration, pure dependency upgrades, pure performance calculation logic, and internal refactors that don't affect DOM/styles/interaction. If any one of these changes, this standard must still be automatically applied.

Mixed tasks must be split: the UI portion follows this standard; the API, backend, database, and business logic portions follow the existing service, DTO, permission, and business rules. Do not skip the UI standard just because a task includes an API, and do not change the API, payload, permissions, readonly state, or state machine in the name of UI optimization.

Only ask the user first in these cases: a change to business flow, API, payload, permissions, readonly state, or state machine is required; multiple options exist that would clearly affect product direction; the standard conflicts with the user's explicit request.

## 2. Scope

Directories and files that must default to this standard:

- `admin/src/pages/**`
- `admin/src/components/**`
- `admin/src/layouts/**`
- `admin/src/app.tsx`
- `admin/src/global.less`
- Admin-related CSS/LESS/TSX/JSX
- New and modified tasks corresponding to Admin routes

Tasks that must use this standard:

- Adding a new Admin page, modifying an existing Admin page, page refactors
- Page visual optimization, layout adjustments, responsive fixes, UI bug fixes, accessibility fixes
- Adding or modifying shared components
- Adding tables, filter areas, toolbars, forms, detail pages
- Developing or adjusting Modal, Drawer, Popconfirm
- Developing Dashboards, workbenches, empty states, error states, loading states, readonly/disabled states
- Modifying copy and interaction states
- Mobile adaptation

Purely backend tasks, database tasks, and non-Admin frontend tasks are not required to apply the full UI workflow.

## 3. General Design Principles

Must:

- Preserve the project's existing design language first; additions and modifications must be visually consistent.
- Prioritize information hierarchy over decoration — structure, state, and feedback should be clear before considering visual presentation.
- Business state must be real; do not fabricate metrics, progress, or conclusions.
- High-risk actions must be clearly distinguishable, but red must not be overused broadly.
- Copy must accurately reflect the real business impact — do not exaggerate what the system has actually done.

Prohibited:

- Building an independent design system for a single page.
- Using marketing-landing-page-style exaggerated design.
- Using large-area gradients, glassmorphism, or decoration with no business meaning.
- Adding visual noise just to "look premium."
- Using fake data, fake metrics, fake progress, or fake conclusions.

## 4. Shared Capabilities That Must Be Reused First

Before implementation, always check and prefer reusing:

- `TmPageContainer`
- `SectionCard`
- `MetricCard`
- `OperationToolbar`
- `TmProTable`
- `EmptyState`
- `AppDrawer`
- `layoutTokens`
- Ant Design Tokens
- Pro Components

Rules:

- Do not reimplement something an existing shared component can already satisfy.
- Do not create a component that duplicates the responsibility of `TmPageContainer`, `SectionCard`, or `TmProTable`.
- Do not duplicate a shared layout for a single page.
- Only consider abstracting a new shared component once three or more genuinely isomorphic scenarios exist.
- New components must have a single responsibility.
- Shared components must not couple to a page's business API, permissions, or state machine.

## 5. Page Container and the Horizontal Baseline

Must:

- `TmPageContainer` is the standard post-login page container.
- Breadcrumb, Page Header, Title, Description, Header Extra, and the body must use the same content track.
- Header and Content share the same `max-width`, `margin-inline`, and `padding-inline`.
- Use `layoutTokens.pageMaxWidth`.
- Standard list and table pages default to fully using the available width.
- A card's outer edge should align with the Page Header's content track.
- In principle, the outer-edge deviation between Header and main content must not exceed 4px.

Allowed:

- A clearly narrow form page may use `max-width`, but only with a genuine design purpose.
- Text inside a Card is not required to sit at exactly the same X coordinate as the page title text.

Prohibited:

- Adding a duplicate full-page left/right padding at the page root.
- Using a meaningless `margin-left`.
- Hardcoding the sidebar width.
- Using `width: calc(100vw - sidebarWidth)`.
- Fixing alignment via `transform`, negative margins, or hardcoded offsets.

## 6. Page Information Architecture

Recommended order for a standard page:

1. Breadcrumb
2. Page title
3. Page description
4. Header Extra / primary action
5. Status or context explanation
6. Filters and toolbar
7. Main content
8. Pagination, record count, or technical info

Must:

- A page has exactly one clear primary action.
- Secondary actions must not all use `primary`.
- View, configure, submit, and dangerous actions must be layered.
- Request failure, empty state, unconfigured state, readonly, and permission errors must reflect real state.

Copy semantics must be precise:

- "Create draft" must not be described as "publish successful."
- "Task created successfully" must not be described as "task execution complete."
- "Check passed" must not be described as "approved by the platform."
- A request failure must not be shown as an empty state.
- An unconfigured state must not be shown as a load failure.
- Readonly must not be shown as a permission error unless that's genuinely the logic.

## 7. Lists and Tables

Must:

- Prefer `TmProTable`.
- Filter area, toolbar, and table use a clear hierarchy.
- The table card's outer edge aligns with the page content track.
- The table can scroll horizontally within its own container.
- The page root must not scroll horizontally.
- Use a stable, unique `rowKey`.
- Long IDs, URLs, store names, platform names, and error messages must be truncated, wrapped, or shown via Tooltip.
- Status Tags must map to real states.
- Unknown states should fall back safely and must not be disguised as success.
- Empty state, load failure, and no permission must be distinguished.

Prohibited:

- Using the array `index` as a persistent list's `rowKey`.
- Using a random value that changes on every render as `rowKey`.
- Squeezing the actions column with a meaningless fixed width.

Temporary client-side records must generate and keep a stable temporary ID.

## 8. Forms

Must:

- Preserve the Form's real data structure.
- Loading/disabled/validation states must be explicit.
- Do not clear user input for no reason after a form failure.
- Readonly behavior follows the existing permission rules.
- When diagnosing a Modal Form mounting issue, pinpoint the specific Form instance and the Modal lifecycle.

Prohibited:

- Arbitrarily changing `Form.Item name`.
- Arbitrarily changing default values.
- Auto-selecting the first option unless the original business logic clearly supports it.
- Auto-save, auto-submit.
- Nested Forms.
- A Button causing double submission via both `htmlType="submit"` and `onClick`.
- Guessing a fix for a useForm warning via a blanket `forceRender`.

## 9. Modal, Drawer, Popconfirm

Modal must check:

- Title, current business context, default values, validation
- Loading, confirmLoading, disabled
- Cancel, state cleanup on close, footer
- Long text, mobile width
- A single confirm sends exactly one request

Drawer must check:

- Loading, normal, empty, error
- Long text, internal scrolling, close, reopen state
- Near-full width at 375px
- No horizontal overflow at the root

High-risk actions must:

- Retain confirmation.
- Not send a request on cancel.
- Use the danger tier for the action.
- Not sit at the same level as a plain view button.
- Not skip the original confirmation logic.

## 10. Loading, Empty, Error, Readonly

Every async module must consider at least:

- Initial loading
- Refreshing
- Success
- Empty
- Partial data
- Request error
- Business error
- Readonly
- Disabled
- Submitting

Must:

- An error must not be disguised as empty data.
- Empty data must not be disguised as an error.
- Do not flash the previous object's data during loading.
- On failure, preserve user input and existing data per the original logic.
- Readonly must only preserve existing business semantics.
- If permission coverage is incomplete, record the issue rather than unilaterally unifying it.

Prohibited:

- The AI subjectively expanding readonly and permission policy.

## 11. Responsive Standard

Mandatory acceptance viewports:

- 1440x900
- 1280x800
- 1024x768
- 768x900
- 375x812

Must:

- No horizontal overflow at the page root.
- Header and Content use the same gutter.
- At 375px, the page gutter generally stays 12px-16px.
- Cards do not exceed the viewport.
- The action area may wrap.
- Modal is near-full width but doesn't exceed the viewport.
- Drawer is near-full width and scrollable.
- Tables scroll only within their own container.
- Tabs may scroll internally but must not cause page-root overflow.
- Long text must be able to wrap.
- Sidebar expand and collapse both work correctly.

Prohibited:

- Retaining meaningless large desktop padding.
- Accommodating content via horizontal scroll on the page root.

Root-node horizontal overflow standard:

```js
document.documentElement.scrollWidth <= document.documentElement.clientWidth;
document.body.scrollWidth <= document.body.clientWidth;
```

## 12. Style Standard

Must:

- Page styles are preferentially written into a page-local LESS file.
- Shared container styles go into the corresponding shared component or existing shell styles.
- Prefer Ant Design Tokens.
- Prefer `layoutTokens`.
- Global selectors must have a scoped qualifier.
- Modifying `global.less` requires demonstrating that the page shell or a genuinely global shared root cause requires it.
- Do not uniformly change LF/CRLF when editing styles.

Prohibited:

- Scattered, duplicated HEX values.
- Adding new global `.ant-*` overrides.
- Adding new `!important`.
- Fixing layout via negative margins.
- Fixing alignment via `transform`.
- Rewriting an entire LESS file.
- Affecting the login page, error pages, Modal, Drawer, or full-screen pages.

## 13. Copy Standard

Must:

- Copy accurately reflects the real business impact.
- "Save," "create draft," "submit," "publish," "sync," and "generate task" are clearly distinguished.
- Buttons use clear verbs.
- High-risk buttons state the outcome clearly.
- Technical errors and actionable user suggestions are shown in separate layers.
- Run `pnpm.cmd check:ui-copy --strict` after copy changes.

Prohibited:

- Exaggerated marketing copy.
- Using vague "OK" / "complete now" language in place of the real action.
- Describing an incomplete process as "complete."

## 14. Accessibility

Must:

- Tabs `activeKey`, `aria-selected`, and the active pane are consistent.
- Button, Link, Tab, Modal, and Drawer support keyboard operation.
- Focus styles are clear.
- Disabled state is recognizable.
- Icon buttons must have an accessible name.
- Form errors are associated with their fields.

Prohibited:

- Using Tooltip as the sole source of information.
- Hand-writing aria attributes that conflict with component state.
- Using coordinate-based clicks in place of role/locator-based acceptance testing.

## 15. Business Protection Rules

By default, UI tasks must not modify:

- API URL
- HTTP method
- Service
- DTO
- Request payload
- Handler
- Routing
- Permissions
- Readonly state
- State machine
- Reload order
- Pagination params
- Sorting
- Business logic decisions
- Auto-refresh policy
- Task state
- Platform state
- Inventory semantics
- Publish semantics

If a business behavior genuinely must change, it must first be clearly stated, explained, its impact listed, and the user's confirmation obtained. Do not quietly change behavior under the guise of "UI optimization."

Do not add any of the following automatic behaviors unless the requirement explicitly requires it and it has been confirmed:

- Auto-save
- Auto-submit
- Auto-select
- Auto-retry
- Auto-poll
- Auto-create task
- Auto-upload
- Auto-sync
- Auto-bind
- Auto-publish
- One-click "fix everything"

## 16. AI Implementation Workflow

### Step 1: Read the Standard

At the start of the task, must read:

- `AGENTS.md`
- `.agents/skills/frontend-design/SKILL.md`
- The Cursor rules applicable to the current directory
- The target page's TSX/JSX
- The target page's LESS/CSS
- Related shared components
- `TmPageContainer`
- `layoutTokens`
- Existing UI wrappers related to the target page
- The target page's existing implementation

### Step 2: Check Git

Run:

```bash
git status --short --branch
git diff --stat
git log -5 --oneline
```

Do not overwrite the user's existing changes. Do not create branches, commit, push, reset, restore, clean, or stash unless explicitly requested by the user. `git add .` is prohibited.

### Step 3: Locate the Real Code

Must locate first:

- Page JSX
- Page LESS
- Shared UI
- Service
- Types
- Handler
- Loading
- Error
- Readonly
- Disabled
- URL state
- Modal/Drawer
- Table `rowKey`
- Section id

Do not guess the business implementation from UI screenshots.

### Step 4: Analyze Before Modifying

Before modifying, must provide:

- Current information architecture
- Real business flow
- UI issues
- Proposed scope of changes
- Business behavior that stays unchanged
- Risks
- Verification plan

For clear, small-scope tasks, a brief analysis may be given and work can proceed without waiting for confirmation.

Must stop and ask the user in these situations:

- Changing the API is required
- Changing the payload is required
- Changing permissions is required
- Changing the state machine is required
- The requirement clearly conflicts with existing business logic
- Two different implementation choices exist that affect the business
- The real effect of a write operation cannot be confirmed

### Step 5: Minimal Implementation

Must:

- Prefer reusing shared components.
- Change only the target scope.
- A pure UI task must not change business behavior.

Prohibited:

- Opportunistic refactoring.
- Reformatting an entire file.
- Modifying unrelated Tabs.
- Duplicating existing logic.
- Creating duplicate components.

### Step 6: Static Checks

Must run at least:

```bash
git diff --check
pnpm.cmd check:dev
pnpm.cmd check:ui-copy --strict
pnpm.cmd build:admin
git diff --stat
git diff --numstat
git status --short --branch
```

If the project later adds a stable lint/typecheck tool, run that too, but do not add a new dependency ad hoc.

### Step 7: Browser Acceptance

The automated acceptance requirements after Admin UI implementation are uniformly defined by `.agents/skills/admin-e2e-testing/SKILL.md`; this Skill only retains the UI acceptance entry point and does not duplicate the full E2E standard.

Admin page changes should, in principle, use Playwright MCP for acceptance. When the Admin service was started by the user, do not start it yourself, do not stop it, and do not kill its process; if the service is unavailable, stop and report.

Must:

- Use the five-tier viewport matrix.
- Check root-node overflow.
- Check Header/Content alignment.
- Check loading/empty/error.
- Check Modal/Drawer/Popconfirm.
- Check long text.
- Check readonly.
- Check console warnings/errors.

All non-GET requests must be intercepted with `browser_route`. Real platform write operations must never be executed.

### Step 8: Network Side-Effect Check

For write operations, must capture:

- Method
- URL
- Path params
- Query
- Payload
- Count
- Reload
- Extra requests

Must verify:

- Cancel sends no request.
- A single confirm sends exactly one request.
- Rapid repeated confirms don't cause duplicate submission.
- Failure does not auto-retry.
- No unrelated business write requests are triggered.

### Step 9: Report Results

Must report:

- Current branch
- Starting workspace state
- Modified files
- What was changed
- Behavior that remains unchanged
- Check commands run
- Browser viewports
- Overflow data
- Write requests
- Console info
- Current diff
- Uncommitted files
- Remaining risks
- Whether it's ready for sign-off
- Whether it's suitable for manual acceptance

Do not commit or push unless the user explicitly requests it.

## 17. Mandatory Checklist for New Admin Pages

A new page must:

- Use the standard post-login Layout.
- Prefer `TmPageContainer`.
- Header and Content share a content track.
- Provide breadcrumb, title, description, and extra based on real need.
- Use shared components such as `SectionCard`/`TmProTable`/`OperationToolbar`.
- Provide loading, empty, error.
- Consider readonly, permissions, long text.
- Provide a stable `rowKey`.
- Support the five-tier viewport matrix.
- Have no overflow at the root.
- Not duplicate existing page layout code.
- Not add a second design token system.
- Not modify global styles to solve a single-page problem.
- Not auto-execute write operations.
- Add browser mock acceptance coverage.
- Pass all check commands.

A new page must not directly use a bare `PageContainer`, `ProTable`, `Card`, or a custom page wrapper without justification; first confirm whether an existing project shared wrapper already applies.

## 18. Mandatory Checklist for Modifying Existing Pages

Before modifying, must confirm:

- Original API
- Original payload
- Original handler
- Original state machine
- Original permissions
- Original readonly state
- Original loading
- Original URL
- Original reload
- Original write-request count

After modifying, must confirm:

- No business regression.
- No new duplicate submissions.
- No broken deep links.
- No new root-node overflow.
- No new key warnings.
- No new useForm warnings.
- No error displayed as an empty state.
- No draft described as formally complete.
- No broken mobile layout.
- No unrelated diff expansion.

## 19. Master Prohibited List

Prohibited:

- Tailwind
- shadcn
- Adding a new UI framework
- Adding a dependency that duplicates Ant Design
- Large-scale global CSS overrides
- Global `.ant-*` hacking
- `!important`
- Negative-margin offsetting
- Using `transform` to fix layout position
- Hardcoding the sidebar width
- Duplicating page gutters
- Cards nested inside cards
- Every button being `primary`
- Every state being a `MetricCard`
- Fake data and fake metrics
- Disguising an API change as a UI optimization
- Blanket, speculative `forceRender` additions to silence warnings
- Using `index` as a persistent table `rowKey`
- Opportunistically changing business logic during a UI task
- Executing a real write operation without interception
- Declaring completion without acceptance testing
- Committing or pushing without authorization

## 20. Discoverability and Entry-Point Maintenance

- `AGENTS.md` is the cross-tool project entry point; it should only require reading this skill and list the gates, without duplicating the full standard.
- The Cursor rule is only responsible for automatically applying to Admin TSX/JSX/LESS/CSS files and pointing to this skill.
- If `CLAUDE.md` exists for Claude Code, it should only reference this skill, not duplicate the full standard.
- Other AI instruction files may add a reference if they already exist; do not blindly create one if it doesn't.
- No multiple files may simultaneously claim to be the single complete source of truth for the Admin UI standard.
