import { useEffect, useState } from 'react';
import { ShieldAlert, Users, Radio, TrendingUp } from 'lucide-react';
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  PieChart, Pie, Cell,
} from 'recharts';
import { statsApi } from '@/api/stats';
import { useChannelStore } from '@/store/channelStore';
import type { DashboardStats } from '@/types';

const COLORS = ['#3b82f6', '#ef4444', '#10b981', '#f59e0b', '#8b5cf6', '#ec4899'];

export default function Dashboard() {
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [loading, setLoading] = useState(true);
  const selectedGid = useChannelStore((s) => s.selectedGid);

  useEffect(() => {
    const fetchStats = async () => {
      setLoading(true);
      try {
        const response = await statsApi.get(selectedGid || undefined);
        setStats(response.data);
      } catch {
        // handle error
      } finally {
        setLoading(false);
      }
    };
    fetchStats();
  }, [selectedGid]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600" />
      </div>
    );
  }

  const pieData = stats?.spam_by_type
    ? Object.entries(stats.spam_by_type).map(([name, value]) => ({ name, value }))
    : [];

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold text-gray-900">Dashboard</h2>

      {/* stat cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          title="Total Spam Blocked"
          value={stats?.total_spam ?? 0}
          icon={<ShieldAlert size={24} />}
          color="text-red-600 bg-red-100"
        />
        <StatCard
          title="Approved Users"
          value={stats?.total_approved ?? 0}
          icon={<Users size={24} />}
          color="text-green-600 bg-green-100"
        />
        <StatCard
          title="Active Channels"
          value={stats?.active_channels ?? 0}
          icon={<Radio size={24} />}
          color="text-blue-600 bg-blue-100"
        />
        <StatCard
          title="Detection Rate"
          value={stats?.total_spam ? `${stats.total_spam}/day` : '0'}
          icon={<TrendingUp size={24} />}
          color="text-purple-600 bg-purple-100"
        />
      </div>

      {/* charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* timeline */}
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Spam Over Time</h3>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={stats?.spam_timeline ?? []}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="date" tick={{ fontSize: 12 }} />
              <YAxis tick={{ fontSize: 12 }} />
              <Tooltip />
              <Line type="monotone" dataKey="count" stroke="#3b82f6" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>

        {/* by type */}
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Spam by Detection Type</h3>
          {pieData.length > 0 ? (
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie data={pieData} dataKey="value" nameKey="name" cx="50%" cy="50%" outerRadius={100} label>
                  {pieData.map((_, index) => (
                    <Cell key={index} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex items-center justify-center h-[300px] text-gray-400">
              No data available
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function StatCard({ title, value, icon, color }: {
  title: string;
  value: number | string;
  icon: React.ReactNode;
  color: string;
}) {
  return (
    <div className="bg-white rounded-xl border border-gray-200 p-6">
      <div className="flex items-center gap-4">
        <div className={`w-12 h-12 rounded-xl flex items-center justify-center ${color}`}>
          {icon}
        </div>
        <div>
          <p className="text-sm text-gray-500">{title}</p>
          <p className="text-2xl font-bold text-gray-900">{value}</p>
        </div>
      </div>
    </div>
  );
}
