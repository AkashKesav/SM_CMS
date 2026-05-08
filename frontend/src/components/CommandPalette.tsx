import { useEffect, useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { CommandDialog, CommandInput, CommandList, CommandEmpty, CommandGroup, CommandItem } from './ui/Command';
import { useSchemaStore } from '@/stores/schemaStore';
import { formatColumnName } from '@/utils/fieldMapper';
import { 
  Table2, 
  Home, 
  RefreshCw,
  Moon,
  Sun,
} from 'lucide-react';

export function CommandPalette() {
  const [open, setOpen] = useState(false);
  const navigate = useNavigate();
  const { getTables, refreshSchema } = useSchemaStore();
  const tables = getTables();
  const [isDark, setIsDark] = useState(false);

  // Toggle command palette with Cmd/Ctrl+K
  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === 'k' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen((open) => !open);
      }
    };

    document.addEventListener('keydown', down);
    return () => document.removeEventListener('keydown', down);
  }, []);

  // Detect system theme
  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    setIsDark(mediaQuery.matches);
    
    const handleChange = (e: MediaQueryListEvent) => {
      setIsDark(e.matches);
    };
    
    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, []);

  const toggleTheme = () => {
    document.documentElement.classList.toggle('dark');
    setIsDark(!isDark);
  };

  const tableItems = useMemo(() => 
    tables.map(table => ({
      id: table.table_name,
      label: formatColumnName(table.table_name),
      action: () => navigate(`/tables/${table.table_name}`),
    })),
    [tables, navigate]
  );

  const actions = [
    { id: 'dashboard', label: 'Go to Dashboard', icon: Home, action: () => navigate('/') },
    { id: 'refresh', label: 'Refresh Schema', icon: RefreshCw, action: () => refreshSchema() },
    { id: 'theme', label: isDark ? 'Switch to Light Mode' : 'Switch to Dark Mode', icon: isDark ? Sun : Moon, action: toggleTheme },
  ];

  return (
    <CommandDialog open={open} onOpenChange={setOpen}>
      <CommandInput placeholder="Type a command or search..." />
      <CommandList>
        <CommandEmpty>No results found.</CommandEmpty>

        <CommandGroup heading="Actions">
          {actions.map(action => (
            <CommandItem
              key={action.id}
              onSelect={() => {
                action.action();
                setOpen(false);
              }}
            >
              <action.icon className="w-4 h-4 mr-2" />
              {action.label}
            </CommandItem>
          ))}
        </CommandGroup>

        <CommandGroup heading="Tables">
          {tableItems.map(item => (
            <CommandItem
              key={item.id}
              onSelect={() => {
                item.action();
                setOpen(false);
              }}
            >
              <Table2 className="w-4 h-4 mr-2" />
              {item.label}
            </CommandItem>
          ))}
        </CommandGroup>
      </CommandList>
    </CommandDialog>
  );
}
