# Shared type definitions and validation schemas
# This directory serves as the contract between Go backend and TypeScript frontend

## Schema Types

See `frontend/src/types/schema.ts` for TypeScript definitions.
Go equivalents are in `backend/internal/model/schema.go`.

## Code Generation (Future)

To keep Go structs and TypeScript interfaces in sync, use:

### Option 1: OpenAPI/Swagger Codegen
```bash
# Generate TypeScript client from OpenAPI spec
openapi-generator-cli generate -i backend/openapi.yaml -g typescript-axios -o shared/generated/
```

### Option 2: JSON Schema Bridge
```bash
# Export Go types as JSON Schema
go run ./tools/types-export > shared/schema.json

# Generate TypeScript from JSON Schema
quicktype shared/schema.json -o shared/types.ts -l typescript
```

### Option 3: Manual Contract
For now, types are maintained manually in both languages. Keep these files synchronized:

| Go Type | TypeScript Type | Location |
|---------|----------------|----------|
| `model.ColumnInfo` | `ColumnInfo` | `backend/internal/model/schema.go` ↔ `frontend/src/types/schema.ts` |
| `model.TableSchema` | `TableSchema` | Same |
| `model.DatabaseSchema` | `DatabaseSchema` | Same |
| `model.ForeignKeyInfo` | `ForeignKeyInfo` | Same |
