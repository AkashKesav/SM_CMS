import { TableSchema } from '@/types/schema';

export function getPrimaryKeys(schema: TableSchema): string[] {
  if (schema.primary_keys && schema.primary_keys.length > 0) {
    return schema.primary_keys;
  }
  if (schema.primary_key) {
    return [schema.primary_key];
  }
  return ['id'];
}

export function getRecordIdentity(schema: TableSchema, record: Record<string, any>): string {
  const primaryKeys = getPrimaryKeys(schema);

  if (primaryKeys.length === 1) {
    const value = record[primaryKeys[0]];
    return value == null ? '' : String(value);
  }

  const compositeKey = primaryKeys.reduce<Record<string, any>>((acc, key) => {
    acc[key] = record[key];
    return acc;
  }, {});

  return JSON.stringify(compositeKey);
}
