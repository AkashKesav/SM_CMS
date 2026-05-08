import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { 
  SidebarProvider,
  Sidebar,
  SidebarHeader,
  SidebarContent,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
  SidebarTrigger,
  SidebarFooter,
  SidebarBackdrop,
} from './ui/Sidebar';
import { Button } from './ui/Button';
import { Avatar, AvatarFallback } from './ui/Avatar';
import { Badge } from './ui/Badge';
import { useSchemaStore } from '@/stores/schemaStore';
import { useAuthStore } from '@/stores/authStore';
import { formatColumnName } from '@/utils/fieldMapper';
import { 
  Database, 
  RefreshCw, 
  LogOut, 
  Settings, 
  Home,
  Table2,
  Activity,
  ClipboardCheck,
  UserRound,
  UsersRound,
} from 'lucide-react';
import toast from 'react-hot-toast';

export function Layout() {
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuthStore();
  const { getTables, refreshSchema, isLoading } = useSchemaStore();
  const tables = getTables();
  const isAdmin = user?.role === 'admin';

  const handleRefreshSchema = async () => {
    try {
      await refreshSchema();
      toast.success('Schema refreshed successfully');
    } catch (error: any) {
      toast.error(error.message || 'Failed to refresh schema');
    }
  };

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  return (
    <SidebarProvider>
      <div className="flex min-h-screen w-full bg-background">
        <SidebarBackdrop />
        <Sidebar>
          <SidebarHeader className="border-b border-sidebar-border px-4 py-4">
            <div className="flex items-center gap-2">
              <div className="flex h-14 w-14 items-center justify-center p-1 shrink-0">
                <img src="/logo.png" alt="IIITDMJ Logo" className="h-full w-full object-contain" />
              </div>
              <div className="min-w-0">
                <span className="block truncate font-semibold text-sidebar-foreground">SM CMS</span>
                <span className="block text-xs text-sidebar-foreground/60">
                  {isAdmin ? `${tables.length} tables indexed` : 'Internal Portal'}
                </span>
              </div>
            </div>
          </SidebarHeader>

          <SidebarContent>
            <SidebarGroup>
              <SidebarGroupLabel>Navigation</SidebarGroupLabel>
              <SidebarGroupContent>
                <SidebarMenu>
                  <SidebarMenuItem>
                    <SidebarMenuButton
                      isActive={location.pathname === '/'}
                      onClick={() => navigate('/')}
                    >
                      {isAdmin ? <Home className="w-4 h-4" /> : <UserRound className="w-4 h-4" />}
                      <span>{isAdmin ? 'Dashboard' : 'My Info'}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                  {isAdmin && (
                    <SidebarMenuItem>
                      <SidebarMenuButton
                        isActive={location.pathname === '/access'}
                        onClick={() => navigate('/access')}
                      >
                        <UsersRound className="w-4 h-4" />
                        <span>Access Control</span>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  )}
                  {isAdmin && (
                    <SidebarMenuItem>
                      <SidebarMenuButton
                        isActive={location.pathname === '/review'}
                        onClick={() => navigate('/review')}
                      >
                        <ClipboardCheck className="w-4 h-4" />
                        <span>Review Queue</span>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  )}
                  <SidebarMenuItem>
                    <SidebarMenuButton
                      isActive={location.pathname === '/settings'}
                      onClick={() => navigate('/settings')}
                    >
                      <Settings className="w-4 h-4" />
                      <span>Settings</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>

            {isAdmin && (
            <SidebarGroup className="pt-1">
              <SidebarGroupLabel className="flex items-center justify-between">
                <span>Tables</span>
                <span className="rounded-md bg-sidebar-accent px-1.5 py-0.5 text-[11px] text-sidebar-foreground/80">
                  {tables.length}
                </span>
              </SidebarGroupLabel>
              <SidebarGroupContent>
                <SidebarMenu>
                  {tables.map((table) => (
                    <SidebarMenuItem key={table.table_name}>
                      <SidebarMenuButton
                        isActive={location.pathname === `/tables/${table.table_name}`}
                        onClick={() => navigate(`/tables/${table.table_name}`)}
                      >
                        <Table2 className="w-4 h-4" />
                        <span>{formatColumnName(table.table_name)}</span>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  ))}
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>
            )}
          </SidebarContent>

          <SidebarFooter className="border-t border-sidebar-border p-4">
            <div className="flex items-center gap-3">
              <Avatar className="h-9 w-9 border border-sidebar-border bg-sidebar-accent">
                <AvatarFallback>
                  {user?.email?.charAt(0).toUpperCase() || 'U'}
                </AvatarFallback>
              </Avatar>
              <div className="flex-1 min-w-0">
                <p className="truncate text-sm font-medium text-sidebar-foreground">{user?.email}</p>
                <p className="text-xs capitalize text-sidebar-foreground/60">{user?.role}</p>
              </div>
              <Button
                variant="ghost"
                size="icon"
                onClick={handleLogout}
                title="Logout"
                className="text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-foreground"
              >
                <LogOut className="w-4 h-4" />
              </Button>
            </div>

            {isAdmin && (
            <div className="flex gap-2 mt-3">
              <Button
                variant="outline"
                size="sm"
                className="flex-1 border-sidebar-border bg-sidebar-accent text-sidebar-foreground hover:bg-sidebar-accent/80"
                onClick={handleRefreshSchema}
                disabled={isLoading}
              >
                <RefreshCw className={`w-4 h-4 mr-1 ${isLoading ? 'animate-spin' : ''}`} />
                Sync
              </Button>
              <Button
                variant="outline"
                size="icon"
                title="Settings"
                onClick={() => navigate('/settings')}
                className="border-sidebar-border bg-sidebar-accent text-sidebar-foreground hover:bg-sidebar-accent/80"
              >
                <Settings className="w-4 h-4" />
              </Button>
            </div>
            )}
          </SidebarFooter>
        </Sidebar>

        <main className="min-w-0 flex-1 overflow-auto bg-background/50">
          <div className="sticky top-0 z-20 flex h-16 items-center justify-between gap-4 border-b border-border/40 bg-background/80 px-4 backdrop-blur-xl lg:px-8">
            <div className="flex min-w-0 items-center gap-4">
              <SidebarTrigger className="text-muted-foreground hover:text-foreground transition-colors" />
              <div className="min-w-0">
                <h1 className="truncate font-display text-lg font-semibold tracking-tight text-foreground">
                  {location.pathname === '/'
                    ? (isAdmin ? 'Overview' : 'My Information')
                    : location.pathname === '/review'
                      ? 'Review Queue'
                    : location.pathname === '/access'
                      ? 'Access Control'
                    : formatColumnName(location.pathname.split('/').pop() || '')}
                </h1>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <span className="hidden items-center gap-2 text-xs font-medium text-muted-foreground sm:flex">
                <span className="relative flex h-2 w-2">
                  <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-40"></span>
                  <span className="relative inline-flex h-2 w-2 rounded-full bg-emerald-500"></span>
                </span>
                System Operational
              </span>
            </div>
          </div>
          <div className="p-4 lg:p-6">
            <Outlet />
          </div>
        </main>
      </div>
    </SidebarProvider>
  );
}
