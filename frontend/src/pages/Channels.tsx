import { useEffect, useState } from 'react';
import { Plus, Trash2, Settings, Radio } from 'lucide-react';
import { channelsApi } from '@/api/channels';
import { botsApi } from '@/api/bots';
import { useChannelStore } from '@/store/channelStore';
import type { Bot, Channel } from '@/types';
import { useNavigate } from 'react-router-dom';

export default function Channels() {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [bots, setBots] = useState<Bot[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [form, setForm] = useState({ gid: '', telegram_id: '', name: '', username: '', bot_id: '0' });
  const navigate = useNavigate();
  const fetchChannels = useChannelStore((s) => s.fetchChannels);

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      try {
        const [chResp, botResp] = await Promise.all([channelsApi.list(), botsApi.list()]);
        setChannels(chResp.data);
        setBots(botResp.data);
      } catch {
        // handle error
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await channelsApi.create({
        gid: form.gid,
        telegram_id: parseInt(form.telegram_id),
        bot_id: parseInt(form.bot_id),
        name: form.name,
        username: form.username,
        active: true,
      });
      const response = await channelsApi.list();
      setChannels(response.data);
      fetchChannels();
      setForm({ gid: '', telegram_id: '', name: '', username: '', bot_id: '0' });
      setShowAdd(false);
    } catch {
      // handle error
    }
  };

  const handleDelete = async (gid: string) => {
    if (!confirm('Delete this channel and all its settings?')) return;
    try {
      await channelsApi.delete(gid);
      setChannels((prev) => prev.filter((c) => c.gid !== gid));
      fetchChannels();
    } catch {
      // handle error
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-gray-900">Channels</h2>
        <button
          onClick={() => setShowAdd(!showAdd)}
          className="inline-flex items-center gap-2 bg-primary-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-primary-700 transition-colors"
        >
          <Plus size={16} />
          Add Channel
        </button>
      </div>

      {showAdd && (
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <form onSubmit={handleAdd} className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Group ID (gid)</label>
              <input
                type="text"
                value={form.gid}
                onChange={(e) => setForm({ ...form, gid: e.target.value })}
                required
                className="w-full px-4 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                placeholder="unique-group-id"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Telegram Chat ID</label>
              <input
                type="text"
                value={form.telegram_id}
                onChange={(e) => setForm({ ...form, telegram_id: e.target.value })}
                required
                className="w-full px-4 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                placeholder="-1001234567890"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
              <input
                type="text"
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                required
                className="w-full px-4 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                placeholder="My Channel"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Username</label>
              <input
                type="text"
                value={form.username}
                onChange={(e) => setForm({ ...form, username: e.target.value })}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                placeholder="@channel_username (optional)"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Bot</label>
              <select
                value={form.bot_id}
                onChange={(e) => setForm({ ...form, bot_id: e.target.value })}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
              >
                <option value="0">No bot assigned</option>
                {bots.filter((b) => b.active).map((bot) => (
                  <option key={bot.id} value={bot.id}>
                    {bot.name} {bot.username ? `(@${bot.username})` : ''}
                  </option>
                ))}
              </select>
            </div>
            <div className="col-span-2">
              <button
                type="submit"
                className="bg-green-600 text-white px-6 py-2 rounded-lg text-sm font-medium hover:bg-green-700 transition-colors"
              >
                Create Channel
              </button>
            </div>
          </form>
        </div>
      )}

      <div className="grid gap-4">
        {channels.map((channel) => (
          <div key={channel.gid} className="bg-white rounded-xl border border-gray-200 p-6">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div className={`w-12 h-12 rounded-xl flex items-center justify-center ${
                  channel.active ? 'bg-green-100 text-green-600' : 'bg-gray-100 text-gray-400'
                }`}>
                  <Radio size={24} />
                </div>
                <div>
                  <h3 className="font-semibold text-gray-900">{channel.name}</h3>
                  <p className="text-sm text-gray-500">
                    {channel.username ? `@${channel.username} · ` : ''}
                    GID: {channel.gid} · TG ID: {channel.telegram_id}
                    {channel.bot_id > 0 && (() => {
                      const bot = bots.find((b) => b.id === channel.bot_id);
                      return bot ? ` · Bot: ${bot.name}` : '';
                    })()}
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <span className={`px-2.5 py-1 rounded-full text-xs font-medium ${
                  channel.active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'
                }`}>
                  {channel.active ? 'Active' : 'Inactive'}
                </span>
                <button
                  onClick={() => navigate(`/channels/${channel.gid}/settings`)}
                  className="p-2 text-gray-400 hover:text-primary-600 hover:bg-primary-50 rounded-lg transition-colors"
                  title="Settings"
                >
                  <Settings size={18} />
                </button>
                <button
                  onClick={() => handleDelete(channel.gid)}
                  className="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                  title="Delete"
                >
                  <Trash2 size={18} />
                </button>
              </div>
            </div>
          </div>
        ))}
        {channels.length === 0 && (
          <div className="bg-white rounded-xl border border-gray-200 p-12 text-center text-gray-400">
            <Radio size={48} className="mx-auto mb-3 opacity-30" />
            No channels registered yet
          </div>
        )}
      </div>
    </div>
  );
}
