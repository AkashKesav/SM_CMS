import * as React from 'react';
import { cn } from '@/lib/utils';
import { PanelLeftClose, PanelLeftOpen } from 'lucide-react';

interface SidebarContextValue {
  open: boolean;
  setOpen: (open: boolean) => void;
  isMobile: boolean;
}

const SidebarContext = React.createContext<SidebarContextValue | undefined>(undefined);

function useSidebar() {
  const context = React.useContext(SidebarContext);
  if (!context) {
    throw new Error('useSidebar must be used within SidebarProvider');
  }
  return context;
}

export function SidebarProvider({ children }: { children: React.ReactNode }) {
  const [isMobile, setIsMobile] = React.useState(false);
  const [open, setOpen] = React.useState(() =>
    typeof window === 'undefined' ? true : window.innerWidth >= 1024
  );

  React.useEffect(() => {
    const query = window.matchMedia('(max-width: 1023px)');
    const update = () => {
      setIsMobile(query.matches);
      setOpen(!query.matches);
    };

    update();
    query.addEventListener('change', update);
    return () => query.removeEventListener('change', update);
  }, []);

  return (
    <SidebarContext.Provider value={{ open, setOpen, isMobile }}>
      <div className="flex min-h-screen w-full">{children}</div>
    </SidebarContext.Provider>
  );
}

export function Sidebar({ children, className }: { children: React.ReactNode; className?: string }) {
  const { open } = useSidebar();

  return (
    <aside
      className={cn(
        'fixed inset-y-0 left-0 z-40 flex h-screen w-[17rem] flex-col border-r border-sidebar-border bg-sidebar shadow-xl transition-transform duration-300 lg:sticky lg:top-0 lg:z-0 lg:shadow-none',
        open ? 'translate-x-0' : '-translate-x-full lg:w-0 lg:overflow-hidden',
        className
      )}
    >
      {children}
    </aside>
  );
}

export function SidebarBackdrop() {
  const { open, setOpen, isMobile } = useSidebar();

  if (!open || !isMobile) return null;

  return (
    <button
      aria-label="Close sidebar"
      className="fixed inset-0 z-30 bg-foreground/20 backdrop-blur-sm lg:hidden"
      onClick={() => setOpen(false)}
    />
  );
}

export function SidebarHeader({ children, className }: { children: React.ReactNode; className?: string }) {
  return <div className={cn('flex-shrink-0', className)}>{children}</div>;
}

export function SidebarContent({ children, className }: { children: React.ReactNode; className?: string }) {
  return <div className={cn('min-h-0 flex-1 overflow-auto py-2', className)}>{children}</div>;
}

export function SidebarFooter({ children, className }: { children: React.ReactNode; className?: string }) {
  return <div className={cn('flex-shrink-0', className)}>{children}</div>;
}

export function SidebarGroup({ children, className }: { children: React.ReactNode; className?: string }) {
  return <div className={cn('px-3 py-2', className)}>{children}</div>;
}

export function SidebarGroupLabel({ children, className }: { children: React.ReactNode; className?: string }) {
  return (
    <div className={cn('px-2 py-1.5 text-[11px] font-semibold uppercase tracking-[0.08em] text-sidebar-foreground/50', className)}>
      {children}
    </div>
  );
}

export function SidebarGroupContent({ children, className }: { children: React.ReactNode; className?: string }) {
  return <div className={cn('', className)}>{children}</div>;
}

export function SidebarMenu({ children }: { children: React.ReactNode }) {
  return <div className="space-y-1">{children}</div>;
}

export function SidebarMenuItem({ children }: { children: React.ReactNode }) {
  return <div>{children}</div>;
}

export function SidebarMenuButton({
  children,
  isActive = false,
  onClick,
  className,
}: {
  children: React.ReactNode;
  isActive?: boolean;
  onClick?: () => void;
  className?: string;
}) {
  const { isMobile, setOpen } = useSidebar();

  const handleClick = () => {
    onClick?.();
    if (isMobile) setOpen(false);
  };

  return (
    <button
      className={cn(
        'flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm font-medium transition-colors [&_svg]:shrink-0 [&_span]:truncate',
        isActive
          ? 'bg-primary text-primary-foreground shadow-sm'
          : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
        className
      )}
      onClick={handleClick}
    >
      {children}
    </button>
  );
}

export function SidebarTrigger({ className }: { className?: string }) {
  const { open, setOpen } = useSidebar();
  const Icon = open ? PanelLeftClose : PanelLeftOpen;

  return (
    <button
      className={cn('rounded-md p-2 text-muted-foreground hover:bg-accent hover:text-foreground', className)}
      onClick={() => setOpen(!open)}
      title={open ? 'Collapse sidebar' : 'Expand sidebar'}
    >
      <Icon className="h-5 w-5" />
    </button>
  );
}
