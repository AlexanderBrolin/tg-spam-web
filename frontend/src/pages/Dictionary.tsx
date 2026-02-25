import { useEffect, useState } from 'react';
import { Plus, Trash2, BookText } from 'lucide-react';
import { dictionaryApi } from '@/api/dictionary';
import { useChannelStore } from '@/store/channelStore';
import type { DictionaryEntry } from '@/types';

export default function Dictionary() {
  const [entries, setEntries] = useState<DictionaryEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<'stop_phrase' | 'ignored_word'>('stop_phrase');
  const [newEntry, setNewEntry] = useState('');
  const selectedGid = useChannelStore((s) => s.selectedGid);

  useEffect(() => {
    if (!selectedGid) return;
    const fetchData = async () => {
      setLoading(true);
      try {
        const response = await dictionaryApi.get(selectedGid);
        setEntries(response.data);
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
    if (!selectedGid || !newEntry.trim()) return;
    try {
      await dictionaryApi.add(selectedGid, activeTab, newEntry);
      setEntries((prev) => [...prev, { type: activeTab, data: newEntry }]);
      setNewEntry('');
    } catch {
      // handle error
    }
  };

  const handleRemove = async (type: string, data: string) => {
    if (!selectedGid) return;
    try {
      await dictionaryApi.remove(selectedGid, type, data);
      setEntries((prev) => prev.filter((e) => !(e.type === type && e.data === data)));
    } catch {
      // handle error
    }
  };

  const filtered = entries.filter((e) => e.type === activeTab);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold text-gray-900">Dictionary</h2>

      <div className="flex gap-2">
        <button
          onClick={() => setActiveTab('stop_phrase')}
          className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
            activeTab === 'stop_phrase'
              ? 'bg-red-100 text-red-700'
              : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
          }`}
        >
          Stop Phrases ({entries.filter((e) => e.type === 'stop_phrase').length})
        </button>
        <button
          onClick={() => setActiveTab('ignored_word')}
          className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
            activeTab === 'ignored_word'
              ? 'bg-blue-100 text-blue-700'
              : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
          }`}
        >
          Ignored Words ({entries.filter((e) => e.type === 'ignored_word').length})
        </button>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 p-6">
        <form onSubmit={handleAdd} className="flex gap-4">
          <input
            type="text"
            value={newEntry}
            onChange={(e) => setNewEntry(e.target.value)}
            className="flex-1 px-4 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
            placeholder={activeTab === 'stop_phrase' ? 'Add stop phrase...' : 'Add ignored word...'}
          />
          <button
            type="submit"
            disabled={!newEntry.trim()}
            className="inline-flex items-center gap-2 bg-primary-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-primary-700 disabled:opacity-50 transition-colors"
          >
            <Plus size={16} />
            Add
          </button>
        </form>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        <div className="divide-y divide-gray-200">
          {filtered.map((entry, i) => (
            <div key={i} className="flex items-center justify-between px-4 py-3 hover:bg-gray-50">
              <span className="text-sm text-gray-700">{entry.data}</span>
              <button
                onClick={() => handleRemove(entry.type, entry.data)}
                className="text-red-500 hover:text-red-700 transition-colors"
              >
                <Trash2 size={16} />
              </button>
            </div>
          ))}
          {filtered.length === 0 && (
            <div className="px-4 py-12 text-center text-gray-400">
              <BookText size={48} className="mx-auto mb-3 opacity-30" />
              No {activeTab === 'stop_phrase' ? 'stop phrases' : 'ignored words'} yet
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
