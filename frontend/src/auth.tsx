import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { api, getToken, onUnauthorized, setToken, type Profile, type Role } from './api';

interface AuthState {
  /** Signed-in user with loyalty data, or null for a guest. */
  user: Profile | null;
  /** True while the stored token is being checked on start-up. */
  initializing: boolean;
  login: (email: string, password: string) => Promise<Profile>;
  register: (name: string, email: string, password: string) => Promise<Profile>;
  logout: () => void;
  /** Re-reads /api/me, e.g. after an order is paid (loyalty may change). */
  refreshProfile: () => Promise<void>;
  hasRole: (...roles: Role[]) => boolean;
}

const AuthContext = createContext<AuthState | null>(null);

/** safeNext accepts only same-site absolute paths, so "?next=" cannot redirect off-site. */
export function safeNext(next: string | null | undefined, fallback = '/'): string {
  if (!next || !next.startsWith('/') || next.startsWith('//') || next.includes('\\')) return fallback;
  return next;
}

/** signInPath builds the sign-in URL that returns to `next` afterwards. */
export function signInPath(next: string): string {
  return `/signin?next=${encodeURIComponent(next)}`;
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<Profile | null>(null);
  const [initializing, setInitializing] = useState(() => getToken() !== null);
  const navigate = useNavigate();
  const location = useLocation();

  // Restore the session from the stored token.
  useEffect(() => {
    if (!getToken()) return;
    let cancelled = false;
    api
      .me()
      .then((p) => !cancelled && setUser(p))
      .catch(() => {
        if (!cancelled) {
          setToken(null);
          setUser(null);
        }
      })
      .finally(() => !cancelled && setInitializing(false));
    return () => {
      cancelled = true;
    };
  }, []);

  // A 401 on an authenticated request: drop the session and ask the user to sign in again.
  const here = location.pathname + location.search;
  useEffect(() => {
    onUnauthorized(() => {
      setUser(null);
      if (!location.pathname.startsWith('/signin')) {
        navigate(signInPath(here), { replace: true, state: { expired: true } });
      }
    });
    return () => onUnauthorized(null);
  }, [navigate, here, location.pathname]);

  const finishSignIn = useCallback(async (token: string) => {
    setToken(token);
    const profile = await api.me();
    setUser(profile);
    return profile;
  }, []);

  const login = useCallback(
    async (email: string, password: string) => {
      const res = await api.login(email, password);
      return finishSignIn(res.token);
    },
    [finishSignIn],
  );

  const register = useCallback(
    async (name: string, email: string, password: string) => {
      const res = await api.register(name, email, password);
      return finishSignIn(res.token);
    },
    [finishSignIn],
  );

  const logout = useCallback(() => {
    setToken(null);
    setUser(null);
    navigate('/');
  }, [navigate]);

  const refreshProfile = useCallback(async () => {
    if (!getToken()) return;
    try {
      setUser(await api.me());
    } catch {
      /* 401 is handled globally; other errors keep the old profile */
    }
  }, []);

  const hasRole = useCallback((...roles: Role[]) => !!user && roles.includes(user.role), [user]);

  const value = useMemo(
    () => ({ user, initializing, login, register, logout, refreshProfile, hasRole }),
    [user, initializing, login, register, logout, refreshProfile, hasRole],
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used inside <AuthProvider>');
  return ctx;
}
