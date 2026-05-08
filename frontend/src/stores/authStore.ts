import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { User, AuthTokens } from '@/types/schema';
import { authApi } from '@/lib/api';

interface AuthState {
  user: User | null;
  tokens: AuthTokens | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;

  // Actions
  initialize: () => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, name?: string, rollNo?: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshToken: () => Promise<void>;
  setUser: (user: User) => void;
  clearError: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      tokens: null,
      isAuthenticated: false,
      isLoading: true,
      error: null,

      initialize: async () => {
        try {
          const { tokens } = get();
          if (tokens?.access_token) {
            const response = await authApi.me();
            if (response.success && response.data?.user) {
              set({
                user: response.data.user,
                isLoading: false,
                isAuthenticated: true,
                error: null,
              });
            } else {
              set({ isLoading: false, isAuthenticated: false, tokens: null, user: null });
            }
          } else {
            set({ isLoading: false, isAuthenticated: false });
          }
        } catch (error) {
          set({ isLoading: false, isAuthenticated: false, tokens: null, user: null });
        }
      },

      login: async (email: string, password: string) => {
        set({ isLoading: true, error: null });
        try {
          const response = await authApi.login(email, password);
          if (response.success && response.data) {
            set({
              user: response.data.user,
              tokens: response.data,
              isAuthenticated: true,
              isLoading: false,
              error: null,
            });
          } else {
            set({ isLoading: false, error: response.error || 'Login failed' });
          }
        } catch (error: any) {
          set({ isLoading: false, error: error.message || 'Login failed' });
        }
      },

      register: async (email: string, password: string, name?: string, rollNo?: string) => {
        set({ isLoading: true, error: null });
        try {
          const response = await authApi.register(email, password, name, rollNo);
          if (response.success) {
            if (response.data?.access_token && response.data?.user) {
              set({
                user: response.data.user,
                tokens: response.data,
                isAuthenticated: true,
                isLoading: false,
                error: null,
              });
            } else {
              set({ isLoading: false, error: null });
            }
          } else {
            set({ isLoading: false, error: response.error || 'Registration failed' });
          }
        } catch (error: any) {
          set({ isLoading: false, error: error.message || 'Registration failed' });
        }
      },

      logout: async () => {
        try {
          await authApi.logout();
        } catch {
          // Ignore logout errors
        }
        set({
          user: null,
          tokens: null,
          isAuthenticated: false,
          error: null,
        });
      },

      refreshToken: async () => {
        const { tokens } = get();
        if (!tokens?.refresh_token) return;

        try {
          const response = await authApi.refreshToken(tokens.refresh_token);
          if (response.success && response.data) {
            set({
              tokens: response.data,
              error: null,
            });
          }
        } catch (error) {
          set({ error: 'Token refresh failed' });
          // Force logout if refresh fails
          get().logout();
        }
      },

      setUser: (user: User) => {
        set({ user });
      },

      clearError: () => {
        set({ error: null });
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        tokens: state.tokens,
        user: state.user,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
);
