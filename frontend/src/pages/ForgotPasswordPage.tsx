import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { ArrowLeft, Mail, CheckCircle2 } from 'lucide-react';
import { authApi } from '@/lib/api';
import toast from 'react-hot-toast';

export function ForgotPasswordPage() {
  const [email, setEmail] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isSent, setIsSent] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!email.trim()) {
      toast.error('Please enter your email or roll number');
      return;
    }

    setIsLoading(true);
    const response = await authApi.forgotPassword(email.trim());
    setIsLoading(false);

    if (response.success) {
      setIsSent(true);
    } else {
      toast.error(response.error || 'Something went wrong. Please try again.');
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
            Forgot your<br />
            <span className="text-muted-foreground">password?</span>
          </h1>
          <p className="mt-6 text-lg text-muted-foreground leading-relaxed">
            No worries. Enter your registered email or roll number and we'll send you a secure link to reset it.
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
            {isSent ? (
              /* Success State */
              <div className="text-center">
                <div className="mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-full bg-emerald-500/10">
                  <CheckCircle2 className="h-8 w-8 text-emerald-500" />
                </div>
                <h2 className="font-display text-2xl font-bold tracking-tight">Check your email</h2>
                <p className="mt-3 text-sm text-muted-foreground leading-relaxed">
                  We've sent a password reset link to <strong className="text-foreground">{email}</strong>. Please check your inbox and follow the instructions.
                </p>
                <p className="mt-4 text-xs text-muted-foreground">
                  Didn't receive it? Check your spam folder or{' '}
                  <button
                    onClick={() => { setIsSent(false); setEmail(''); }}
                    className="font-medium text-primary hover:text-primary/80 transition-colors"
                  >
                    try again
                  </button>
                </p>
                <div className="mt-8">
                  <Link to="/login">
                    <Button variant="outline" className="w-full">
                      <ArrowLeft className="mr-2 h-4 w-4" />
                      Back to Sign In
                    </Button>
                  </Link>
                </div>
              </div>
            ) : (
              /* Form State */
              <>
                <div className="mb-8">
                  <h2 className="font-display text-2xl font-bold tracking-tight">Reset Password</h2>
                  <p className="mt-2 text-sm text-muted-foreground">
                    Enter your email address or roll number and we'll send you a link to reset your password.
                  </p>
                </div>

                <form onSubmit={handleSubmit} className="space-y-5">
                  <div className="space-y-2">
                    <Label htmlFor="email" className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                      Email or Roll Number
                    </Label>
                    <div className="relative">
                      <Mail className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                      <Input
                        id="email"
                        type="text"
                        placeholder="name@example.com or 23BSM006"
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                        required
                        autoComplete="email"
                        className="h-11 bg-background/50 pl-10 focus:bg-background"
                      />
                    </div>
                  </div>

                  <Button type="submit" className="w-full h-11 text-base mt-2" disabled={isLoading}>
                    {isLoading ? 'Sending...' : 'Send Reset Link'}
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
