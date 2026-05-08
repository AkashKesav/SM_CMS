import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { Badge } from '@/components/ui/Badge';
import { GraduationCap, ShieldCheck } from 'lucide-react';
import toast from 'react-hot-toast';

export function RegisterPage() {
  const navigate = useNavigate();
  const { register, isLoading, error } = useAuthStore();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [name, setName] = useState('');
  const [rollNo, setRollNo] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (password !== confirmPassword) {
      toast.error('Passwords do not match');
      return;
    }

    if (password.length < 6) {
      toast.error('Password must be at least 6 characters');
      return;
    }

    await register(email, password, name, rollNo);

    const authState = useAuthStore.getState();
    if (authState.isAuthenticated) {
      toast.success('Account created');
      navigate('/');
    } else if (!authState.error) {
      toast.success('Registration submitted');
      navigate('/login');
    } else {
      toast.error(authState.error);
    }
  };

  return (
    <div className="grid min-h-screen bg-background lg:grid-cols-[0.92fr_1.08fr]">
      <aside className="hidden bg-sidebar text-sidebar-foreground lg:flex lg:flex-col lg:justify-between">
        <div className="p-10">
          <div className="flex items-center gap-3">
            <div className="flex h-20 w-20 shrink-0 items-center justify-center p-1">
              <img src="/logo.png" alt="IIITDMJ Logo" className="h-full w-full object-contain drop-shadow-md" />
            </div>
            <div>
              <p className="text-sm font-semibold">SM CMS</p>
              <p className="text-xs text-sidebar-foreground/60">Student access</p>
            </div>
          </div>

          <div className="mt-20 max-w-md">
            <Badge variant="outline" className="rounded-md border-sidebar-border bg-sidebar-accent text-sidebar-foreground">
              <ShieldCheck className="mr-1.5 h-3.5 w-3.5" />
              Admin reviewed
            </Badge>
            <h1 className="mt-5 text-4xl font-semibold tracking-tight">Student records with approval built in.</h1>
            <div className="mt-8 grid gap-3">
              {['Roll number claim', 'Profile updates', 'Linked records'].map((item) => (
                <div key={item} className="rounded-md border border-sidebar-border bg-sidebar-accent/70 px-4 py-3 text-sm">
                  {item}
                </div>
              ))}
            </div>
          </div>
        </div>
        <div className="border-t border-sidebar-border p-10 text-xs text-sidebar-foreground/60">
          Requests become public after review
        </div>
      </aside>

      <main className="auth-grid flex min-h-screen items-center justify-center p-6">
        <div className="w-full max-w-md">
          <div className="mb-6 lg:hidden">
            <div className="flex items-center gap-3">
              <div className="flex h-16 w-16 shrink-0 items-center justify-center p-1">
                <img src="/logo.png" alt="IIITDMJ Logo" className="h-full w-full object-contain drop-shadow-md" />
              </div>
              <div>
                <p className="font-semibold">SM CMS</p>
                <p className="text-xs text-muted-foreground">Student access</p>
              </div>
            </div>
          </div>

          <div className="cms-surface bg-card/95 p-6">
            <div className="mb-6">
              <div className="mb-3 flex h-10 w-10 items-center justify-center rounded-md bg-primary/10 text-primary">
                <GraduationCap className="h-5 w-5" />
              </div>
              <h2 className="text-xl font-semibold">Create account</h2>
              <p className="mt-1 text-sm text-muted-foreground">Student account request.</p>
            </div>

            <form onSubmit={handleSubmit} className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2 sm:col-span-2">
                <Label htmlFor="name">Name</Label>
                <Input
                  id="name"
                  type="text"
                  placeholder="Full name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  autoComplete="name"
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="roll-no">Roll number</Label>
                <Input
                  id="roll-no"
                  type="text"
                  placeholder="Roll number"
                  value={rollNo}
                  onChange={(e) => setRollNo(e.target.value)}
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="email">Email</Label>
                <Input
                  id="email"
                  type="email"
                  placeholder="name@example.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  autoComplete="email"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="password">Password</Label>
                <Input
                  id="password"
                  type="password"
                  placeholder="Min 6 characters"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  autoComplete="new-password"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="confirm-password">Confirm</Label>
                <Input
                  id="confirm-password"
                  type="password"
                  placeholder="Repeat password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  required
                  autoComplete="new-password"
                />
              </div>

              {error && <p className="text-sm text-destructive sm:col-span-2">{error}</p>}

              <div className="sm:col-span-2">
                <Button type="submit" className="w-full" disabled={isLoading}>
                  {isLoading ? 'Creating account...' : 'Create Account'}
                </Button>
              </div>
            </form>

            <div className="mt-5 border-t pt-4 text-center text-sm">
              <span className="text-muted-foreground">Already registered? </span>
              <Link to="/login" className="font-medium text-primary hover:underline">
                Sign in
              </Link>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
