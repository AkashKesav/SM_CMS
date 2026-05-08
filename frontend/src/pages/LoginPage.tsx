import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { ArrowRight } from 'lucide-react';
import toast from 'react-hot-toast';

export function LoginPage() {
  const navigate = useNavigate();
  const { login, isLoading, error } = useAuthStore();
  const [identifier, setIdentifier] = useState('');
  const [password, setPassword] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await login(identifier, password);

    const authState = useAuthStore.getState();
    if (authState.isAuthenticated) {
      toast.success('Signed in');
      navigate('/');
    } else if (authState.error) {
      toast.error(authState.error);
    }
  };

  return (
    <div className="grid min-h-screen bg-background lg:grid-cols-[1fr_1.2fr]">
      {/* Left Editorial Panel */}
      <aside className="hidden flex-col justify-between border-r border-border/40 bg-card p-12 lg:flex">
        <div>
          <div className="flex items-center gap-3">
            <div className="flex h-20 w-20 shrink-0 items-center justify-center p-1">
              <img src="/logo.png" alt="IIITDMJ Logo" className="h-full w-full object-contain drop-shadow-md" />
            </div>
            <div>
              <p className="font-display text-lg font-bold tracking-tight text-foreground">SM CMS</p>
              <p className="font-display text-xs font-semibold uppercase tracking-widest text-muted-foreground/80">Internal Portal</p>
            </div>
          </div>
        </div>

        <div className="max-w-md animate-spring-in">
          <h1 className="font-display text-5xl font-bold tracking-tight text-foreground leading-[1.1]">
            Curate your profile,<br />
            <span className="text-muted-foreground">effortlessly.</span>
          </h1>
          <p className="mt-6 text-lg text-muted-foreground leading-relaxed">
            The internal portal to manage, update, and submit changes for your official SM website profile.
          </p>
        </div>

        <div className="flex items-center gap-4 text-sm font-medium text-muted-foreground">
          <span className="flex items-center gap-2">
            <span className="relative flex h-2 w-2">
              <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
              <span className="relative inline-flex h-2 w-2 rounded-full bg-emerald-500"></span>
            </span>
            System Operational
          </span>
          <span className="text-border">•</span>
          <span>Internal Access Only</span>
        </div>
      </aside>

      {/* Right Login Panel */}
      <main className="auth-grid flex min-h-screen flex-col justify-center p-6 sm:p-12">
        <div className="mx-auto w-full max-w-md animate-spring-in" style={{ animationDelay: '100ms' }}>
          {/* Mobile Header */}
          <div className="mb-10 lg:hidden">
            <div className="flex items-center gap-3">
              <div className="flex h-16 w-16 shrink-0 items-center justify-center p-1">
                <img src="/logo.png" alt="IIITDMJ Logo" className="h-full w-full object-contain drop-shadow-md" />
              </div>
              <div>
                <p className="font-display text-lg font-bold">SM CMS</p>
              </div>
            </div>
          </div>

          <div className="glass cms-surface p-8 sm:p-10">
            <div className="mb-8">
              <h2 className="font-display text-2xl font-bold tracking-tight">Sign In</h2>
              <p className="mt-2 text-sm text-muted-foreground">Enter your credentials to access the workspace.</p>
            </div>

            <form onSubmit={handleSubmit} className="space-y-5">
              <div className="space-y-2">
                <Label htmlFor="identifier" className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                  Email or Roll Number
                </Label>
                <Input
                  id="identifier"
                  type="text"
                  placeholder="name@example.com"
                  value={identifier}
                  onChange={(e) => setIdentifier(e.target.value)}
                  required
                  autoComplete="username"
                  className="h-11 bg-background/50 focus:bg-background"
                />
              </div>

              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <Label htmlFor="password" className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                    Password
                  </Label>
                </div>
                <Input
                  id="password"
                  type="password"
                  placeholder="••••••••"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  autoComplete="current-password"
                  className="h-11 bg-background/50 focus:bg-background"
                />
              </div>

              {error && (
                <div className="rounded-lg bg-destructive/10 p-3 text-sm text-destructive">
                  {error}
                </div>
              )}

              <Button type="submit" className="group w-full h-11 text-base mt-2" disabled={isLoading}>
                {isLoading ? 'Authenticating...' : (
                  <>
                    Sign In
                    <ArrowRight className="ml-2 h-4 w-4 transition-transform group-hover:translate-x-1" />
                  </>
                )}
              </Button>
            </form>

            <div className="mt-8 text-center">
              <p className="text-sm text-muted-foreground">
                Don't have an account?{' '}
                <Link to="/register" className="font-medium text-primary hover:text-primary/80 transition-colors">
                  Create one now
                </Link>
              </p>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
