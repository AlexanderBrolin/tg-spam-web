import { useEffect, useState } from 'react';
import { ShieldAlert, Plus, Radio, UserCheck } from 'lucide-react';
import { spamApi } from '@/api/spam';
import { useChannelStore } from '@/store/channelStore';
import type { DetectedSpamEntry } from '@/types';
import { format } from 'date-fns';

export default function DetectedSpam() {
  const [entries, setEntries] = useState<DetectedSpamEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [unbanning, setUnbanning] = useState<number | null>(null);
  const selectedGid = useChannelStore((s) => s.selectedGid);

  useEffect(() => {
    if (!selectedGid) return;
    const fetchData = async () => {
      setLoading(true);
      try {
        const response = await spamApi.getDetected(selectedGid);
        setEntries(response.data.entries || []);
      } catch {
        // handle error
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [selectedGid]);

  const handleAddToSamples = async (id: number, text: string) => {
    try {
      await spamApi.addToSamples(id, text);
      setEntries((prev) =>
        prev.map((e) => (e.id === id ? { ...e, added: true } : e))
      );
    } catch {
      // handle error
    }
  };

  const handleUnban = async (entry: DetectedSpamEntry) => {
    if (!selectedGid) return;
    setUnbanning(entry.id);
    try {
      await spamApi.unban(selectedGid, entry.user_id);
      setEntries((prev) => prev.filter((e) => e.id !== entry.id));
    } catch {
      // handle error
    } finally {
      setUnbanning(null);
    }
  };

  if (!selectedGid) {
    return (
      <div className="flex flex-col items-center justify-center h-64 text-gray-400">
        <Radio size={48} className="mb-3 opacity-30" />
        <p className="text-sm">Select a channel in the sidebar to view detected spam</p>
      </div>
    );
  }

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
        <h2 className="text-2xl font-bold text-gray-900">Detected Spam</h2>
        <span className="text-sm text-gray-500">{entries.length} entries</span>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="bg-gray-50 border-b border-gray-200">
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Time</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">User</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Message</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Checks</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {entries.map((entry) => (
              <tr key={entry.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 text-sm text-gray-600 whitespace-nowrap">
                  {format(new Date(entry.timestamp), 'dd.MM.yyyy HH:mm')}
                </td>
                <td className="px-4 py-3 text-sm text-gray-900">
                  {entry.user_name}
                  <span className="block text-xs text-gray-400">ID: {entry.user_id}</span>
                </td>
                <td className="px-4 py-3 text-sm text-gray-700 max-w-md truncate">
                  {entry.text}
                </td>
                <td className="px-4 py-3">
                  <div className="flex flex-wrap gap-1">
                    {entry.checks?.filter(c => c.spam).map((check, i) => (
                      <span
                        key={i}
                        className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-700"
                      >
                        {check.name}
                      </span>
                    ))}
                  </div>
                </td>
                <td className="px-4 py-3">
                  <div className="flex flex-col gap-1">
                    {entry.added ? (
                      <span className="text-xs text-green-600">Added</span>
                    ) : (
                      <button
                        onClick={() => handleAddToSamples(entry.id, entry.text)}
                        className="inline-flex items-center gap-1 text-xs text-primary-600 hover:text-primary-700 font-medium"
                      >
                        <Plus size={14} />
                        Add to samples
                      </button>
                    )}
                    <button
                      onClick={() => handleUnban(entry)}
                      disabled={unbanning === entry.id}
                      className="inline-flex items-center gap-1 text-xs text-amber-600 hover:text-amber-700 font-medium disabled:opacity-50"
                    >
                      <UserCheck size={14} />
                      {unbanning === entry.id ? 'Unbanning...' : 'Unban'}
                    </button>
                  </div>
                </td>
              </tr>
            ))}
            {entries.length === 0 && (
              <tr>
                <td colSpan={5} className="px-4 py-12 text-center text-gray-400">
                  <ShieldAlert size={48} className="mx-auto mb-3 opacity-30" />
                  No spam detected yet
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
