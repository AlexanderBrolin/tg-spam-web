import { useEffect, useState } from 'react';
import { Plus, Trash2, Users } from 'lucide-react';
import { usersApi } from '@/api/users';
import { useChannelStore } from '@/store/channelStore';
import type { ApprovedUser } from '@/types';

export default function ApprovedUsers() {
  const [users, setUsers] = useState<ApprovedUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [newUserId, setNewUserId] = useState('');
  const [newUserName, setNewUserName] = useState('');
  const selectedGid = useChannelStore((s) => s.selectedGid);

  useEffect(() => {
    if (!selectedGid) return;
    const fetchData = async () => {
      setLoading(true);
      try {
        const response = await usersApi.getApproved(selectedGid);
        setUsers(response.data);
      } catch {
        // handle error
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [selectedGid]);

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedGid || !newUserId.trim()) return;
    try {
      await usersApi.addApproved(selectedGid, newUserId, newUserName);
      const response = await usersApi.getApproved(selectedGid);
      setUsers(response.data);
      setNewUserId('');
      setNewUserName('');
      setShowAdd(false);
    } catch {
      // handle error
    }
  };

  const handleRemove = async (userId: string) => {
    if (!selectedGid) return;
    if (!confirm('Remove this user from the approved list?')) return;
    try {
      await usersApi.removeApproved(selectedGid, userId);
      setUsers((prev) => prev.filter((u) => u.user_id !== userId));
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
        <h2 className="text-2xl font-bold text-gray-900">Approved Users</h2>
        <button
          onClick={() => setShowAdd(!showAdd)}
          className="inline-flex items-center gap-2 bg-primary-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-primary-700 transition-colors"
        >
          <Plus size={16} />
          Add User
        </button>
      </div>

      {showAdd && (
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <form onSubmit={handleAdd} className="flex gap-4 items-end">
            <div className="flex-1">
              <label className="block text-sm font-medium text-gray-700 mb-1">User ID</label>
              <input
                type="text"
                value={newUserId}
                onChange={(e) => setNewUserId(e.target.value)}
                required
                className="w-full px-4 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                placeholder="Telegram User ID"
              />
            </div>
            <div className="flex-1">
              <label className="block text-sm font-medium text-gray-700 mb-1">Username</label>
              <input
                type="text"
                value={newUserName}
                onChange={(e) => setNewUserName(e.target.value)}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                placeholder="@username (optional)"
              />
            </div>
            <button
              type="submit"
              className="bg-green-600 text-white px-6 py-2 rounded-lg text-sm font-medium hover:bg-green-700 transition-colors"
            >
              Add
            </button>
          </form>
        </div>
      )}

      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="bg-gray-50 border-b border-gray-200">
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">User ID</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Username</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {users.map((user) => (
              <tr key={user.user_id} className="hover:bg-gray-50">
                <td className="px-4 py-3 text-sm text-gray-900 font-mono">{user.user_id}</td>
                <td className="px-4 py-3 text-sm text-gray-700">{user.user_name || '—'}</td>
                <td className="px-4 py-3">
                  <button
                    onClick={() => handleRemove(user.user_id)}
                    className="text-red-500 hover:text-red-700 transition-colors"
                    title="Remove"
                  >
                    <Trash2 size={16} />
                  </button>
                </td>
              </tr>
            ))}
            {users.length === 0 && (
              <tr>
                <td colSpan={3} className="px-4 py-12 text-center text-gray-400">
                  <Users size={48} className="mx-auto mb-3 opacity-30" />
                  No approved users yet
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
