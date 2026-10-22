import { useEffect, useState } from 'react';
import { Link, NavLink, useLocation } from 'react-router-dom';
import { useAuth } from '../auth';
import { useCart } from '../cart';
import { ROLE_LABEL } from '../format';

export function Header() {
  const { user, logout, hasRole } = useAuth();
  const { season } = useCart();
  const [open, setOpen] = useState(false);
  const location = useLocation();

  // Close the mobile menu after navigation.
  useEffect(() => {
    setOpen(false);
  }, [location.pathname]);

  const links: { to: string; label: string; badge?: number }[] = [{ to: '/', label: 'Programme' }];
  if (hasRole('customer')) {
    links.push({ to: '/orders', label: 'My tickets' });
    if (user?.loyal) links.push({ to: '/season', label: 'Season ticket', badge: season.length || undefined });
  }
  if (hasRole('cashier', 'admin')) links.push({ to: '/box-office', label: 'Box office' });
  if (hasRole('manager', 'admin')) links.push({ to: '/admin', label: 'Back office' });

  return (
    <header className="site-header">
      <div className="container header-inner">
        <Link to="/" className="brand">
          <span className="brand-mark" aria-hidden="true">
            <svg viewBox="0 0 32 32" width="28" height="28">
              <rect width="32" height="32" rx="8" fill="currentColor" />
              <path d="M8 11h16v3a2 2 0 0 0 0 4v3H8v-3a2 2 0 0 0 0-4z" fill="#fff" />
            </svg>
          </span>
          <span>Online Tickets</span>
        </Link>
        <button
          type="button"
          className="menu-toggle"
          aria-expanded={open}
          aria-controls="main-nav"
          onClick={() => setOpen((o) => !o)}
        >
          <span className="sr-only">Menu</span>
          <span className="burger" aria-hidden="true" />
        </button>
        <nav id="main-nav" className={`main-nav${open ? ' open' : ''}`} aria-label="Main">
          <ul>
            {links.map((l) => (
              <li key={l.to}>
                <NavLink to={l.to} end={l.to === '/'}>
                  {l.label}
                  {l.badge ? <span className="nav-count">{l.badge}</span> : null}
                </NavLink>
              </li>
            ))}
          </ul>
          <div className="nav-user">
            {user ? (
              <>
                <span className="user-chip" title={user.email}>
                  <span className="user-name">{user.name}</span>
                  <span className="user-role">
                    {ROLE_LABEL[user.role]}
                    {user.role === 'customer' && user.loyal ? ' · Loyal' : ''}
                  </span>
                </span>
                <button type="button" className="btn btn-small btn-ghost" onClick={logout}>
                  Sign out
                </button>
              </>
            ) : (
              <>
                <Link className="btn btn-small btn-ghost" to={`/signin?next=${encodeURIComponent(location.pathname + location.search)}`}>
                  Sign in
                </Link>
                <Link className="btn btn-small btn-primary" to="/register">
                  Register
                </Link>
              </>
            )}
          </div>
        </nav>
      </div>
    </header>
  );
}
