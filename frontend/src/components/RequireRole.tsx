import type { ReactNode } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import type { Role } from '../api';
import { signInPath, useAuth } from '../auth';
import { ROLE_LABEL } from '../format';
import { Loading, Notice } from './ui';

/** RequireRole renders children only for a signed-in user with one of the roles. */
export function RequireRole({ roles, children }: { roles: Role[]; children: ReactNode }) {
  const { user, initializing } = useAuth();
  const location = useLocation();

  if (initializing) return <Loading label="Checking your session…" />;
  if (!user) return <Navigate to={signInPath(location.pathname + location.search)} replace />;
  if (!roles.includes(user.role)) {
    return (
      <div className="container narrow section">
        <Notice kind="warning">
          This page is available to: {roles.map((r) => ROLE_LABEL[r]).join(', ')}. You are signed in as{' '}
          <strong>{ROLE_LABEL[user.role]}</strong>.
        </Notice>
      </div>
    );
  }
  return <>{children}</>;
}
