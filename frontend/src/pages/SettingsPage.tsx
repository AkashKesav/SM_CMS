import { useState, useEffect } from 'react';
import { useAuthStore } from '@/stores/authStore';
import { authApi } from '@/lib/api';
import { Button } from '@/components/ui/Button';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/Card';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/Select';
import { Sun, Moon, Monitor, UserRound, KeyRound } from 'lucide-react';
import toast from 'react-hot-toast';

type Theme = 'light' | 'dark' | 'system';

interface Settings {
  theme: Theme;
  language: string;
}

function getStoredSettings(): Settings {
  try {
    const stored = localStorage.getItem('cms-settings');
    if (stored) return JSON.parse(stored);
  } catch {}
  return { theme: 'system', language: 'en' };
}

function applyTheme(theme: Theme) {
  const root = document.documentElement;
  root.classList.remove('light', 'dark');

  if (theme === 'system') {
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    root.classList.add(prefersDark ? 'dark' : 'light');
  } else {
    root.classList.add(theme);
  }
}

function getBackendUrl() {
  return import.meta.env.VITE_BACKEND_URL || '';
}

export function SettingsPage() {
  const { user } = useAuthStore();
  const [settings, setSettings] = useState<Settings>(getStoredSettings);
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [isChangingPassword, setIsChangingPassword] = useState(false);

  useEffect(() => {
    applyTheme(settings.theme);
    localStorage.setItem('cms-settings', JSON.stringify(settings));
  }, [settings]);


  const handleThemeChange = (theme: Theme) => {
    setSettings((prev) => ({ ...prev, theme }));
    toast.success(`Theme set to ${theme}`);
  };

  const handlePasswordChange = async (event: React.FormEvent) => {
    event.preventDefault();
    if (newPassword.length < 6) {
      toast.error('Password must be at least 6 characters');
      return;
    }
    if (newPassword !== confirmPassword) {
      toast.error('Passwords do not match');
      return;
    }
    setIsChangingPassword(true);
    const response = await authApi.changePassword(newPassword);
    if (response.success) {
      toast.success('Password changed');
      setNewPassword('');
      setConfirmPassword('');
    } else {
      toast.error(response.error || response.details || 'Failed to change password');
    }
    setIsChangingPassword(false);
  };

  return (
    <div className="cms-page">
      <div className="cms-page-header">
        <div>
        <div className="cms-eyebrow mb-2">
          <Monitor className="h-3.5 w-3.5" />
          Workspace
        </div>
        <h2 className="text-2xl font-semibold tracking-tight">Settings</h2>
        <p className="mt-1 text-sm text-muted-foreground">Manage local preferences and connection status.</p>
        </div>
      </div>

      <div className="grid gap-4 xl:grid-cols-3">
        <Card className="xl:col-span-2">
          <CardHeader>
            <CardTitle>Appearance</CardTitle>
            <CardDescription>Local display preferences for this device.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-5">
            <div className="space-y-2">
              <Label>Theme</Label>
              <div className="flex flex-wrap gap-2">
                {([
                  { value: 'light' as Theme, icon: Sun, label: 'Light' },
                  { value: 'dark' as Theme, icon: Moon, label: 'Dark' },
                  { value: 'system' as Theme, icon: Monitor, label: 'System' },
                ]).map(({ value, icon: Icon, label }) => (
                  <Button
                    key={value}
                    variant={settings.theme === value ? 'default' : 'outline'}
                    size="sm"
                    onClick={() => handleThemeChange(value)}
                  >
                    <Icon className="mr-2 h-4 w-4" />
                    {label}
                  </Button>
                ))}
              </div>
            </div>

            <div className="space-y-2">
              <Label>Language</Label>
              <Select
                value={settings.language}
                onValueChange={(value) => setSettings((prev) => ({ ...prev, language: value }))}
              >
                <SelectTrigger className="w-full sm:w-[220px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="en">English</SelectItem>
                  <SelectItem value="es">Espanol</SelectItem>
                  <SelectItem value="fr">Francais</SelectItem>
                  <SelectItem value="de">Deutsch</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <UserRound className="h-5 w-5 text-primary" />
              Account
            </CardTitle>
            <CardDescription>Current signed-in user.</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-3">
            <div className="rounded-md border bg-muted/40 p-3">
              <p className="text-xs text-muted-foreground">Email</p>
              <p className="truncate text-sm font-medium">{user?.email || 'N/A'}</p>
            </div>
            <div className="rounded-md border bg-muted/40 p-3">
              <p className="text-xs text-muted-foreground">Role</p>
              <p className="text-sm font-medium capitalize">{user?.role || 'N/A'}</p>
            </div>
            <div className="rounded-md border bg-muted/40 p-3">
              <p className="text-xs text-muted-foreground">Name</p>
              <p className="truncate text-sm font-medium">{user?.name || 'Not set'}</p>
            </div>
          </CardContent>
        </Card>

        <Card className="xl:col-span-2">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <KeyRound className="h-5 w-5 text-primary" />
              Password
            </CardTitle>
            <CardDescription>Update the password for the current account.</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handlePasswordChange} className="grid gap-4 md:grid-cols-[1fr_1fr_auto] md:items-end">
              <div className="space-y-2">
                <Label htmlFor="new-password">New password</Label>
                <Input
                  id="new-password"
                  type="password"
                  value={newPassword}
                  onChange={(event) => setNewPassword(event.target.value)}
                  autoComplete="new-password"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="confirm-password">Confirm password</Label>
                <Input
                  id="confirm-password"
                  type="password"
                  value={confirmPassword}
                  onChange={(event) => setConfirmPassword(event.target.value)}
                  autoComplete="new-password"
                />
              </div>
              <Button type="submit" disabled={isChangingPassword}>
                {isChangingPassword ? 'Saving...' : 'Change'}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
