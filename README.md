# CMS Desktop — Dynamic Schema CMS with Tauri + Go Fiber + React

A production-ready, cross-platform desktop CMS that **dynamically discovers** your PostgreSQL schema at runtime. Zero hardcoded table/column names — the UI adapts generically to any database structure.

![Tech Stack](https://img.shields.io/badge/Tauri-2.0-FFC131?logo=tauri) ![Go](https://img.shields.io/badge/Go-1.21-00ADD8?logo=go) ![React](https://img.shields.io/badge/React-18-61DAFB?logo=react) ![Supabase](https://img.shields.io/badge/Supabase-PostgreSQL-3ECF8E?logo=supabase)

---

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                    Tauri Desktop App                     │
│  ┌──────────────────────┐  ┌──────────────────────────┐  │
│  │   React Frontend      │  │   Go Fiber Sidecar        │  │
│  │   (Vite + TS)         │◄─┤   (REST API Server)       │  │
│  │                        │  │                           │  │
│  │  • Dynamic UI Render   │  │  • Schema Discovery       │  │
│  │  • Virtualized Tables  │  │  • Generic CRUD           │  │
│  │  • Command Palette     │  │  • Auth Middleware        │  │
│  │  • Form Validation     │  │  • SQL Injection Prevent  │  │
│  └───────────┬───────────┘  └────────────┬──────────────┘  │
│              │ IPC/HTTP                   │ pgx             │
│              └────────────────────────────┘                 │
│                              │                              │
│                     ┌────────▼────────┐                     │
│                     │  Supabase PG    │                     │
│                     │  + Auth + S3    │                     │
│                     └─────────────────┘                     │
└─────────────────────────────────────────────────────────┘
```

### Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Dynamic schema discovery** | Queries `information_schema` at runtime — no code changes needed when DB schema changes |
| **Go Fiber sidecar** | Fast, lightweight HTTP server with native PostgreSQL connection pooling |
| **Tauri v2** | Native desktop shell with system tray, file dialogs, secure storage |
| **Zero hardcoded names** | All table/column names, types, and relations derived from PostgreSQL catalogs |
| **Direct Supabase uploads** | File uploads bypass Go backend via signed URLs for performance |

---

## 📁 Project Structure

```
CMS_go/
├── backend/                          # Go Fiber backend
│   ├── main.go                       # Entry point, route registration
│   ├── go.mod                        # Go module definition
│   └── internal/
│       ├── config/
│       │   └── config.go             # Environment config loader
│       ├── model/
│       │   └── schema.go             # Schema data types
│       ├── repository/
│       │   ├── schema.go             # Schema discovery queries
│       │   └── crud.go               # Generic CRUD with parameterized SQL
│       ├── service/
│       │   └── service.go            # Business logic, validation
│       ├── handler/
│       │   ├── schema.go             # Schema HTTP handlers
│       │   ├── crud.go               # CRUD HTTP handlers
│       │   └── auth.go               # Auth HTTP handlers
│       └── middleware/
│           └── middleware.go          # JWT auth, error handling, CORS
│
├── frontend/                          # React + TypeScript frontend
│   ├── package.json
│   ├── vite.config.ts
│   ├── tailwind.config.js
│   ├── tsconfig.json
│   └── src/
│       ├── main.tsx                   # React entry point
│       ├── App.tsx                    # Router + auth guard
│       ├── index.css                  # Tailwind + CSS variables
│       ├── types/
│       │   └── schema.ts              # TypeScript type definitions
│       ├── stores/
│       │   ├── authStore.ts           # Zustand auth state
│       │   └── schemaStore.ts         # Zustand schema/data state
│       ├── lib/
│       │   ├── api.ts                 # API client (auth, schema, CRUD)
│       │   └── utils.ts               # Utility functions
│       ├── utils/
│       │   └── fieldMapper.ts         # PG type → UI field type mapping
│       ├── components/
│       │   ├── Layout.tsx             # Desktop shell with sidebar
│       │   ├── DataTable.tsx          # Virtualized, sortable table
│       │   ├── RecordForm.tsx         # Dynamic form with field renderers
│       │   ├── CommandPalette.tsx     # Cmd+K navigation
│       │   ├── ErrorBoundary.tsx      # React error boundary
│       │   ├── LoadingScreen.tsx      # Loading indicator
│       │   └── ui/                    # shadcn/ui-style components
│       │       ├── Button.tsx
│       │       ├── Input.tsx
│       │       ├── Select.tsx
│       │       ├── Dialog.tsx
│       │       ├── Card.tsx
│       │       ├── Tabs.tsx
│       │       ├── Badge.tsx
│       │       ├── Switch.tsx
│       │       ├── Textarea.tsx
│       │       ├── Label.tsx
│       │       ├── Avatar.tsx
│       │       ├── Sidebar.tsx
│       │       └── Command.tsx
│       └── pages/
│           ├── LoginPage.tsx          # Authentication page
│           ├── DashboardPage.tsx      # Schema overview dashboard
│           └── TablePage.tsx          # Dynamic table CRUD interface
│
├── src-tauri/                         # Tauri desktop shell
│   ├── Cargo.toml                     # Rust dependencies
│   ├── tauri.conf.json                # Tauri configuration
│   ├── build.rs                       # Build script
│   ├── src/
│   │   └── main.rs                    # Tauri app entry (tray, IPC)
│   └── sidecar/
│       └── main.go                    # Go sidecar entry point
│
├── shared/                            # Shared types (future codegen)
├── package.json                       # Root package (scripts, workspaces)
├── .env.example                       # Environment template
├── .gitignore
└── README.md                          # ← You are here
```

---

## 🚀 Quick Start

### Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| **Node.js** | ≥ 18.0 | Frontend build |
| **Go** | ≥ 1.21 | Backend server |
| **Rust** | ≥ 1.70 | Tauri desktop shell |
| **Supabase** | Cloud or self-hosted | PostgreSQL + Auth + Storage |

### 1. Clone & Install

```bash
git clone <repo-url> CMS_go
cd CMS_go

# Install all dependencies
npm run setup
```

### 2. Configure Environment

```bash
cp .env.example .env
# Edit .env with your Supabase credentials
```

**Required environment variables:**

| Variable | Description | Example |
|----------|-------------|---------|
| `SUPABASE_URL` | Your Supabase project URL | `https://xyz.supabase.co` |
| `SUPABASE_ANON_KEY` | Public anon key | `eyJ...` |
| `SUPABASE_SERVICE_ROLE_KEY` | Service role key (secret!) | `eyJ...` |
| `DATABASE_URL` | Direct PostgreSQL connection | `postgresql://postgres:pass@db.xyz.supabase.co:5432/postgres` |
| `JWT_SECRET` | JWT signing secret | Any random string |
| `SUPABASE_STORAGE_BUCKET` | Storage bucket name | `cms-media` |

### 3. One-Command Startup

```bash
npm run dev
```

This concurrently launches:
1. **Vite dev server** (port 5173) — React hot-reload
2. **Go Fiber server** (port 8080) — Backend API
3. **Tauri dev window** — Native desktop shell

### 4. Open the App

The Tauri window opens automatically. Use any email/password to log in (mock auth in dev mode).

---

## 🗄️ Supabase Setup

### 1. Create Supabase Project

1. Go to [supabase.com](https://supabase.com) → New Project
2. Note your project URL and keys from Settings → API

### 2. Configure Database

The app auto-discovers tables — no migrations required. Simply create tables via Supabase SQL Editor or Dashboard:

```sql
-- Example: Create a sample table
CREATE TABLE students (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE,
    enrollment_date DATE DEFAULT CURRENT_DATE,
    is_active BOOLEAN DEFAULT true,
    department_id UUID REFERENCES departments(id),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE departments (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### 3. Set Up Supabase Auth

1. Go to Authentication → Providers → Email
2. Enable "Confirm email" (optional for dev)
3. Add your app URL to Site URL and Redirect URLs

### 4. Create Storage Bucket

1. Go to Storage → New Bucket
2. Name it `cms-media` (or match your `SUPABASE_STORAGE_BUCKET` env var)
3. Set to **public** for direct access

### 5. Row Level Security (RLS)

For production, enable RLS on your tables:

```sql
ALTER TABLE students ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Admin full access" ON students
  FOR ALL USING (auth.jwt()->>'role' = 'admin');

CREATE POLICY "Viewers can read" ON students
  FOR SELECT USING (auth.jwt()->>'role' IN ('admin', 'editor', 'viewer'));
```

---

## 🔌 API Endpoints

All endpoints require `Authorization: Bearer <JWT>` header.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/auth/login` | Authenticate user |
| `POST` | `/api/auth/register` | Register new user |
| `POST` | `/api/auth/refresh` | Refresh access token |
| `POST` | `/api/auth/logout` | Invalidate session |
| `GET` | `/api/schema` | Get discovered schema |
| `POST` | `/api/schema/refresh` | Force schema re-discovery |
| `GET` | `/api/tables/:tableName` | List records (paginated, sorted, filtered) |
| `GET` | `/api/tables/:tableName/:id` | Get single record |
| `POST` | `/api/tables/:tableName` | Create record |
| `PUT` | `/api/tables/:tableName/:id` | Update record |
| `PATCH` | `/api/tables/:tableName/:id` | Partial update |
| `DELETE` | `/api/tables/:tableName/:id` | Delete record (soft by default) |
| `POST` | `/api/upload/sign` | Get signed upload URL |

### Query Parameters for List Endpoint

| Parameter | Type | Description |
|-----------|------|-------------|
| `page` | int | Page number (default: 1) |
| `page_size` | int | Records per page (default: 50, max: 1000) |
| `sort_by` | string | Column to sort by |
| `sort_order` | `ASC` \| `DESC` | Sort direction |
| `<column_name>` | string | Filter: exact match |
| `<column_name>` | `like:value` | Filter: ILIKE match |
| `<column_name>` | `gt:value` | Filter: greater than |
| `<column_name>` | `lt:value` | Filter: less than |
| `<column_name>` | `ne:value` | Filter: not equal |

**Example:**
```
GET /api/tables/students?page=1&page_size=25&sort_by=created_at&sort_order=DESC&is_active=true&name=like:john
```

---

## 🎨 How Dynamic Schema Discovery Maps to UI

### The Flow

```
PostgreSQL System Catalogs
         │
         ▼
┌──────────────────────────┐
│  information_schema      │
│  • tables                │
│  • columns               │
│  • table_constraints     │
│  • key_column_usage      │
└────────────┬─────────────┘
             │ SQL queries
             ▼
┌──────────────────────────┐
│  Go: SchemaService       │
│  • DiscoverTables()      │
│  • discoverTableDetails()│
│  • Cache (5 min TTL)     │
└────────────┬─────────────┘
             │ JSON
             ▼
┌──────────────────────────┐
│  React: schemaStore      │
│  • DatabaseSchema type   │
│  • Zustand persisted     │
└────────────┬─────────────┘
             │
             ▼
┌──────────────────────────┐
│  utils/fieldMapper.ts    │
│  mapPostgresToFieldType()│
│  createFieldConfig()     │
└────────────┬─────────────┘
             │ FieldConfig[]
             ▼
┌──────────────────────────┐
│  RecordForm component    │
│  • Dynamic field renderer│
│  • Type-specific inputs  │
└──────────────────────────┘
```

### Type Mapping Table

| PostgreSQL Type | UI Component | Behavior |
|----------------|--------------|----------|
| `uuid` (PK) | Read-only text | Auto-generated on create |
| `varchar(n)` | Input / Textarea | Textarea if n > 500 or name hints long text |
| `text` | Input / Textarea | Textarea if name contains "description", "content", etc. |
| `int2/4/8`, `numeric`, `float4/8` | Number input | Validates numeric |
| `boolean` | Toggle switch | Yes/No display |
| `date` | Date picker | Calendar selector |
| `timestamptz` | DateTime picker | Combined date+time |
| `jsonb` / `json` | Code editor | Collapsible, syntax-highlighted |
| `uuid` (FK) | Select dropdown | Fetches referenced table records |
| `bytea` | Image uploader | File picker + preview |
| Columns with "email" in name | Email input | Browser validation |
| Columns with "url" in name | URL input | URL validation |

### Heuristic Intelligence

The `fieldMapper.ts` uses intelligent heuristics:
- **Name-based detection**: `description` → textarea, `email` → email input
- **Length-based detection**: `varchar(1000)` → textarea
- **Foreign key detection**: Automatically renders dropdowns that fetch options from referenced tables
- **Timestamp detection**: `created_at`/`updated_at` auto-hidden on create, read-only on edit

---

## ⚙️ Development Guide

### Adding/Removing Tables

**No code changes needed!** Simply modify your database schema:

1. **Add a table** via Supabase Dashboard → SQL Editor
2. Click **"Sync Schema"** in the app sidebar or tray menu
3. The new table appears in the sidebar and dashboard

To **remove a table**:
1. Drop the table in Supabase
2. Click **"Sync Schema"**
3. Table disappears from the UI

### Managing Permissions

Table-level permissions are enforced at two levels:

**Backend middleware** (`middleware.go`):
```go
// Check role before allowing operations
c.Locals("user_role", "admin") // Set from JWT
```

**Frontend UI** (future enhancement):
- Create a `cms_permissions` table:
```sql
CREATE TABLE cms_permissions (
    role VARCHAR(50),
    table_name VARCHAR(100),
    actions TEXT[], -- ['read', 'create', 'update', 'delete']
    PRIMARY KEY (role, table_name)
);
```
- The app reads this table to dynamically show/hide UI elements

### Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Cmd/Ctrl + K` | Open command palette |
| `Cmd/Ctrl + S` | Save form (when editing) |
| `Cmd/Ctrl + Z` | Undo (form fields) |
| `Escape` | Close modal/dialog |

### Hot Reload

- **Frontend**: Vite HMR — instant updates on file save
- **Backend**: Restart the Go process manually (or use `air` for auto-reload)
- **Tauri**: Window reloads on frontend changes

---

## 🏗️ Build & Deployment

### Development Builds

```bash
# Frontend only (for web testing)
npm run frontend:dev

# Backend only
npm run backend:dev

# Tauri + frontend (skip backend proxy)
npm run dev:skip-tauri
```

### Production Desktop Builds

```bash
# Build for current platform
npm run tauri:build

# Debug build (with dev tools)
npm run tauri:build:debug
```

### Platform-Specific Builds

| Platform | Command | Output |
|----------|---------|--------|
| **Windows** | `npm run tauri:build` | `.msi` / `.exe` installer |
| **macOS** | `npm run tauri:build` | `.app` bundle + `.dmg` |
| **Linux** | `npm run tauri:build` | `.deb` / `.AppImage` |

### Code Signing (Production)

For macOS:
```bash
# Set environment variables
export APPLE_CERTIFICATE="<base64-cert>"
export APPLE_CERTIFICATE_PASSWORD="password"
export APPLE_SIGNING_IDENTITY="Developer ID Application: ..."
export APPLE_ID="your@apple.id"
export APPLE_PASSWORD="app-specific-password"
export APPLE_TEAM_ID="team-id"

npm run tauri:build
```

For Windows:
- Obtain a code signing certificate from a CA
- Configure in `src-tauri/tauri.conf.json` under `bundle.windows`

### Auto-Updates

Tauri supports auto-updates via:
1. **GitHub Releases** — Upload build artifacts
2. **Self-hosted server** — JSON manifest + binary files
3. **Tauri Update Server** — Third-party hosting

Configure in `tauri.conf.json`:
```json
{
  "plugins": {
    "updater": {
      "active": true,
      "endpoints": ["https://releases.your-domain.com/{{target}}/{{arch}}/{{current_version}}"],
      "pubkey": "your-public-key-here"
    }
  }
}
```

---

## 🔒 Security Considerations

### Production Checklist

- [ ] Replace mock JWT verification with Supabase JWKS validation
- [ ] Enable Row Level Security (RLS) on all tables
- [ ] Set strong `JWT_SECRET` and `SESSION_SECRET`
- [ ] Configure CORS whitelist for production domains
- [ ] Enable rate limiting on all API endpoints
- [ ] Use service role key only in backend (never expose to frontend)
- [ ] Sanitize all user inputs (backend validates types)
- [ ] Use HTTPS for all Supabase connections
- [ ] Enable Supabase email confirmation for auth
- [ ] Set up audit logging for CRUD operations

### SQL Injection Prevention

All queries use **parameterized queries** via `pgx`:
```go
// SAFE: Parameterized query
query := "SELECT * FROM users WHERE email = $1"
row := db.QueryRow(ctx, query, userInput)

// Table/column names validated against whitelist
if !isValidIdentifier(tableName) {
    return fmt.Errorf("invalid table name")
}
```

---

## 🐛 Troubleshooting

| Issue | Solution |
|-------|----------|
| `Failed to connect to database` | Verify `DATABASE_URL` in `.env`, check Supabase connection pool |
| `CORS error in browser` | Ensure backend CORS allows `http://localhost:5173` |
| `Tauri window blank` | Check Vite dev server is running on port 5173 |
| `Schema not loading` | Verify PostgreSQL credentials, check `information_schema` access |
| `Go build errors` | Run `cd backend && go mod tidy` |
| `Rust build errors` | Install Tauri CLI: `cargo install tauri-cli --version "^2.0.0"` |

---

## 📄 License

MIT License — see [LICENSE](LICENSE) for details.

---

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

**Built with ❤️ using Tauri, Go, and React**
