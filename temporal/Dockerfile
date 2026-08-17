FROM node:22-bookworm-slim

WORKDIR /workspace
RUN corepack enable

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml .npmrc ./
COPY temporal/package.json temporal/package.json
COPY scripts/patch-pro-field-antd-select.mjs scripts/patch-pro-field-antd-select.mjs
RUN pnpm install --frozen-lockfile --filter @trademind/temporal...

COPY temporal temporal
USER node
CMD ["pnpm", "--filter", "@trademind/temporal", "worker"]
