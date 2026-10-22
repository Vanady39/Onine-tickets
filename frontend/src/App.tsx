import { useEffect } from 'react';
import { Link, Route, Routes, useLocation } from 'react-router-dom';
import { Header } from './components/Header';
import { RequireRole } from './components/RequireRole';
import { Empty } from './components/ui';
import { AdminPage } from './pages/admin/AdminPage';
import { Register, SignIn } from './pages/AuthPages';
import { BoxOffice } from './pages/BoxOffice';
import { Checkout } from './pages/Checkout';
import { EventPage } from './pages/EventPage';
import { MyOrders } from './pages/MyOrders';
import { OrderPage } from './pages/OrderPage';
import { Programme } from './pages/Programme';
import { SeasonTicket } from './pages/SeasonTicket';
import { SessionPage } from './pages/SessionPage';

export function App() {
  const { pathname } = useLocation();
  useEffect(() => {
    window.scrollTo(0, 0);
  }, [pathname]);

  return (
    <div className="app">
      <a className="skip-link" href="#content">
        Skip to content
      </a>
      <Header />
      <main id="content">
        <Routes>
          <Route path="/" element={<Programme />} />
          <Route path="/events/:id" element={<EventPage />} />
          <Route path="/sessions/:id" element={<SessionPage />} />
          <Route path="/signin" element={<SignIn />} />
          <Route path="/register" element={<Register />} />
          <Route
            path="/checkout"
            element={
              <RequireRole roles={['customer']}>
                <Checkout />
              </RequireRole>
            }
          />
          <Route
            path="/orders"
            element={
              <RequireRole roles={['customer']}>
                <MyOrders />
              </RequireRole>
            }
          />
          <Route
            path="/orders/:id"
            element={
              <RequireRole roles={['customer']}>
                <OrderPage />
              </RequireRole>
            }
          />
          <Route
            path="/season"
            element={
              <RequireRole roles={['customer']}>
                <SeasonTicket />
              </RequireRole>
            }
          />
          <Route
            path="/box-office"
            element={
              <RequireRole roles={['cashier', 'admin']}>
                <BoxOffice />
              </RequireRole>
            }
          />
          <Route
            path="/admin"
            element={
              <RequireRole roles={['manager', 'admin']}>
                <AdminPage />
              </RequireRole>
            }
          />
          <Route
            path="*"
            element={
              <div className="container narrow section">
                <Empty title="Page not found">
                  <Link className="btn btn-primary" to="/">
                    Go to the programme
                  </Link>
                </Empty>
              </div>
            }
          />
        </Routes>
      </main>
      <footer className="site-footer">
        <div className="container">
          Online Tickets · concerts, cinema, theatre and exhibitions · prices in roubles (₽)
        </div>
      </footer>
    </div>
  );
}
