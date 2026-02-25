import { Outlet } from 'react-router-dom';
import { useEffect } from 'react';
import Sidebar from './Sidebar';
import Header from './Header';
import { useAuthStore } from '@/store/authStore';
import { useChannelStore } from '@/store/channelStore';

export default function Layout() {
  const fetchMe = useAuthStore((s) => s.fetchMe);
  const fetchChannels = useChannelStore((s) => s.fetchChannels);

  useEffect(() => {
    fetchMe();
    fetchChannels();
  }, [fetchMe, fetchChannels]);

  return (
    <div className="flex min-h-screen bg-gray-50">
      <Sidebar />
      <div className="flex-1 flex flex-col">
        <Header />
        <main className="flex-1 p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
