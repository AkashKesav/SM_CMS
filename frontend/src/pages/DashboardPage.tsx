import { useNavigate } from 'react-router-dom';
import { useSchemaStore } from '@/stores/schemaStore';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { formatColumnName } from '@/utils/fieldMapper';
import {
  Table2,
  Columns3,
  Link,
  RefreshCw,
  ArrowRight,
  Database,
} from 'lucide-react';

export function DashboardPage() {
  const navigate = useNavigate();
  const { getTables, refreshSchema, isLoading } = useSchemaStore();
  const tables = getTables();

  const totalColumns = tables.reduce((sum, table) => sum + (table.columns?.length || 0), 0);
  const totalRelations = tables.reduce((sum, table) => sum + (table.foreign_keys?.length || 0), 0);
  const relationalTables = tables.filter((table) => (table.foreign_keys?.length || 0) > 0).length;

  const stats = [
    {
      label: 'Tables',
      value: tables.length,
      detail: 'Discovered models',
      icon: Table2,
      surface: 'bg-blue-50 text-blue-700 dark:bg-blue-950 dark:text-blue-300',
    },
    {
      label: 'Columns',
      value: totalColumns,
      detail: 'Editable fields',
      icon: Columns3,
      surface: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300',
    },
    {
      label: 'Relations',
      value: totalRelations,
      detail: `${relationalTables} linked tables`,
      icon: Link,
      surface: 'bg-violet-50 text-violet-700 dark:bg-violet-950 dark:text-violet-300',
    },
  ];

  return (
    <div className="cms-page">
      <div className="cms-page-header">
        <div className="min-w-0">
          <div className="cms-eyebrow mb-2">
            <Database className="h-3.5 w-3.5" />
            Schema workspace
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">Dashboard</h1>
          <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
            {tables.length} tables discovered from your database with {totalColumns} fields and {totalRelations} relationships.
          </p>
        </div>
        <Button
          variant="outline"
          onClick={() => refreshSchema()}
          disabled={isLoading}
          className="w-full sm:w-auto"
        >
          <RefreshCw className={`mr-2 h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} />
          Refresh Schema
        </Button>
      </div>

      <div className="grid gap-6 md:grid-cols-3">
        {stats.map((stat, i) => (
          <Card key={stat.label} className="group overflow-hidden animate-spring-in" style={{ animationDelay: `${i * 50}ms` }}>
            <CardHeader className="flex flex-row items-start justify-between space-y-0 pb-2">
              <div>
                <CardDescription className="uppercase tracking-wider text-xs font-semibold">{stat.label}</CardDescription>
                <CardTitle className="mt-2 text-4xl font-display font-bold tracking-tight">{stat.value}</CardTitle>
              </div>
              <div className={`rounded-xl p-2.5 transition-transform duration-300 group-hover:scale-110 ${stat.surface}`}>
                <stat.icon className="h-5 w-5" />
              </div>
            </CardHeader>
            <CardContent className="pt-0">
              <p className="text-sm text-muted-foreground font-medium">{stat.detail}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="space-y-6 pt-4 animate-spring-in" style={{ animationDelay: '200ms' }}>
        <div className="flex items-center justify-between">
          <div>
            <h2 className="font-display text-xl font-semibold tracking-tight">Active Models</h2>
            <p className="text-sm text-muted-foreground mt-1">Select a table to manage its records and relationships.</p>
          </div>
        </div>

        <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {tables.map((table) => (
            <Card
              key={table.table_name}
              className="group hover-lift cursor-pointer border-transparent bg-card shadow-card hover:border-primary/20 hover:ring-1 hover:ring-primary/20"
              onClick={() => navigate(`/tables/${table.table_name}`)}
            >
              <CardHeader className="pb-3">
                <div className="flex items-start justify-between gap-3">
                  <div className="flex items-center gap-3">
                    <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary transition-colors group-hover:bg-primary group-hover:text-primary-foreground">
                      <Table2 className="h-4 w-4" />
                    </div>
                    <CardTitle className="truncate font-display text-base">
                      {formatColumnName(table.table_name)}
                    </CardTitle>
                  </div>
                  <ArrowRight className="h-4 w-4 shrink-0 text-muted-foreground/40 transition-all duration-300 group-hover:-rotate-45 group-hover:text-primary" />
                </div>
              </CardHeader>
              <CardContent>
                <div className="flex items-center gap-4 text-xs font-medium text-muted-foreground">
                  <span className="flex items-center gap-1.5">
                    <Columns3 className="h-3.5 w-3.5" />
                    {table.columns?.length || 0} fields
                  </span>
                  <span className="flex items-center gap-1.5">
                    <Link className="h-3.5 w-3.5" />
                    {table.foreign_keys?.length || 0} refs
                  </span>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>

        {tables.length === 0 && !isLoading && (
          <div className="flex flex-col items-center justify-center py-24 text-center animate-fade-in">
            <div className="flex h-20 w-20 items-center justify-center rounded-3xl bg-muted/50 mb-6">
              <Database className="h-8 w-8 text-muted-foreground/50" />
            </div>
            <h3 className="font-display text-xl font-semibold tracking-tight">No Schema Detected</h3>
            <p className="mt-2 max-w-sm text-sm text-muted-foreground leading-relaxed">
              We couldn't detect any user tables in the connected database. Run a schema sync to discover your models.
            </p>
            <Button onClick={() => refreshSchema()} className="mt-8 rounded-full h-11 px-6">
              <RefreshCw className="mr-2 h-4 w-4" />
              Discover Models
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
