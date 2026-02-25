import { BrowserRouter, Routes, Route } from 'react-router-dom';
import Layout from '@/components/layout/Layout';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import Login from '@/pages/Login';
import Dashboard from '@/pages/Dashboard';
import DetectedSpam from '@/pages/DetectedSpam';
import SpamCheck from '@/pages/SpamCheck';
import ApprovedUsers from '@/pages/ApprovedUsers';
import Samples from '@/pages/Samples';
import Dictionary from '@/pages/Dictionary';
import Channels from '@/pages/Channels';
import ChannelSettings from '@/pages/ChannelSettings';
import Bots from '@/pages/Bots';
import AdminUsers from '@/pages/AdminUsers';
import Settings from '@/pages/Settings';
import ChangePassword from '@/pages/ChangePassword';

export default function App() {
  return (
    <BrowserRouter basename="/app">
      <Routes>
        <Route path="/login" element={<Login />} />

        <Route path="/" element={
          <ProtectedRoute>
            <Layout />
          </ProtectedRoute>
        }>
          <Route index element={<Dashboard />} />
          <Route path="detected-spam" element={<DetectedSpam />} />
          <Route path="spam-check" element={
            <ProtectedRoute requireRole={['superadmin', 'admin']}>
              <SpamCheck />
            </ProtectedRoute>
          } />
          <Route path="approved-users" element={<ApprovedUsers />} />
          <Route path="samples" element={
            <ProtectedRoute requireRole={['superadmin', 'admin']}>
              <Samples />
            </ProtectedRoute>
          } />
          <Route path="dictionary" element={
            <ProtectedRoute requireRole={['superadmin', 'admin']}>
              <Dictionary />
            </ProtectedRoute>
          } />
          <Route path="bots" element={
            <ProtectedRoute requireRole={['superadmin']}>
              <Bots />
            </ProtectedRoute>
          } />
          <Route path="channels" element={
            <ProtectedRoute requireRole={['superadmin']}>
              <Channels />
            </ProtectedRoute>
          } />
          <Route path="channels/:gid/settings" element={
            <ProtectedRoute requireRole={['superadmin', 'admin']}>
              <ChannelSettings />
            </ProtectedRoute>
          } />
          <Route path="admin-users" element={
            <ProtectedRoute requireRole={['superadmin']}>
              <AdminUsers />
            </ProtectedRoute>
          } />
          <Route path="settings" element={
            <ProtectedRoute requireRole={['superadmin', 'admin']}>
              <Settings />
            </ProtectedRoute>
          } />
          <Route path="change-password" element={<ChangePassword />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
