import { useSearchParams } from 'react-router-dom';
import { useAuth } from '../../auth';
import { PageHeader } from '../../components/ui';
import { useTitle } from '../../hooks';
import { EventsAdmin } from './EventsAdmin';
import { Reports } from './Reports';
import { SessionForm } from './SessionForm';
import { Venues } from './Venues';

const TABS = [
  { id: 'events', label: 'Events' },
  { id: 'sessions', label: 'Schedule session' },
  { id: 'reports', label: 'Sales reports' },
  { id: 'venues', label: 'Venues' },
] as const;

type TabId = (typeof TABS)[number]['id'];

/** Back office (UC24–UC27) for event managers and administrators. */
export function AdminPage() {
  useTitle('Back office');
  const { user } = useAuth();
  const [params, setParams] = useSearchParams();
  const raw = params.get('tab');
  const tab: TabId = TABS.some((t) => t.id === raw) ? (raw as TabId) : 'events';
  const eventId = Number(params.get('event')) || undefined;

  const go = (id: TabId, event?: number) => {
    const next = new URLSearchParams({ tab: id });
    if (event) next.set('event', String(event));
    setParams(next);
  };

  return (
    <div className="container section">
      <PageHeader title="Back office" subtitle={user?.role === 'admin' ? 'Administrator' : 'Event manager'} />
      <div className="tabs" role="tablist" aria-label="Back office sections">
        {TABS.map((t) => (
          <button
            key={t.id}
            type="button"
            role="tab"
            aria-selected={tab === t.id}
            className={`tab${tab === t.id ? ' active' : ''}`}
            onClick={() => go(t.id)}
          >
            {t.label}
          </button>
        ))}
      </div>
      <div role="tabpanel">
        {tab === 'events' && <EventsAdmin onSchedule={(id) => go('sessions', id)} onReport={(id) => go('reports', id)} />}
        {tab === 'sessions' && <SessionForm initialEventId={eventId} />}
        {tab === 'reports' && <Reports eventId={eventId} onSelect={(id) => go('reports', id)} />}
        {tab === 'venues' && <Venues canCreate={user?.role === 'admin'} />}
      </div>
    </div>
  );
}
