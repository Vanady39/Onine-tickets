import { useState, type FormEvent } from 'react';
import { Link, Navigate, useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import type { Profile } from '../api';
import { errorMessage } from '../api';
import { safeNext, useAuth } from '../auth';
import { ErrorBox, Notice } from '../components/ui';
import { useTitle } from '../hooks';

/** Where to go after signing in: the requested page, or the role's home page. */
function destination(next: string | null, user: Profile): string {
  const target = safeNext(next, '');
  if (target && !target.startsWith('/signin') && !target.startsWith('/register')) return target;
  switch (user.role) {
    case 'cashier':
      return '/box-office';
    case 'manager':
    case 'admin':
      return '/admin';
    default:
      return '/';
  }
}

const DEMO = [
  ['customer@tickets.local', 'Customer'],
  ['loyal@tickets.local', 'Loyal customer'],
  ['cashier@tickets.local', 'Cashier'],
  ['manager@tickets.local', 'Manager'],
  ['admin@tickets.local', 'Admin'],
] as const;

/** Sign in (UC06). */
export function SignIn() {
  useTitle('Sign in');
  const { login, user, initializing } = useAuth();
  const [params] = useSearchParams();
  const next = params.get('next');
  const navigate = useNavigate();
  const expired = !!(useLocation().state as { expired?: boolean } | null)?.expired;
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  if (!initializing && user && !busy) return <Navigate to={destination(next, user)} replace />;

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const u = await login(email.trim(), password);
      navigate(destination(next, u), { replace: true });
    } catch (err) {
      setError(errorMessage(err));
      setBusy(false);
    }
  };

  return (
    <div className="container auth section">
      <form className="panel auth-card" onSubmit={submit}>
        <h1>Sign in</h1>
        {expired && <Notice kind="warning">Your session has expired. Please sign in again.</Notice>}
        {next === '/checkout' && !expired && <Notice kind="info">Sign in to complete your order.</Notice>}
        <label className="field">
          <span>E-mail</span>
          <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="email" required autoFocus />
        </label>
        <label className="field">
          <span>Password</span>
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="current-password" required />
        </label>
        {error && <ErrorBox message={error} />}
        <button type="submit" className="btn btn-primary btn-block" disabled={busy}>
          {busy ? 'Signing in…' : 'Sign in'}
        </button>
        <p className="muted small center">
          New here? <Link to={`/register${next ? `?next=${encodeURIComponent(next)}` : ''}`}>Create an account</Link>
        </p>
        <details className="demo">
          <summary>Demo accounts</summary>
          <p className="small muted">Password for all: demo-pass-123</p>
          <div className="chips">
            {DEMO.map(([mail, label]) => (
              <button key={mail} type="button" className="chip" onClick={() => setEmail(mail)}>
                {label}
              </button>
            ))}
          </div>
        </details>
      </form>
    </div>
  );
}

/** Register (UC05). */
export function Register() {
  useTitle('Register');
  const { register, user, initializing } = useAuth();
  const [params] = useSearchParams();
  const next = params.get('next');
  const navigate = useNavigate();
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  if (!initializing && user && !busy) return <Navigate to={destination(next, user)} replace />;

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (password !== confirm) {
      setError('The passwords do not match.');
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const u = await register(name.trim(), email.trim(), password);
      navigate(destination(next, u), { replace: true });
    } catch (err) {
      setError(errorMessage(err));
      setBusy(false);
    }
  };

  return (
    <div className="container auth section">
      <form className="panel auth-card" onSubmit={submit}>
        <h1>Create an account</h1>
        <p className="muted">An account keeps your tickets and purchase history in one place.</p>
        <label className="field">
          <span>Name</span>
          <input value={name} onChange={(e) => setName(e.target.value)} autoComplete="name" required maxLength={100} autoFocus />
        </label>
        <label className="field">
          <span>E-mail</span>
          <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="email" required />
        </label>
        <label className="field">
          <span>Password</span>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="new-password"
            required
            minLength={8}
            maxLength={72}
          />
          <small className="muted">8–72 characters.</small>
        </label>
        <label className="field">
          <span>Repeat password</span>
          <input type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} autoComplete="new-password" required />
        </label>
        {error && <ErrorBox message={error} />}
        <button type="submit" className="btn btn-primary btn-block" disabled={busy}>
          {busy ? 'Creating account…' : 'Register'}
        </button>
        <p className="muted small center">
          Already registered? <Link to={`/signin${next ? `?next=${encodeURIComponent(next)}` : ''}`}>Sign in</Link>
        </p>
      </form>
    </div>
  );
}
