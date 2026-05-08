import { Routes, Route, Navigate } from 'react-router-dom';
import { useEffect } from 'react';
import { useAuthStore } from './stores/authStore';
import { useSchemaStore } from './stores/schemaStore';
import { Layout } from './components/Layout';
import { LoginPage } from './pages/LoginPage';
import { RegisterPage } from './pages/RegisterPage';
import { DashboardPage } from './pages/DashboardPage';
import { TablePage } from './pages/TablePage';
import { SettingsPage } from './pages/SettingsPage';
import { StudentWorkspacePage } from './pages/StudentWorkspacePage';
import { AdminReviewPage } from './pages/AdminReviewPage';
import { AccessControlPage } from './pages/AccessControlPage';
import { NotFoundPage } from './pages/NotFoundPage';
import { CommandPalette } from './components/CommandPalette';
import { ErrorBoundary } from './components/ErrorBoundary';
import { LoadingScreen } from './components/LoadingScreen';

function App() {
  const { isAuthenticated, isLoading: authLoading, initialize, user } = useAuthStore();
  const {
    loadSchema,
    refreshSchema,
    clearCache,
    isLoading: schemaLoading,
  } = useSchemaStore();

  useEffect(() => {
    initialize();
  }, [initialize]);

  useEffect(() => {
    if (isAuthenticated && user?.role === 'admin') {
      loadSchema();
    }
  }, [isAuthenticated, user?.role, loadSchema]);

  useEffect(() => {
    if (!(window as any).__TAURI_INTERNALS__) {
      return;
    }

    let unlistenSync: (() => void) | undefined;
    let unlistenClear: (() => void) | undefined;

    import('@tauri-apps/api/event')
      .then(async ({ listen }) => {
        unlistenSync = await listen('sync-schema-request', () => {
          refreshSchema();
        });
        unlistenClear = await listen('clear-cache-request', () => {
          clearCache();
        });
      })
      .catch(() => {
        // The web build can run without Tauri APIs.
      });

    return () => {
      unlistenSync?.();
      unlistenClear?.();
    };
  }, [refreshSchema, clearCache]);

  if (authLoading || (isAuthenticated && user?.role === 'admin' && schemaLoading)) {
    return <LoadingScreen />;
  }

  const isAdmin = user?.role === 'admin';

  return (
    <ErrorBoundary>
      <Routes>
        <Route path="/login" element={!isAuthenticated ? <LoginPage /> : <Navigate to="/" replace />} />
        <Route path="/register" element={!isAuthenticated ? <RegisterPage /> : <Navigate to="/" replace />} />
        <Route element={<Layout />}>
          <Route path="/" element={isAuthenticated ? (isAdmin ? <DashboardPage /> : <StudentWorkspacePage />) : <Navigate to="/login" replace />} />
          <Route path="/review" element={isAuthenticated && isAdmin ? <AdminReviewPage /> : <Navigate to="/" replace />} />
          <Route path="/access" element={isAuthenticated && isAdmin ? <AccessControlPage /> : <Navigate to="/" replace />} />
          <Route path="/tables/:tableName" element={isAuthenticated && isAdmin ? <TablePage /> : <Navigate to="/" replace />} />
          <Route path="/settings" element={isAuthenticated ? <SettingsPage /> : <Navigate to="/login" replace />} />
        </Route>
        <Route path="*" element={<NotFoundPage />} />
      </Routes>
      {isAdmin && <CommandPalette />}
    </ErrorBoundary>
  );
}

export default App;
