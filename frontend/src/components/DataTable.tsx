import { useState, useRef, useMemo } from 'react';
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  getFilteredRowModel,
  flexRender,
  type ColumnDef,
  type SortingState,
} from '@tanstack/react-table';
import { useVirtualizer } from '@tanstack/react-virtual';
import { Button } from './ui/Button';
import { Badge } from './ui/Badge';
import { Input } from './ui/Input';
import { Switch } from './ui/Switch';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from './ui/Select';
import { TableSchema, ColumnInfo, ForeignKeyInfo } from '@/types/schema';
import { formatColumnName, mapPostgresToFieldType } from '@/utils/fieldMapper';
import { getRecordIdentity } from '@/utils/recordIdentity';
import { cn } from '@/lib/utils';
import {
  ChevronUp,
  ChevronDown,
  ChevronsUpDown,
  Plus,
  Trash2,
  Edit,
  Loader2,
  Search,
  Database,
} from 'lucide-react';

interface DataTableProps {
  schema: TableSchema;
  data: Record<string, any>[];
  total: number;
  isLoading: boolean;
  page: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
  onSortChange: (sortBy: string, sortOrder: 'ASC' | 'DESC') => void;
  onCreate: () => void;
  onEdit: (record: Record<string, any>) => void;
  onDelete: (id: string) => void;
}

export function DataTable({
  schema,
  data,
  total,
  isLoading,
  page,
  pageSize,
  onPageChange,
  onPageSizeChange,
  onSortChange,
  onCreate,
  onEdit,
  onDelete,
}: DataTableProps) {
  const [sorting, setSorting] = useState<SortingState>([]);
  const [rowSelection, setRowSelection] = useState({});
  const [globalFilter, setGlobalFilter] = useState('');
  const parentRef = useRef<HTMLDivElement>(null);

  // Build column definitions from schema
  const columns = useMemo<ColumnDef<Record<string, any>>[]>(() => {
    const cols: ColumnDef<Record<string, any>>[] = [
      {
        id: 'actions',
        header: 'Actions',
        size: 96,
        cell: ({ row }) => (
          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={() => onEdit(row.original)}
              title="Edit record"
            >
              <Edit className="w-3.5 h-3.5" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={() => onDelete(getRecordIdentity(schema, row.original))}
              title="Delete record"
            >
              <Trash2 className="w-3.5 h-3.5 text-destructive" />
            </Button>
          </div>
        ),
      },
    ];

    for (const column of schema.columns) {
      cols.push({
        accessorKey: column.column_name,
        header: formatColumnName(column.column_name),
        size: getColumnWidth(column),
        cell: ({ getValue }) => {
          const value = getValue();
          return <CellRenderer column={column} value={value} foreignKeys={schema.foreign_keys || []} />;
        },
        enableSorting: true,
      });
    }

    return cols;
  }, [schema, onEdit, onDelete]);

  const handleSortingChange = (updater: any) => {
    const newSorting = typeof updater === 'function' ? updater(sorting) : updater;
    setSorting(newSorting);
    if (newSorting.length > 0) {
      const sort = newSorting[0];
      onSortChange(sort.id, (sort.desc ? 'DESC' : 'ASC') as 'ASC' | 'DESC');
    }
  };

  const table = useReactTable({
    data,
    columns,
    state: { sorting, rowSelection, globalFilter },
    onSortingChange: handleSortingChange,
    onGlobalFilterChange: setGlobalFilter,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    globalFilterFn: (row, _columnId, filterValue) => {
      const search = String(filterValue).toLowerCase();
      if (!search) return true;
      return Object.values(row.original).some(val => {
        if (val == null) return false;
        return String(val).toLowerCase().includes(search);
      });
    },
    manualPagination: true,
    rowCount: total,
    onRowSelectionChange: setRowSelection,
  });

  const { rows } = table.getRowModel();

  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 40,
    overscan: 5,
  });

  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  return (
    <div className="space-y-3">
      <div className="cms-surface flex flex-col gap-3 p-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex min-w-0 flex-1 items-center gap-2">
          <div className="relative w-full max-w-xl">
            <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search records..."
              value={globalFilter}
              onChange={(e) => setGlobalFilter(e.target.value)}
              className="pl-8"
            />
          </div>
        </div>

        <Button onClick={onCreate} size="sm" className="w-full sm:w-auto">
          <Plus className="w-4 h-4 mr-2" />
          Add Record
        </Button>
      </div>

      <div className="overflow-hidden rounded-md border bg-card shadow-sm">
        <div ref={parentRef} className="table-virtual" style={{ maxHeight: 'calc(100vh - 320px)', minHeight: '320px', overflow: 'auto' }}>
          <table style={{ width: table.getCenterTotalSize(), minWidth: '100%' }}>
            <thead>
              {table.getHeaderGroups().map(headerGroup => (
                <tr key={headerGroup.id} className="border-b bg-muted">
                  {headerGroup.headers.map(header => (
                    <th
                      key={header.id}
                      className="h-10 whitespace-nowrap px-3 text-left align-middle text-[11px] font-semibold uppercase tracking-[0.04em] text-muted-foreground [&:has([role=checkbox])]:pr-0"
                      style={{ width: header.getSize() }}
                    >
                      {header.isPlaceholder ? null : (
                        <div
                          className={cn(
                            'flex items-center gap-1',
                            header.column.getCanSort() && 'cursor-pointer select-none'
                          )}
                          onClick={header.column.getToggleSortingHandler()}
                        >
                          {flexRender(header.column.columnDef.header, header.getContext())}
                          {{
                            asc: <ChevronUp className="w-4 h-4" />,
                            desc: <ChevronDown className="w-4 h-4" />,
                          }[header.column.getIsSorted() as string] ?? (
                            header.column.getCanSort() && <ChevronsUpDown className="w-4 h-4 opacity-50" />
                          )}
                        </div>
                      )}
                    </th>
                  ))}
                </tr>
              ))}
            </thead>
            <tbody style={{ position: 'relative', height: `${virtualizer.getTotalSize()}px` }}>
              {isLoading ? (
                <tr>
                  <td colSpan={columns.length} className="h-24 text-center">
                    <div className="flex items-center justify-center gap-2">
                      <Loader2 className="w-4 h-4 animate-spin" />
                      Loading records...
                    </div>
                  </td>
                </tr>
              ) : virtualizer.getVirtualItems().length === 0 ? (
                <tr>
                  <td colSpan={columns.length} className="h-64 text-center text-muted-foreground">
                    <div className="flex flex-col items-center justify-center gap-3">
                      <div className="rounded-md bg-muted p-3">
                        <Database className="h-6 w-6" />
                      </div>
                      <div>
                        <p className="font-medium text-foreground">No records found.</p>
                        <p className="text-sm text-muted-foreground">Try clearing search or add a new record.</p>
                      </div>
                    </div>
                  </td>
                </tr>
              ) : (
                virtualizer.getVirtualItems().map(virtualRow => {
                  const row = rows[virtualRow.index];
                  return (
                    <tr
                      key={row.id}
                      data-index={virtualRow.index}
                      ref={node => virtualizer.measureElement(node)}
                      className={cn(
                    'absolute left-0 right-0 border-b transition-colors hover:bg-accent/45 data-[state=selected]:bg-muted',
                        (rowSelection as Record<string, boolean>)[row.id] && 'bg-muted'
                      )}
                      style={{
                        transform: `translateY(${virtualRow.start}px)`,
                      }}
                    >
                      {row.getVisibleCells().map(cell => (
                        <td
                          key={cell.id}
                          className="h-11 whitespace-nowrap px-3 py-2 align-middle text-sm [&:has([role=checkbox])]:pr-0"
                        >
                          {flexRender(cell.column.columnDef.cell, cell.getContext())}
                        </td>
                      ))}
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      </div>

      <div className="cms-surface flex flex-col gap-3 p-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
          <span>
            {total} total records
          </span>
          <Select
            value={String(pageSize)}
            onValueChange={(value) => onPageSizeChange(Number(value))}
          >
            <SelectTrigger className="w-[100px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {[10, 25, 50, 100].map(size => (
                <SelectItem key={size} value={String(size)}>
                  {size} / page
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="flex items-center justify-between gap-2 sm:justify-end">
          <Button
            variant="outline"
            size="sm"
            onClick={() => onPageChange(page - 1)}
            disabled={page <= 1}
          >
            Previous
          </Button>
          <span className="min-w-[86px] text-center text-sm">
            Page {page} of {totalPages}
          </span>
          <Button
            variant="outline"
            size="sm"
            onClick={() => onPageChange(page + 1)}
            disabled={page >= totalPages}
          >
            Next
          </Button>
        </div>
      </div>
    </div>
  );
}

interface CellRendererProps {
  column: ColumnInfo;
  value: any;
  foreignKeys: ForeignKeyInfo[];
}

function CellRenderer({ column, value, foreignKeys }: CellRendererProps) {
  if (value === null || value === undefined) {
    return <span className="text-muted-foreground">--</span>;
  }

  const fieldType = mapPostgresToFieldType(column, foreignKeys);

  switch (fieldType) {
    case 'boolean':
      return <Switch checked={!!value} disabled />;

    case 'select':
      return (
        <Badge variant="secondary" className="max-w-[170px] truncate rounded-md">
          {String(value).length > 20 ? String(value).slice(0, 20) + '...' : String(value)}
        </Badge>
      );

    case 'datetime':
    case 'date':
      try {
        return new Date(value).toLocaleDateString('en-US', {
          month: 'short',
          day: 'numeric',
          year: 'numeric',
        });
      } catch {
        return String(value);
      }

    case 'json':
      return (
        <Badge variant="outline" className="max-w-[150px] truncate rounded-md font-mono text-xs">
          JSON
        </Badge>
      );

    case 'image':
    case 'url':
      if (value && (value.startsWith('http') || value.startsWith('data:'))) {
        return (
          <a
            href={value}
            target="_blank"
            rel="noopener noreferrer"
            className="block max-w-[180px] truncate text-primary hover:underline"
          >
            {value.slice(0, 30)}...
          </a>
        );
      }
      return <span className="block max-w-[180px] truncate">{String(value)}</span>;

    default:
      return (
        <span className="block max-w-[240px] truncate" title={String(value)}>
          {String(value)}
        </span>
      );
  }
}

function getColumnWidth(column: ColumnInfo): number {
  if (column.is_primary_key) return 220;
  if (column.is_foreign_key) return 180;

  switch (column.udt_name) {
    case 'bool':
      return 96;
    case 'date':
    case 'timestamptz':
    case 'timestamp':
      return 160;
    case 'jsonb':
    case 'json':
      return 120;
    default:
      return 200;
  }
}
