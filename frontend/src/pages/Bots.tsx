import { useEffect, useState } from 'react';
import { Plus, Trash2, Bot as BotIcon, CheckCircle, RefreshCw } from 'lucide-react';
import { botsApi } from '@/api/bots';
import type { Bot } from '@/types';

export default function Bots() {
  const [bots, setBots] = useState<Bot[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [form, setForm] = useState({ name: '', token: '' });
  const [validating, setValidating] = useState<number | null>(null);

  useEffect(() => {
    fetchBots();
  }, []);

  const fetchBots = async () => {
    setLoading(true);
    try {
      const response = await botsApi.list();
      setBots(response.data || []);
    } catch {
      // handle error
    } finally {
      setLoading(false);
    }
  };

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await botsApi.create(form);
      await fetchBots();
      setForm({ name: '', token: '' });
      setShowAdd(false);
    } catch {
      // handle error
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this bot?')) return;
    try {
      await botsApi.delete(id);
      setBots((prev) => prev.filter((b) => b.id !== id));
    } catch {
      // handle error
    }
  };

  const handleToggleActive = async (bot: Bot) => {
    try {
      await botsApi.update(bot.id, { active: !bot.active });
      setBots((prev) =>
        prev.map((b) => (b.id === bot.id ? { ...b, active: !b.active } : b))
      );
    } catch {
      // handle error
    }
  };

  const handleValidate = async (id: number) => {
    setValidating(id);
    try {
      const response = await botsApi.validate(id);
      if (response.data.valid) {
        await fetchBots();
        alert(`Bot is valid! Username: @${response.data.username}`);
      } else {
        alert(`Bot validation failed: ${response.data.error}`);
      }
    } catch {
      alert('Failed to validate bot');
    } finally {
      setValidating(null);
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
        <h2 className="text-2xl font-bold text-gray-900">Bots</h2>
        <button
          onClick={() => setShowAdd(!showAdd)}
          className="inline-flex items-center gap-2 bg-primary-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-primary-700 transition-colors"
        >
          <Plus size={16} />
          Add Bot
        </button>
      </div>

      {showAdd && (
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <form onSubmit={handleAdd} className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
              <input
                type="text"
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                placeholder="My Spam Bot"
                required
                className="w-full px-4 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Bot Token</label>
              <input
                type="text"
                value={form.token}
                onChange={(e) => setForm({ ...form, token: e.target.value })}
                placeholder="123456:ABC-DEF..."
                required
                className="w-full px-4 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
            </div>
            <div className="col-span-2">
              <button
                type="submit"
                className="bg-green-600 text-white px-6 py-2 rounded-lg text-sm font-medium hover:bg-green-700 transition-colors"
              >
                Create Bot
              </button>
            </div>
          </form>
        </div>
      )}

      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="bg-gray-50 border-b border-gray-200">
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Name</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Username</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Token</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {bots.map((bot) => (
              <tr key={bot.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 text-sm text-gray-900 font-medium">{bot.name}</td>
                <td className="px-4 py-3 text-sm text-gray-700">
                  {bot.username ? `@${bot.username}` : '—'}
                </td>
                <td className="px-4 py-3 text-sm text-gray-500 font-mono">{bot.token}</td>
                <td className="px-4 py-3">
                  <button
                    onClick={() => handleToggleActive(bot)}
                    className={`px-2.5 py-1 rounded-full text-xs font-medium ${
                      bot.active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'
                    }`}
                  >
                    {bot.active ? 'Active' : 'Disabled'}
                  </button>
                </td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => handleValidate(bot.id)}
                      disabled={validating === bot.id}
                      className="text-blue-500 hover:text-blue-700 transition-colors disabled:opacity-50"
                      title="Validate token"
                    >
                      {validating === bot.id ? (
                        <RefreshCw size={16} className="animate-spin" />
                      ) : (
                        <CheckCircle size={16} />
                      )}
                    </button>
                    <button
                      onClick={() => handleDelete(bot.id)}
                      className="text-red-500 hover:text-red-700 transition-colors"
                      title="Delete"
                    >
                      <Trash2 size={16} />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
            {bots.length === 0 && (
              <tr>
                <td colSpan={5} className="px-4 py-12 text-center text-gray-400">
                  <BotIcon size={48} className="mx-auto mb-3 opacity-30" />
                  No bots registered yet
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
