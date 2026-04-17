# @cropdoc/web-admin

Internal agronomist / staff admin portal. Same stack as `web-client` with a denser theme (via `data-app="admin"`).

## Dev

```bash
pnpm install
pnpm --filter @cropdoc/web-admin dev      # :5174

# Go API
cd services/api && make run                # :8080
```

Deployed at `admin.cropdoc.lk`. Access restricted — future wiring: email + TOTP with optional IP allowlist.

## Layout

```
src/
├── main.tsx
├── routes/
│   ├── __root.tsx       # Sidebar shell
│   ├── index.tsx        # Dashboard
│   └── crops.tsx        # Crops review table
├── lib/
│   ├── api.ts           # (copied from web-client)
│   └── utils.ts
└── styles/globals.css
```
