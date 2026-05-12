import { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { ArrowLeft, Lock, CheckCircle2, AlertCircle } from 'lucide-react';
import { authApi } from '@/lib/api';
import toast from 'react-hot-toast';

export function ResetPasswordPage() {
  const navigate = useNavigate();
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);
  const [accessToken, setAccessToken] = useState('');
  const [refreshToken, setRefreshToken] = useState('');
  const [tokenError, setTokenError] = useState('');

  // Extract token from the URL query string
  // Our custom flow redirects to: ?token=...
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const token = params.get('token');

    if (!token) {
      // Also check hash just in case of old Supabase link
      const hash = window.location.hash.substring(1);
      if (hash) {
        setTokenError('This is an old Supabase reset link. Please request a new one.');
        return;
      }
      setTokenError('No recovery token found. Please use the link from your email.');
      return;
    }

    setAccessToken(token);
    // Refresh token is not used in our custom JWT flow
    setRefreshToken('');
  }, []);

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

    setIsLoading(true);
    const response = await authApi.resetPassword(accessToken, refreshToken, password);
    setIsLoading(false);

    if (response.success) {
      setIsSuccess(true);
      // Clear the hash from URL for security
      window.history.replaceState(null, '', window.location.pathname);
    } else {
      toast.error(response.error || 'Failed to reset password. The link may have expired.');
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
            Set your new<br />
            <span className="text-muted-foreground">password.</span>
          </h1>
          <p className="mt-6 text-lg text-muted-foreground leading-relaxed">
            Choose a strong password to keep your account secure.
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
          <span>Password Recovery</span>
        </div>
      </aside>

      {/* Right Panel */}
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
            {tokenError ? (
              /* Error State - Invalid/missing token */
              <div className="text-center">
                <div className="mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-full bg-destructive/10">
                  <AlertCircle className="h-8 w-8 text-destructive" />
                </div>
                <h2 className="font-display text-2xl font-bold tracking-tight">Invalid Reset Link</h2>
                <p className="mt-3 text-sm text-muted-foreground leading-relaxed">
                  {tokenError}
                </p>
                <div className="mt-8 space-y-3">
                  <Link to="/forgot-password">
                    <Button className="w-full">Request New Reset Link</Button>
                  </Link>
                  <Link to="/login">
                    <Button variant="outline" className="w-full">
                      <ArrowLeft className="mr-2 h-4 w-4" />
                      Back to Sign In
                    </Button>
                  </Link>
                </div>
              </div>
            ) : isSuccess ? (
              /* Success State */
              <div className="text-center">
                <div className="mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-full bg-emerald-500/10">
                  <CheckCircle2 className="h-8 w-8 text-emerald-500" />
                </div>
                <h2 className="font-display text-2xl font-bold tracking-tight">Password Reset</h2>
                <p className="mt-3 text-sm text-muted-foreground leading-relaxed">
                  Your password has been successfully updated. You can now sign in with your new password.
                </p>
                <div className="mt-8">
                  <Button className="w-full" onClick={() => navigate('/login')}>
                    Sign In
                  </Button>
                </div>
              </div>
            ) : (
              /* Form State */
              <>
                <div className="mb-8">
                  <h2 className="font-display text-2xl font-bold tracking-tight">New Password</h2>
                  <p className="mt-2 text-sm text-muted-foreground">
                    Enter your new password below. It must be at least 6 characters long.
                  </p>
                </div>

                <form onSubmit={handleSubmit} className="space-y-5">
                  <div className="space-y-2">
                    <Label htmlFor="password" className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                      New Password
                    </Label>
                    <div className="relative">
                      <Lock className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                      <Input
                        id="password"
                        type="password"
                        placeholder="••••••••"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        required
                        minLength={6}
                        autoComplete="new-password"
                        className="h-11 bg-background/50 pl-10 focus:bg-background"
                      />
                    </div>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="confirm-password" className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                      Confirm New Password
                    </Label>
                    <div className="relative">
                      <Lock className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                      <Input
                        id="confirm-password"
                        type="password"
                        placeholder="••••••••"
                        value={confirmPassword}
                        onChange={(e) => setConfirmPassword(e.target.value)}
                        required
                        minLength={6}
                        autoComplete="new-password"
                        className="h-11 bg-background/50 pl-10 focus:bg-background"
                      />
                    </div>
                    {password && confirmPassword && password !== confirmPassword && (
                      <p className="text-xs text-destructive">Passwords do not match</p>
                    )}
                  </div>

                  <Button type="submit" className="w-full h-11 text-base mt-2" disabled={isLoading || !password || !confirmPassword}>
                    {isLoading ? 'Resetting...' : 'Reset Password'}
                  </Button>
                </form>

                <div className="mt-8 text-center">
                  <Link to="/login" className="inline-flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors">
                    <ArrowLeft className="h-4 w-4" />
                    Back to Sign In
                  </Link>
                </div>
              </>
            )}
          </div>
        </div>
      </main>
    </div>
  );
}
