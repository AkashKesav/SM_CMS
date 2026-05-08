
export function LoadingScreen({ message = 'Loading...' }: { message?: string }) {
  return (
    <div className="auth-grid flex min-h-screen flex-col items-center justify-center bg-background">
      <div className="glass animate-spring-in flex items-center gap-4 rounded-2xl p-4 pr-6 shadow-float">
        <div className="flex h-16 w-16 items-center justify-center p-0.5">
          <img src="/logo.png" alt="IIITDMJ Logo" className="h-full w-full object-contain animate-pulse drop-shadow-sm" />
        </div>
        <div>
          <p className="font-display text-sm font-semibold text-foreground">SM CMS</p>
          <p className="text-xs text-muted-foreground">{message}</p>
        </div>
      </div>
    </div>
  );
}

export function LoadingSpinner({ size = 'default' }: { size?: 'sm' | 'default' | 'lg' }) {
  const sizeClasses = {
    sm: 'w-4 h-4 border-2',
    default: 'w-6 h-6 border-2',
    lg: 'w-10 h-10 border-3',
  };

  return (
    <div className={`animate-spin rounded-full border-t-primary border-r-primary border-b-transparent border-l-transparent ${sizeClasses[size]}`} />
  );
}
