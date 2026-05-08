import { useState, useEffect } from 'react';
import { useSchemaStore } from '@/stores/schemaStore';
import { recordsApi } from '@/lib/api';
import { Button } from './ui/Button';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from './ui/Select';
import { Loader2, Plus, X, Link as LinkIcon, RefreshCw } from 'lucide-react';
import toast from 'react-hot-toast';
import { formatColumnName } from '@/utils/fieldMapper';
import { cn } from '@/lib/utils';

interface RelatedRecordsProps {
  sourceTableName: string;
  sourceRecordId: string;
}

export function RelatedRecords({ sourceTableName, sourceRecordId }: RelatedRecordsProps) {
  const { getTables, refreshSchema, isLoading: isSyncing } = useSchemaStore();
  const tables = getTables();
  
  // Find tables that have a foreign key pointing to the source table
  const relatedTables = tables.filter(t => 
    t.foreign_keys?.some(fk => fk.foreign_table_name.toLowerCase() === sourceTableName.toLowerCase())
  );

  if (relatedTables.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center border border-dashed rounded-xl bg-muted/50">
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted mb-4">
          <LinkIcon className="h-6 w-6 text-muted-foreground/40" />
        </div>
        <p className="text-sm font-medium text-foreground">No relationships discovered yet.</p>
        <p className="text-xs text-muted-foreground max-w-xs mt-1 mb-6">
          If you recently added the <code className="font-mono bg-muted px-1 rounded">achievement_members</code> or <code className="font-mono bg-muted px-1 rounded">project_members</code> tables, you need to sync the schema.
        </p>
        <Button 
          variant="outline" 
          size="sm" 
          onClick={() => refreshSchema()} 
          disabled={isSyncing}
          className="rounded-full px-6"
        >
          <RefreshCw className={cn("h-3.5 w-3.5 mr-2", isSyncing && "animate-spin")} />
          {isSyncing ? 'Syncing...' : 'Sync Database Schema'}
        </Button>
      </div>
    );
  }

  return (
    <div className="mt-4 space-y-6">
      <div className="space-y-1">
        <h3 className="font-display text-lg font-semibold tracking-tight text-foreground flex items-center gap-2">
          <LinkIcon className="h-4 w-4 text-primary" />
          Associated Records
        </h3>
        <p className="text-sm text-muted-foreground">Manage multi-select relationships and linked contributors.</p>
      </div>
      
      <div className="grid gap-6">
        {relatedTables.map(table => {
          const fk = table.foreign_keys!.find(f => f.foreign_table_name === sourceTableName)!;
          return (
            <RelatedTableManager
              key={table.table_name}
              tableName={table.table_name}
              fkColumn={fk.column_name}
              sourceId={sourceRecordId}
            />
          );
        })}
      </div>
    </div>
  );
}

interface RelatedTableManagerProps {
  tableName: string;
  fkColumn: string;
  sourceId: string;
}

function RelatedTableManager({ tableName, fkColumn, sourceId }: RelatedTableManagerProps) {
  const { getTableSchema } = useSchemaStore();
  const schema = getTableSchema(tableName);
  
  const [records, setRecords] = useState<any[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isAdding, setIsAdding] = useState(false);
  const [selectedFks, setSelectedFks] = useState<string[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  
  // For junction tables, find the "other" foreign key
  const otherFk = schema?.foreign_keys?.find(fk => fk.column_name !== fkColumn);
  const [otherOptions, setOtherOptions] = useState<{label: string, value: string}[]>([]);

  const fetchRecords = async () => {
    setIsLoading(true);
    try {
      const response = await recordsApi.list(tableName, { page: 1, page_size: 500 });
      if (response.success && response.data) {
        const filtered = (response.data as any[]).filter(r => String(r[fkColumn]) === String(sourceId));
        setRecords(filtered);
      }
    } catch (e) {
      console.error(e);
    } finally {
      setIsLoading(false);
    }
  };

  const fetchOtherOptions = async () => {
    if (!otherFk) return;
    try {
      const response = await recordsApi.list(otherFk.foreign_table_name, { page: 1, page_size: 500 });
      if (response.success && response.data) {
        const opts = (response.data as any[]).map(r => ({
          label: r.name || r.title || r.roll_no || r.email || r[otherFk.foreign_column_name],
          value: String(r[otherFk.foreign_column_name])
        }));
        setOtherOptions(opts);
      }
    } catch (e) {
      console.error(e);
    }
  };

  useEffect(() => {
    fetchRecords();
    if (otherFk) fetchOtherOptions();
  }, [tableName, sourceId]);

  const handleAddMultiple = async () => {
    if (!otherFk || selectedFks.length === 0) return;
    
    setIsSubmitting(true);
    // We run the creations in parallel
    try {
      const promises = selectedFks.map(fkValue => 
        recordsApi.create(tableName, {
          [fkColumn]: sourceId,
          [otherFk.column_name]: fkValue
        })
      );
      
      await Promise.all(promises);
      toast.success(`Added ${selectedFks.length} record(s)`);
      setSelectedFks([]);
      setIsAdding(false);
      setSearchQuery('');
      fetchRecords();
    } catch (e: any) {
      toast.error('Some records failed to add');
      fetchRecords();
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleRemove = async (id: string) => {
    try {
      const res = await recordsApi.delete(tableName, id, true);
      if (res.success) {
        toast.success('Removed successfully');
        fetchRecords();
      } else {
        toast.error('Failed to remove');
      }
    } catch (e: any) {
      toast.error(e.message || 'Error');
    }
  };

  if (!schema) return null;

  return (
    <div className="rounded-xl border bg-card p-4 shadow-sm">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h4 className="font-semibold text-sm">{formatColumnName(tableName)}</h4>
          <p className="text-xs text-muted-foreground">Manage associated records</p>
        </div>
        {!isAdding && otherFk && (
          <Button size="sm" variant="outline" onClick={() => setIsAdding(true)}>
            <Plus className="w-4 h-4 mr-1" /> Add
          </Button>
        )}
      </div>

      {isAdding && otherFk && (
        <div className="mb-4 p-4 bg-muted/30 rounded-xl border space-y-4 animate-in fade-in zoom-in-95 duration-200">
          <div className="flex items-center justify-between">
            <h5 className="text-sm font-semibold">Select {formatColumnName(otherFk.foreign_table_name)}</h5>
            <div className="space-x-2">
              <Button size="sm" onClick={handleAddMultiple} disabled={selectedFks.length === 0 || isSubmitting}>
                {isSubmitting ? <Loader2 className="w-3 h-3 animate-spin mr-1" /> : null}
                Save ({selectedFks.length})
              </Button>
              <Button size="sm" variant="ghost" onClick={() => { setIsAdding(false); setSelectedFks([]); setSearchQuery(''); }} disabled={isSubmitting}>
                Cancel
              </Button>
            </div>
          </div>
          
          <input 
            type="text"
            placeholder={`Search by name, roll no, etc...`}
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
            disabled={isSubmitting}
          />

          <div className="max-h-60 overflow-y-auto space-y-1 rounded-md border bg-background p-2 shadow-inner">
            {otherOptions
              .filter(opt => !records.some(r => String(r[otherFk.column_name]) === opt.value))
              .filter(opt => opt.label.toLowerCase().includes(searchQuery.toLowerCase()))
              .map(opt => (
              <div 
                key={opt.value} 
                onClick={() => {
                  if (isSubmitting) return;
                  if (selectedFks.includes(opt.value)) {
                    setSelectedFks(prev => prev.filter(v => v !== opt.value));
                  } else {
                    setSelectedFks(prev => [...prev, opt.value]);
                  }
                }}
                className={`flex items-center gap-3 p-2 hover:bg-muted/50 rounded-lg cursor-pointer transition-colors ${isSubmitting ? 'opacity-50 cursor-not-allowed' : ''}`}
              >
                <div className={`w-4 h-4 rounded border flex items-center justify-center transition-colors ${selectedFks.includes(opt.value) ? 'bg-primary border-primary text-primary-foreground' : 'border-input'}`}>
                  {selectedFks.includes(opt.value) && (
                    <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                    </svg>
                  )}
                </div>
                <span className="text-sm">{opt.label}</span>
              </div>
            ))}
            
            {otherOptions.filter(opt => !records.some(r => String(r[otherFk.column_name]) === opt.value)).length === 0 && (
              <p className="text-xs text-muted-foreground p-2 text-center">All available records have been added.</p>
            )}
          </div>
        </div>
      )}

      {isLoading ? (
        <div className="flex justify-center p-4"><Loader2 className="w-5 h-5 animate-spin text-muted-foreground" /></div>
      ) : records.length === 0 ? (
        <div className="text-center p-6 text-sm text-muted-foreground border border-dashed rounded-xl bg-muted/10">
          No records associated yet.
        </div>
      ) : (
        <div className="space-y-2">
          {records.map(record => {
            // Find a way to display the record
            // If it's a junction table, we want to display the "other" field
            let displayValue = 'Record';
            let recordId = record.id; // Assume id exists for deletion

            if (otherFk && record[otherFk.column_name]) {
              const opt = otherOptions.find(o => o.value === String(record[otherFk.column_name]));
              if (opt) displayValue = opt.label;
              else displayValue = String(record[otherFk.column_name]);
            } else {
              displayValue = record.name || record.title || record.id || 'Item';
            }

            // Junction tables might use composite primary keys. If there's no `id`, we might not be able to delete easily
            // using the generic delete which expects `id`. The generic api requires `id`.
            // Note: If the junction table lacks an `id` column, deletion might fail via the generic API unless adjusted.

            return (
              <div key={record.id || JSON.stringify(record)} className="flex items-center justify-between p-2 text-sm border rounded-md bg-background">
                <span>{displayValue}</span>
                {recordId && (
                  <Button size="sm" variant="ghost" className="h-8 w-8 p-0 text-destructive hover:bg-destructive/10 hover:text-destructive" onClick={() => handleRemove(recordId)}>
                    <X className="w-4 h-4" />
                  </Button>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
