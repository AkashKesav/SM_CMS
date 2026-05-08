import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { DatabaseSchema, TableSchema, TableViewConfig } from '@/types/schema';
import { schemaApi, recordsApi } from '@/lib/api';

interface SchemaState {
  schema: DatabaseSchema | null;
  isLoading: boolean;
  error: string | null;
  lastSync: string | null;

  // Table data cache
  tableData: Record<string, {
    records: Record<string, any>[];
    total: number;
    page: number;
    pageSize: number;
    isLoading: boolean;
  }>;

  // View configurations per table
  viewConfigs: Record<string, TableViewConfig>;

  // Actions
  loadSchema: () => Promise<void>;
  refreshSchema: () => Promise<void>;
  getTableSchema: (tableName: string) => TableSchema | undefined;
  getTables: () => TableSchema[];
  getTableNames: () => string[];
  
  // Table data actions
  loadTableData: (tableName: string, options?: {
    page?: number;
    pageSize?: number;
    sortBy?: string;
    sortOrder?: 'ASC' | 'DESC';
    filters?: Record<string, string>;
  }) => Promise<void>;
  updateViewConfig: (tableName: string, config: Partial<TableViewConfig>) => void;
  clearTableCache: (tableName?: string) => void;
  clearCache: () => void;
}

export const useSchemaStore = create<SchemaState>()(
  persist(
    (set, get) => ({
      schema: null,
      isLoading: false,
      error: null,
      lastSync: null,
      tableData: {},
      viewConfigs: {},

      loadSchema: async () => {
        // Load from cache first
        const cached = get().schema;
        const cacheSupportsPrimaryKeys = cached?.tables.every(table =>
          Array.isArray(table.primary_keys)
        );
        if (cached && cacheSupportsPrimaryKeys) {
          set({ isLoading: false });
          return;
        }

        set({ isLoading: true, error: null });
        try {
          const response = await schemaApi.getSchema();
          if (response.success && response.data) {
            set({
              schema: response.data,
              isLoading: false,
              lastSync: new Date().toISOString(),
            });
          } else {
            set({ isLoading: false, error: response.error || 'Failed to load schema' });
          }
        } catch (error: any) {
          set({ isLoading: false, error: error.message || 'Failed to load schema' });
        }
      },

      refreshSchema: async () => {
        set({ isLoading: true, error: null });
        try {
          const response = await schemaApi.refreshSchema();
          if (response.success && response.data) {
            set({
              schema: response.data,
              isLoading: false,
              lastSync: new Date().toISOString(),
            });
          } else {
            set({ isLoading: false, error: response.error || 'Failed to refresh schema' });
          }
        } catch (error: any) {
          set({ isLoading: false, error: error.message || 'Failed to refresh schema' });
        }
      },

      getTableSchema: (tableName: string) => {
        return get().schema?.tables.find(t => t.table_name === tableName);
      },

      getTables: () => {
        return get().schema?.tables || [];
      },

      getTableNames: () => {
        return get().schema?.tables.map(t => t.table_name) || [];
      },

      loadTableData: async (tableName: string, options = {}) => {
        const {
          page = 1,
          pageSize = 50,
          sortBy = '',
          sortOrder = 'DESC',
          filters = {},
        } = options;

        // Set loading state for this table
        set(state => ({
          tableData: {
            ...state.tableData,
            [tableName]: {
              ...state.tableData[tableName],
              isLoading: true,
            },
          },
        }));

        try {
          const response = await recordsApi.list(tableName, {
            page,
            page_size: pageSize,
            sort_by: sortBy,
            sort_order: sortOrder,
            ...filters,
          });

          if (response.success && response.data) {
            set(state => ({
              tableData: {
                ...state.tableData,
                [tableName]: {
                  records: response.data as Record<string, any>[],
                  total: response.meta?.total || 0,
                  page,
                  pageSize,
                  isLoading: false,
                },
              },
            }));
          } else {
            set(state => ({
              tableData: {
                ...state.tableData,
                [tableName]: {
                  ...state.tableData[tableName],
                  isLoading: false,
                },
              },
              error: response.error || 'Failed to load data',
            }));
          }
        } catch (error: any) {
          set(state => ({
            tableData: {
              ...state.tableData,
              [tableName]: {
                ...state.tableData[tableName],
                isLoading: false,
              },
            },
            error: error.message || 'Failed to load data',
          }));
        }
      },

      updateViewConfig: (tableName: string, config: Partial<TableViewConfig>) => {
        set(state => ({
          viewConfigs: {
            ...state.viewConfigs,
            [tableName]: {
              ...state.viewConfigs[tableName],
              ...config,
            },
          },
        }));
      },

      clearTableCache: (tableName?: string) => {
        if (tableName) {
          set(state => {
            const newData = { ...state.tableData };
            delete newData[tableName];
            return {
              tableData: newData as Record<string, {
                records: Record<string, any>[];
                total: number;
                page: number;
                pageSize: number;
                isLoading: boolean;
              }>,
            };
          });
        } else {
          set({ tableData: {} });
        }
      },

      clearCache: () => {
        set({
          schema: null,
          tableData: {},
          viewConfigs: {},
          lastSync: null,
          error: null,
        });
      },
    }),
    {
      name: 'schema-storage',
      partialize: (state) => ({
        schema: state.schema,
        lastSync: state.lastSync,
        viewConfigs: state.viewConfigs,
      }),
    }
  )
);
