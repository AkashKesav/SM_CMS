import { useEffect, useState, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import { useSchemaStore } from '@/stores/schemaStore';
import { DataTable } from '@/components/DataTable';
import { RecordForm } from '@/components/RecordForm';
import { RelatedRecords } from '@/components/RelatedRecords';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/Dialog';
import { Button } from '@/components/ui/Button';
import { createFieldConfig } from '@/utils/fieldMapper';
import { recordsApi } from '@/lib/api';
import { formatColumnName } from '@/utils/fieldMapper';
import { getRecordIdentity } from '@/utils/recordIdentity';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/Tabs';
import { Table2, Loader2, Plus } from 'lucide-react';
import toast from 'react-hot-toast';

export function TablePage() {
  const { tableName } = useParams<{ tableName: string }>();
  const { getTableSchema, loadTableData, tableData } = useSchemaStore();
  const schema = tableName ? getTableSchema(tableName) : undefined;

  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [sortBy, setSortBy] = useState('');
  const [sortOrder, setSortOrder] = useState<'ASC' | 'DESC'>('DESC');
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<Record<string, any> | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [viewMode, setViewMode] = useState<'table' | 'detail'>('table');

  const tableDataState = tableName ? tableData[tableName] : undefined;

  useEffect(() => {
    if (tableName && schema) {
      loadTableData(tableName, {
        page,
        pageSize,
        sortBy,
        sortOrder,
      });
    }
  }, [tableName, schema, page, pageSize, sortBy, sortOrder, loadTableData]);

  const handleSortChange = useCallback((newSortBy: string, newSortOrder: 'ASC' | 'DESC') => {
    setSortBy(newSortBy);
    setSortOrder(newSortOrder);
    setPage(1);
  }, []);

  const handleCreate = async (data: Record<string, any>) => {
    if (!tableName) return;
    setIsSubmitting(true);
    try {
      const response = await recordsApi.create(tableName, data);
      if (response.success) {
        toast.success('Record created successfully');
        setIsCreateOpen(false);
        loadTableData(tableName, { page, pageSize, sortBy, sortOrder });
      } else {
        toast.error(response.error || 'Failed to create record');
      }
    } catch (error: any) {
      toast.error(error.message || 'Failed to create record');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleUpdate = async (data: Record<string, any>) => {
    if (!tableName || !editingRecord || !schema) return;
    setIsSubmitting(true);
    try {
      const id = getRecordIdentity(schema, editingRecord);
      const response = await recordsApi.update(tableName, id, data);
      if (response.success) {
        toast.success('Record updated successfully');
        setIsEditOpen(false);
        setEditingRecord(null);
        loadTableData(tableName, { page, pageSize, sortBy, sortOrder });
      } else {
        toast.error(response.error || 'Failed to update record');
      }
    } catch (error: any) {
      toast.error(error.message || 'Failed to update record');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!tableName) return;
    if (!confirm('Are you sure you want to delete this record?')) return;

    try {
      const response = await recordsApi.delete(tableName, id, true);
      if (response.success) {
        toast.success('Record deleted successfully');
        loadTableData(tableName, { page, pageSize, sortBy, sortOrder });
      } else {
        toast.error(response.error || 'Failed to delete record');
      }
    } catch (error: any) {
      toast.error(error.message || 'Failed to delete record');
    }
  };

  const handleEdit = (record: Record<string, any>) => {
    setEditingRecord(record);
    setIsEditOpen(true);
  };

  if (!schema) {
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <Loader2 className="mb-4 h-8 w-8 animate-spin text-primary" />
        <p className="text-muted-foreground">Loading table schema...</p>
      </div>
    );
  }

  const fieldConfigs = schema.columns.map((column) =>
    createFieldConfig(column, schema.foreign_keys || [])
  );

  return (
    <div className="cms-page">
      <div className="cms-page-header lg:flex-row lg:items-center">
        <div className="flex min-w-0 items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-md bg-primary/10 text-primary">
            <Table2 className="h-5 w-5" />
          </div>
          <div className="min-w-0">
            <h1 className="truncate text-2xl font-semibold tracking-tight">{formatColumnName(schema.table_name)}</h1>
            <p className="text-muted-foreground">
              {schema.columns.length} columns | {tableDataState?.total || 0} records
            </p>
          </div>
        </div>

        <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
          <Tabs value={viewMode} onValueChange={(value) => setViewMode(value as 'table' | 'detail')}>
            <TabsList>
              <TabsTrigger value="table">Table View</TabsTrigger>
              <TabsTrigger value="detail">Detail View</TabsTrigger>
            </TabsList>
          </Tabs>
        </div>
      </div>

      {viewMode === 'table' && tableDataState && (
        <DataTable
          schema={schema}
          data={tableDataState.records || []}
          total={tableDataState.total}
          isLoading={tableDataState.isLoading}
          page={page}
          pageSize={pageSize}
          onPageChange={setPage}
          onPageSizeChange={(size) => {
            setPageSize(size);
            setPage(1);
          }}
          onSortChange={handleSortChange}
          onCreate={() => setIsCreateOpen(true)}
          onEdit={handleEdit}
          onDelete={handleDelete}
        />
      )}

      {viewMode === 'detail' && tableDataState && (
        <div className="space-y-4">
          <div className="flex justify-end">
            <Button onClick={() => setIsCreateOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              Add Record
            </Button>
          </div>
          {tableDataState.isLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-6 w-6 animate-spin" />
            </div>
          ) : (
            <div className="grid gap-4">
              {(tableDataState.records || []).map((record, index) => (
                <div
                  key={index}
                  className="cursor-pointer rounded-md border bg-card p-4 shadow-sm transition-shadow hover:shadow-md"
                  onClick={() => handleEdit(record)}
                >
                  <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
                    {schema.columns.slice(0, 4).map((column) => (
                      <div key={column.column_name}>
                        <p className="text-xs text-muted-foreground">
                          {formatColumnName(column.column_name)}
                        </p>
                        <p className="truncate text-sm font-medium">
                          {record[column.column_name] !== null && record[column.column_name] !== undefined
                            ? String(record[column.column_name]).slice(0, 50)
                            : '--'}
                        </p>
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
        <DialogContent className="max-h-[90vh] max-w-3xl overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Create New Record</DialogTitle>
            <DialogDescription>
              Fill in the fields below to create a new record in {formatColumnName(schema.table_name)}.
            </DialogDescription>
          </DialogHeader>
          <RecordForm
            tableName={tableName}
            fields={fieldConfigs}
            onSave={handleCreate}
            onCancel={() => setIsCreateOpen(false)}
            isLoading={isSubmitting}
          />
        </DialogContent>
      </Dialog>

      <Dialog open={isEditOpen} onOpenChange={setIsEditOpen}>
        <DialogContent className="max-h-[90vh] max-w-3xl overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Edit Record</DialogTitle>
            <DialogDescription>
              Update the fields below to modify this record.
            </DialogDescription>
          </DialogHeader>
          {editingRecord && (
            <Tabs defaultValue="fields" className="mt-4">
              <TabsList className="grid w-full grid-cols-2">
                <TabsTrigger value="fields">Details</TabsTrigger>
                <TabsTrigger value="related">Contributors & Links</TabsTrigger>
              </TabsList>
              
              <TabsContent value="fields" className="mt-4">
                <RecordForm
                  tableName={tableName}
                  fields={fieldConfigs}
                  initialData={editingRecord}
                  onSave={handleUpdate}
                  onCancel={() => {
                    setIsEditOpen(false);
                    setEditingRecord(null);
                  }}
                  isLoading={isSubmitting}
                  isEdit
                />
              </TabsContent>
              
              <TabsContent value="related" className="mt-4">
                <RelatedRecords 
                  sourceTableName={schema.table_name} 
                  sourceRecordId={getRecordIdentity(schema, editingRecord)} 
                />
              </TabsContent>
            </Tabs>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
