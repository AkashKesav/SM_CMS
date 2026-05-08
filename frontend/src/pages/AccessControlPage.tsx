import { useCallback, useEffect, useMemo, useState } from 'react';
import { adminApi } from '@/lib/api';
import { AccessUser, ProvisionResult } from '@/types/schema';
import { Button } from '@/components/ui/Button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { Badge } from '@/components/ui/Badge';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/Select';
import { AlertTriangle, KeyRound, Loader2, RefreshCw, Search, ShieldCheck } from 'lucide-react';
import toast from 'react-hot-toast';

const DEFAULT_PASSWORD = 'user123';

export function AccessControlPage() {
  const [users, setUsers] = useState<AccessUser[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isProvisioning, setIsProvisioning] = useState(false);
  const [activeUserId, setActiveUserId] = useState<string | null>(null);
  const [query, setQuery] = useState('');
  const [password, setPassword] = useState(DEFAULT_PASSWORD);
  const [lastProvision, setLastProvision] = useState<ProvisionResult | null>(null);

  const loadUsers = useCallback(async () => {
    setIsLoading(true);
    const response = await adminApi.listUsers();
    if (response.success && response.data) {
      setUsers(response.data);
    } else {
      toast.error(response.error || 'Failed to load access users');
    }
    setIsLoading(false);
  }, []);

  useEffect(() => {
    loadUsers();
  }, [loadUsers]);

  const filteredUsers = useMemo(() => {
    const term = query.trim().toLowerCase();
    if (!term) return users;
    return users.filter((user) => (
      user.roll_no.toLowerCase().includes(term) ||
      user.name.toLowerCase().includes(term) ||
      user.login_email.toLowerCase().includes(term) ||
      user.role.toLowerCase().includes(term)
    ));
  }, [query, users]);

  const counts = useMemo(() => ({
    total: users.length,
    admins: users.filter((user) => user.role === 'admin').length,
    accounts: users.filter((user) => user.has_account).length,
  }), [users]);

  const provisionStudents = async () => {
    if (!window.confirm(`Provision all student accounts and set their password to "${password || DEFAULT_PASSWORD}"?`)) {
      return;
    }
    setIsProvisioning(true);
    const response = await adminApi.provisionStudents(password || DEFAULT_PASSWORD);
    if (response.success && response.data) {
      setLastProvision(response.data);
      toast.success(`Provisioned: ${response.data.created} created, ${response.data.updated} updated`);
      await loadUsers();
    } else {
      toast.error(response.error || response.details || 'Provisioning failed');
    }
    setIsProvisioning(false);
  };

  const changeRole = async (user: AccessUser, role: 'admin' | 'student') => {
    if (!user.user_id) {
      toast.error('Provision this student before changing their role');
      return;
    }
    setActiveUserId(user.user_id);
    const response = await adminApi.setUserRole(user.user_id, role);
    if (response.success) {
      toast.success(role === 'admin' ? 'Admin access granted' : 'Admin access removed');
      setUsers((prev) => prev.map((candidate) => (
        candidate.user_id === user.user_id ? { ...candidate, role } : candidate
      )));
    } else {
      toast.error(response.error || response.details || 'Failed to update role');
    }
    setActiveUserId(null);
  };

  const resetPassword = async (user: AccessUser) => {
    if (!user.user_id) {
      toast.error('Provision this student before resetting password');
      return;
    }
    setActiveUserId(user.user_id);
    const response = await adminApi.resetUserPassword(user.user_id, password || DEFAULT_PASSWORD);
    if (response.success) {
      toast.success(`Password reset to ${password || DEFAULT_PASSWORD}`);
    } else {
      toast.error(response.error || response.details || 'Failed to reset password');
    }
    setActiveUserId(null);
  };

  return (
    <div className="cms-page">
      <div className="cms-page-header md:flex-row md:items-center">
        <div>
          <div className="cms-eyebrow mb-2">
            <ShieldCheck className="h-3.5 w-3.5" />
            Access control
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">Student Accounts</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Initial admins are 23bsm006 and 23bsm007. Everyone else starts as student.
          </p>
        </div>
        <div className="flex flex-col gap-2 sm:flex-row">
          <Button variant="outline" onClick={loadUsers} disabled={isLoading || isProvisioning}>
            <RefreshCw className={`mr-2 h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
          <Button onClick={provisionStudents} disabled={isProvisioning}>
            {isProvisioning ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <KeyRound className="mr-2 h-4 w-4" />}
            Provision Students
          </Button>
        </div>
      </div>

      <div className="grid gap-4 lg:grid-cols-[1fr_320px]">
        <Card>
          <CardHeader className="gap-4">
            <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
              <div>
                <CardTitle>Users</CardTitle>
                <CardDescription>{filteredUsers.length} shown from {counts.total} students</CardDescription>
              </div>
              <div className="relative w-full lg:w-80">
                <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                <Input
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  placeholder="Search roll no, name, email"
                  className="pl-9"
                />
              </div>
            </div>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="flex items-center justify-center py-16">
                <Loader2 className="h-8 w-8 animate-spin text-primary" />
              </div>
            ) : (
              <div className="overflow-x-auto rounded-md border">
                <table className="w-full min-w-[860px] border-collapse text-sm">
                  <thead className="bg-muted/70 text-left text-xs uppercase tracking-[0.08em] text-muted-foreground">
                    <tr>
                      <th className="px-3 py-2 font-semibold">Student</th>
                      <th className="px-3 py-2 font-semibold">Login</th>
                      <th className="px-3 py-2 font-semibold">Role</th>
                      <th className="px-3 py-2 font-semibold">Account</th>
                      <th className="px-3 py-2 text-right font-semibold">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredUsers.map((user) => (
                      <tr key={user.student_id} className="border-t">
                        <td className="px-3 py-3">
                          <div className="font-medium">{user.name || 'Unnamed student'}</div>
                          <div className="text-xs text-muted-foreground">{user.roll_no}</div>
                        </td>
                        <td className="px-3 py-3">
                          <div className="font-mono text-xs">{user.login_email}</div>
                        </td>
                        <td className="px-3 py-3">
                          <Select
                            value={user.role}
                            onValueChange={(value) => changeRole(user, value as 'admin' | 'student')}
                            disabled={!user.user_id || activeUserId === user.user_id}
                          >
                            <SelectTrigger className="w-36">
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="student">Student</SelectItem>
                              <SelectItem value="admin">Admin</SelectItem>
                            </SelectContent>
                          </Select>
                        </td>
                        <td className="px-3 py-3">
                          <Badge variant={user.has_account ? 'default' : 'secondary'} className="rounded-md">
                            {user.has_account ? 'Active' : 'Not provisioned'}
                          </Badge>
                        </td>
                        <td className="px-3 py-3 text-right">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => resetPassword(user)}
                            disabled={!user.user_id || activeUserId === user.user_id}
                          >
                            <KeyRound className="mr-2 h-4 w-4" />
                            Reset
                          </Button>
                        </td>
                      </tr>
                    ))}
                    {filteredUsers.length === 0 && (
                      <tr>
                        <td colSpan={5} className="px-3 py-12 text-center text-sm text-muted-foreground">
                          No users found.
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            )}
          </CardContent>
        </Card>

        <div className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Provisioning</CardTitle>
              <CardDescription>Creates or refreshes student auth accounts.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-3 gap-2 text-center">
                <Metric label="Students" value={counts.total} />
                <Metric label="Accounts" value={counts.accounts} />
                <Metric label="Admins" value={counts.admins} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="default-password">Default password</Label>
                <Input
                  id="default-password"
                  type="text"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                />
              </div>
              <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-xs text-amber-800 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-200">
                <div className="mb-1 flex items-center gap-2 font-medium">
                  <AlertTriangle className="h-4 w-4" />
                  Bulk reset
                </div>
                Running provisioning resets existing provisioned student passwords to this value.
              </div>
            </CardContent>
          </Card>

          {lastProvision && (
            <Card>
              <CardHeader>
                <CardTitle>Last Run</CardTitle>
                <CardDescription>
                  {lastProvision.created} created, {lastProvision.updated} updated, {lastProvision.failed} failed
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="max-h-72 space-y-2 overflow-auto pr-1">
                  {lastProvision.users.filter((item) => item.status === 'failed').map((item) => (
                    <div key={`${item.roll_no}-${item.login_email}`} className="rounded-md border p-2 text-xs">
                      <div className="font-medium">{item.roll_no} | {item.login_email}</div>
                      <div className="mt-1 text-destructive">{item.message}</div>
                    </div>
                  ))}
                  {lastProvision.failed === 0 && (
                    <p className="text-sm text-muted-foreground">No failures in the last provisioning run.</p>
                  )}
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md border bg-muted/40 p-3">
      <div className="text-lg font-semibold">{value}</div>
      <div className="text-xs text-muted-foreground">{label}</div>
    </div>
  );
}
