import { ColumnInfo, ForeignKeyInfo, FieldConfig, FieldType } from '@/types/schema';

/**
 * Maps PostgreSQL data types to UI field types
 * This is the core mapping that enables dynamic schema rendering
 */
export function mapPostgresToFieldType(column: ColumnInfo, foreignKeys: ForeignKeyInfo[]): FieldType {
  // Foreign keys become select dropdowns
  if (column.is_foreign_key || foreignKeys.some(fk => fk.column_name === column.column_name)) {
    return 'select';
  }

  // Primary keys are read-only (auto-generated)
  if (column.is_primary_key) {
    if (column.udt_name === 'uuid') {
      return 'uuid'; // treated as read-only auto-generated unless it is also a FK
    }
  }

  // Map based on PostgreSQL type names
  switch (column.udt_name) {
    case 'bool':
      return 'boolean';
    case 'date':
      return 'date';
    case 'timestamptz':
    case 'timestamp':
      return 'datetime';
    case 'jsonb':
    case 'json':
      return 'json';
    case 'uuid':
      return 'uuid';
    case 'bytea':
      return 'image';
    case 'text':
      // Heuristic: columns with name hints should be textareas
      if (isLongTextField(column.column_name)) {
        return 'textarea';
      }
      if (isEmailField(column.column_name)) {
        return 'email';
      }
      if (isImageField(column.column_name)) {
        return 'image';
      }
      if (isUrlField(column.column_name)) {
        return 'url';
      }
      return column.character_maximum_length && column.character_maximum_length > 500 ? 'textarea' : 'text';
    case 'varchar':
      if (isImageField(column.column_name)) {
        return 'image';
      }
      if (isLongTextField(column.column_name) || (column.character_maximum_length && column.character_maximum_length > 500)) {
        return 'textarea';
      }
      // Email detection by name heuristic
      if (isEmailField(column.column_name)) {
        return 'email';
      }
      // URL detection
      if (isUrlField(column.column_name)) {
        return 'url';
      }
      return 'text';
    case 'int2':
    case 'int4':
    case 'int8':
    case 'numeric':
    case 'decimal':
    case 'real':
    case 'double':
    case 'float4':
    case 'float8':
      return 'number';
    default:
      // Fallback to text for unknown types
      return 'text';
  }
}

/**
 * Creates a complete field configuration from column metadata
 */
export function createFieldConfig(
  column: ColumnInfo,
  foreignKeys: ForeignKeyInfo[]
): FieldConfig {
  const type = mapPostgresToFieldType(column, foreignKeys);
  
  const config: FieldConfig = {
    name: column.column_name,
    label: formatColumnName(column.column_name),
    type,
    required: column.is_nullable === 'NO',
    nullable: column.is_nullable === 'YES',
    defaultValue: parseDefaultValue(column.column_default, type),
    maxLength: column.character_maximum_length || undefined,
    precision: column.numeric_precision || undefined,
    placeholder: getPlaceholder(type, column.column_name),
    description: getColumnDescription(column.column_name),
  };

  // Add foreign key reference info
  const fk = foreignKeys.find(fk => fk.column_name === column.column_name);
  if (fk) {
    config.foreignKey = fk;
  }

  return config;
}

/**
 * Converts snake_case to Title Case
 */
export function formatColumnName(name: string): string {
  return name
    .split('_')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}

/**
 * Detects if a column name suggests a long text field
 */
function isLongTextField(name: string): boolean {
  const longTextHints = ['description', 'content', 'body', 'text', 'note', 'comment', 'summary', 'details'];
  return longTextHints.some(hint => name.toLowerCase().includes(hint));
}

/**
 * Detects if a column name suggests an email field
 */
function isEmailField(name: string): boolean {
  return name.toLowerCase().includes('email') || name.toLowerCase() === 'mail';
}

/**
 * Detects if a column name suggests a URL field
 */
function isUrlField(name: string): boolean {
  const urlHints = ['url', 'uri', 'link', 'website'];
  return urlHints.some(hint => name.toLowerCase().includes(hint));
}

function isImageField(name: string): boolean {
  const nameLower = name.toLowerCase();
  const exact = ['image', 'avatar', 'photo', 'logo', 'thumbnail', 'banner', 'cover', 'picture'];
  const suffixes = ['_image', '_avatar', '_photo', '_logo', '_thumbnail', '_banner', '_cover', '_picture'];
  const urlHints = [
    'image_url',
    'avatar_url',
    'photo_url',
    'logo_url',
    'thumbnail_url',
    'banner_url',
    'cover_url',
    'picture_url',
    'image_path',
    'avatar_path',
    'photo_path',
    'logo_path',
    'thumbnail_path',
    'banner_path',
    'cover_path',
    'picture_path',
  ];

  if (exact.includes(nameLower)) {
    return true;
  }
  if (suffixes.some(suffix => nameLower.endsWith(suffix))) {
    return true;
  }
  return urlHints.some(hint => nameLower.includes(hint));
}

/**
 * Parses default values from PostgreSQL format
 */
function parseDefaultValue(defaultValue: string | null, _type: FieldType): any {
  if (!defaultValue) return undefined;
  
  // Remove function calls like now(), gen_random_uuid(), etc.
  if (defaultValue.includes('(')) {
    return undefined;
  }
  
  // Remove quotes
  const clean = defaultValue.replace(/^'(.*)'$/, '$1');
  
  if (clean === 'true') return true;
  if (clean === 'false') return false;
  if (clean === 'NULL') return null;
  
  const num = Number(clean);
  if (!isNaN(num)) return num;
  
  return clean;
}

/**
 * Gets appropriate placeholder for field type
 */
function getPlaceholder(type: FieldType, name: string): string {
  switch (type) {
    case 'email':
      return 'example@email.com';
    case 'url':
      return 'https://example.com';
    case 'number':
      return '0';
    case 'date':
      return 'YYYY-MM-DD';
    case 'datetime':
      return 'YYYY-MM-DD HH:MM';
    case 'json':
      return '{ "key": "value" }';
    case 'uuid':
      return 'auto-generated';
    default:
      return `Enter ${formatColumnName(name).toLowerCase()}`;
  }
}

/**
 * Generates a human-readable description from column name
 */
function getColumnDescription(name: string): string {
  const descriptions: Record<string, string> = {
    id: 'Unique identifier',
    created_at: 'Record creation timestamp',
    updated_at: 'Last update timestamp',
    deleted_at: 'Soft delete timestamp',
    name: 'Display name',
    title: 'Record title',
    slug: 'URL-friendly identifier',
    status: 'Current status',
    type: 'Classification type',
    description: 'Detailed description',
    email: 'Contact email address',
    phone: 'Contact phone number',
    password: 'Secure password hash',
    avatar: 'Profile image URL',
    image: 'Image URL or path',
    url: 'Web address',
    notes: 'Additional notes',
    order: 'Sort order',
    priority: 'Priority level',
    weight: 'Numeric weight',
    price: 'Monetary value',
    amount: 'Numeric amount',
    count: 'Item count',
    rating: 'Rating score',
    score: 'Numeric score',
    percentage: 'Percentage value',
    is_active: 'Whether this record is active',
    is_published: 'Whether this record is published',
    is_featured: 'Whether this record is featured',
    is_default: 'Whether this is the default',
    published_at: 'Publication timestamp',
    expires_at: 'Expiration timestamp',
    starts_at: 'Start timestamp',
    ends_at: 'End timestamp',
    parent_id: 'Reference to parent record',
    user_id: 'Reference to user',
    created_by: 'User who created this record',
    updated_by: 'User who last updated this record',
  };

  return descriptions[name.toLowerCase()] || '';
}

/**
 * Validates a value against a field configuration
 */
export function validateFieldValue(value: any, field: FieldConfig): string | null {
  if (value === null || value === undefined || value === '') {
    return field.required ? `${field.label} is required` : null;
  }

  switch (field.type) {
    case 'email':
      if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) {
        return 'Invalid email address';
      }
      break;
    case 'url':
      try {
        new URL(value);
      } catch {
        return 'Invalid URL';
      }
      break;
    case 'number':
      if (isNaN(Number(value))) {
        return 'Must be a number';
      }
      if (field.maxLength !== undefined && Number(value) > field.maxLength) {
        return `Value must be less than ${field.maxLength}`;
      }
      break;
    case 'json':
      if (typeof value === 'string') {
        try {
          JSON.parse(value);
        } catch {
          return 'Invalid JSON';
        }
      }
      break;
    case 'text':
    case 'textarea':
      if (field.maxLength && value.length > field.maxLength) {
        return `Maximum length is ${field.maxLength} characters`;
      }
      break;
  }

  return null;
}
