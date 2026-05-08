import React, { useState, useCallback, useMemo, useEffect } from 'react';
import { Button } from './ui/Button';
import { Badge } from './ui/Badge';
import { Switch } from './ui/Switch';
import { Input } from './ui/Input';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from './ui/Select';
import { Textarea } from './ui/Textarea';
import { FieldConfig } from '@/types/schema';
import { formatColumnName, validateFieldValue } from '@/utils/fieldMapper';
import { recordsApi, storageApi } from '@/lib/api';
import { X, Plus, Save, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';

interface RecordFormProps {
  tableName?: string;
  fields: FieldConfig[];
  initialData?: Record<string, any>;
  onSave: (data: Record<string, any>) => Promise<void>;
  onCancel: () => void;
  isLoading?: boolean;
  isEdit?: boolean;
}

export function RecordForm({
  tableName,
  fields,
  initialData = {},
  onSave,
  onCancel,
  isLoading = false,
  isEdit = false,
}: RecordFormProps) {
  const [formData, setFormData] = useState<Record<string, any>>({ ...initialData });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [fkOptions, setFkOptions] = useState<Record<string, { label: string; value: any }[]>>({});
  const [fkLoading, setFkLoading] = useState<Record<string, boolean>>({});

  // Filter out auto-managed fields for create
  const editableFields = useMemo(() => {
    return fields.filter(field => {
      // Hide student_id for achievements/projects as they use junction tables now
      if (field.name === 'student_id' && (tableName === 'achievements' || tableName === 'projects')) {
        return false;
      }

      if (!isEdit && field.type === 'uuid' && field.name === 'id') {
        return false;
      }
      if (!isEdit && (field.name === 'created_at' || field.name === 'updated_at')) {
        return false;
      }
      return true;
    });
  }, [fields, isEdit]);

  // Fetch foreign key options
  useEffect(() => {
    const fkFields = fields.filter(f => f.foreignKey);
    if (fkFields.length === 0) return;

    fkFields.forEach(async (field) => {
      const fk = field.foreignKey!;
      setFkLoading(prev => ({ ...prev, [field.name]: true }));

      try {
        const response = await recordsApi.list(fk.foreign_table_name, {
          page: 1,
          page_size: 100,
        });

        if (response.success && response.data) {
          const options = (response.data as Record<string, any>[]).map(record => {
            const pkValue = record[fk.foreign_column_name];
            // Try to find a display name field
            const displayValue =
              record.name || record.title || record.label ||
              record.email || record.slug ||
              String(pkValue);

            return {
              label: String(displayValue),
              value: pkValue,
            };
          });

          setFkOptions(prev => ({ ...prev, [field.name]: options }));
        }
      } catch (err) {
        console.error(`Failed to fetch FK options for ${field.name}:`, err);
      } finally {
        setFkLoading(prev => ({ ...prev, [field.name]: false }));
      }
    });
  }, [fields]);

  const handleChange = useCallback((name: string, value: any) => {
    setFormData(prev => ({ ...prev, [name]: value }));
    setErrors(prev => {
      const next = { ...prev };
      delete next[name];
      return next;
    });
  }, []);

  const validate = useCallback(() => {
    const newErrors: Record<string, string> = {};
    let isValid = true;

    for (const field of editableFields) {
      const error = validateFieldValue(formData[field.name], field);
      if (error) {
        newErrors[field.name] = error;
        isValid = false;
      }
    }

    setErrors(newErrors);
    return isValid;
  }, [formData, editableFields]);

  const handleSubmit = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();
    if (!validate()) return;

    const cleanedData: Record<string, any> = {};
    for (const field of editableFields) {
      const value = formData[field.name];
      if (value === '' && field.nullable) {
        cleanedData[field.name] = null;
      } else if (value !== undefined && value !== '') {
        cleanedData[field.name] = value;
      }
    }

    await onSave(cleanedData);
  }, [formData, editableFields, validate, onSave]);

  // Cmd/Ctrl+S to save
  React.useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 's') {
        e.preventDefault();
        handleSubmit(e as any);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [handleSubmit]);

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {editableFields.map((field) => (
          <div
            key={field.name}
            className={cn(
              "space-y-2",
              field.type === 'textarea' || field.type === 'json' ? 'md:col-span-2' : ''
            )}
          >
            <label className="text-sm font-medium flex items-center gap-2">
              {formatColumnName(field.name)}
              {field.required && <span className="text-destructive">*</span>}
              {field.foreignKey && (
                <Badge variant="secondary" className="text-xs">
                  FK → {field.foreignKey.foreign_table_name}
                </Badge>
              )}
            </label>

            <FieldRenderer
              field={field}
              value={formData[field.name]}
              onChange={(value) => handleChange(field.name, value)}
              error={errors[field.name]}
              fkOptions={fkOptions[field.name]}
              fkLoading={fkLoading[field.name]}
            />

            {errors[field.name] && (
              <p className="text-sm text-destructive">{errors[field.name]}</p>
            )}
            {field.description && !errors[field.name] && (
              <p className="text-xs text-muted-foreground">{field.description}</p>
            )}
          </div>
        ))}
      </div>

      <div className="flex justify-end gap-2 pt-4 border-t">
        <Button type="button" variant="outline" onClick={onCancel} disabled={isLoading}>
          <X className="w-4 h-4 mr-2" />
          Cancel
        </Button>
        <Button type="submit" disabled={isLoading}>
          {isLoading ? (
            <>
              <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              Saving...
            </>
          ) : (
            <>
              <Save className="w-4 h-4 mr-2" />
              {isEdit ? 'Update' : 'Create'}
            </>
          )}
        </Button>
      </div>
    </form>
  );
}

interface FieldRendererProps {
  field: FieldConfig;
  value: any;
  onChange: (value: any) => void;
  error?: string;
  fkOptions?: { label: string; value: any }[];
  fkLoading?: boolean;
}

function FieldRenderer({
  field,
  value,
  onChange,
  error,
  fkOptions,
  fkLoading,
}: FieldRendererProps) {
  const hasError = !!error;

  switch (field.type) {
    case 'boolean':
      return (
        <div className="flex items-center gap-2 py-2">
          <Switch
            checked={!!value}
            onCheckedChange={onChange}
          />
          <span className="text-sm text-muted-foreground">
            {value ? 'Yes' : 'No'}
          </span>
        </div>
      );

    case 'select':
      return (
        <Select
          key={`select-${field.name}-${fkLoading ? 'loading' : 'loaded'}`}
          value={value != null ? String(value) : ''}
          onValueChange={(v) => {
            const selected = fkOptions?.find(opt => String(opt.value) === v);
            onChange(selected ? selected.value : v);
          }}
        >
          <SelectTrigger className={hasError ? 'border-destructive' : ''}>
            <SelectValue placeholder={
              fkLoading ? 'Loading options...' :
              fkOptions?.length ? `Select ${field.label.toLowerCase()}` :
              'No options available'
            } />
          </SelectTrigger>
          <SelectContent>
            {fkOptions && fkOptions.length > 0 ? (
              fkOptions
                .filter(opt => opt.value != null && opt.value !== '')
                .map((opt) => (
                  <SelectItem key={String(opt.value)} value={String(opt.value)}>
                    {opt.label}
                  </SelectItem>
                ))
            ) : (
              <SelectItem value="__none__" disabled>
                {fkLoading ? 'Loading...' : 'No options available'}
              </SelectItem>
            )}
          </SelectContent>
        </Select>
      );

    case 'textarea':
      return (
        <Textarea
          value={value || ''}
          onChange={(e) => onChange(e.target.value)}
          placeholder={field.placeholder}
          rows={4}
          className={cn(hasError && 'border-destructive')}
        />
      );

    case 'json':
      return (
        <JsonEditor
          value={value}
          onChange={onChange}
          placeholder={field.placeholder}
        />
      );

    case 'date':
      return (
        <Input
          type="date"
          value={value ? new Date(value).toISOString().split('T')[0] : ''}
          onChange={(e) => onChange(e.target.value)}
          className={cn(hasError && 'border-destructive')}
        />
      );

    case 'datetime':
      return (
        <Input
          type="datetime-local"
          value={value ? new Date(value).toISOString().slice(0, 16) : ''}
          onChange={(e) => onChange(e.target.value)}
          className={cn(hasError && 'border-destructive')}
        />
      );

    case 'number':
      return (
        <Input
          type="number"
          value={value ?? ''}
          onChange={(e) => onChange(e.target.value === '' ? '' : Number(e.target.value))}
          placeholder={field.placeholder}
          className={cn(hasError && 'border-destructive')}
        />
      );

    case 'email':
      return (
        <Input
          type="email"
          value={value || ''}
          onChange={(e) => onChange(e.target.value)}
          placeholder={field.placeholder}
          className={cn(hasError && 'border-destructive')}
        />
      );

    case 'url':
      return (
        <Input
          type="url"
          value={value || ''}
          onChange={(e) => onChange(e.target.value)}
          placeholder={field.placeholder}
          className={cn(hasError && 'border-destructive')}
        />
      );

    case 'image':
      return (
        <ImageUploader
          value={value}
          onChange={onChange}
          placeholder={field.placeholder}
        />
      );

    case 'uuid':
      return (
        <Input
          type="text"
          value={value || 'auto-generated'}
          disabled
          className="bg-muted"
        />
      );

    default: // text
      return (
        <Input
          type="text"
          value={value || ''}
          onChange={(e) => onChange(e.target.value)}
          placeholder={field.placeholder}
          maxLength={field.maxLength || undefined}
          className={cn(hasError && 'border-destructive')}
        />
      );
  }
}

interface JsonEditorProps {
  value: any;
  onChange: (value: any) => void;
  placeholder?: string;
}

function JsonEditor({ value, onChange, placeholder }: JsonEditorProps) {
  const [text, setText] = useState(() => {
    if (typeof value === 'object' && value !== null) {
      return JSON.stringify(value, null, 2);
    }
    return value || '';
  });
  const [isExpanded, setIsExpanded] = useState(false);
  const [parseError, setParseError] = useState<string | null>(null);

  const handleBlur = () => {
    try {
      if (text.trim() === '') {
        onChange(null);
        setParseError(null);
      } else {
        const parsed = JSON.parse(text);
        onChange(parsed);
        setParseError(null);
      }
    } catch {
      setParseError('Invalid JSON');
    }
  };

  return (
    <div className="relative">
      <div className="flex items-center justify-between mb-1">
        <button
          type="button"
          className="text-xs text-muted-foreground hover:text-foreground"
          onClick={() => setIsExpanded(!isExpanded)}
        >
          {isExpanded ? 'Collapse' : 'Expand'} JSON
        </button>
        {parseError && (
          <span className="text-xs text-destructive">{parseError}</span>
        )}
      </div>
      <Textarea
        value={text}
        onChange={(e) => setText(e.target.value)}
        onBlur={handleBlur}
        placeholder={placeholder}
        rows={isExpanded ? 8 : 3}
        className="font-mono text-xs"
      />
    </div>
  );
}

interface ImageUploaderProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
}

function ImageUploader({ value, onChange, placeholder }: ImageUploaderProps) {
  const [isUploading, setIsUploading] = useState(false);

  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setIsUploading(true);
    try {
      // Get signed URL from backend
      const bucket = import.meta.env.VITE_SUPABASE_STORAGE_BUCKET || 'cms-media';
      const signedUrlResp = await storageApi.getSignedUrl(bucket, file.name, file.type);

      if (signedUrlResp.success && signedUrlResp.data?.signed_url) {
        // Upload to Supabase Storage
        const publicUrl = await storageApi.uploadToSupabase(signedUrlResp.data.signed_url, file);
        onChange(publicUrl);
      } else {
        // Fallback: create local preview if no Supabase configured
        const url = URL.createObjectURL(file);
        onChange(url);
      }
    } catch (error) {
      console.error('Upload failed:', error);
      // Fallback to local preview
      const url = URL.createObjectURL(file);
      onChange(url);
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <div className="space-y-2">
      {value ? (
        <div className="relative">
          <img
            src={value}
            alt="Preview"
            className="w-full h-32 object-cover rounded-md"
            onError={(e) => {
              (e.target as HTMLImageElement).style.display = 'none';
            }}
          />
          <button
            type="button"
            className="absolute top-1 right-1 p-1 bg-destructive text-destructive-foreground rounded-full"
            onClick={() => onChange('')}
          >
            <X className="w-3 h-3" />
          </button>
        </div>
      ) : (
        <label className="flex flex-col items-center justify-center w-full h-24 border-2 border-dashed rounded-md cursor-pointer hover:bg-accent">
          <div className="flex flex-col items-center justify-center py-2">
            {isUploading ? (
              <Loader2 className="w-6 h-6 text-muted-foreground animate-spin" />
            ) : (
              <Plus className="w-6 h-6 text-muted-foreground" />
            )}
            <p className="text-xs text-muted-foreground mt-1">
              {isUploading ? 'Uploading...' : 'Click to upload image'}
            </p>
          </div>
          <input
            type="file"
            accept="image/*"
            className="hidden"
            onChange={handleFileSelect}
            disabled={isUploading}
          />
        </label>
      )}
      <Input
        type="url"
        value={value || ''}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder || 'Or paste image URL'}
        className="text-sm"
      />
    </div>
  );
}
