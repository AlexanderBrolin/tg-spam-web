import { useEffect, useState } from 'react';
import { Plus, Trash2, BookOpen } from 'lucide-react';
import { samplesApi } from '@/api/samples';

export default function Samples() {
  const [spamSamples, setSpamSamples] = useState<string[]>([]);
  const [hamSamples, setHamSamples] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<'spam' | 'ham'>('spam');
  const [newSample, setNewSample] = useState('');

  const fetchData = async () => {
    setLoading(true);
    try {
      const response = await samplesApi.getAll();
      setSpamSamples(response.data.spam || []);
      setHamSamples(response.data.ham || []);
    } catch {
      // handle error
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newSample.trim()) return;
    try {
      if (activeTab === 'spam') {
        await samplesApi.addSpam(newSample);
        setSpamSamples((prev) => [newSample, ...prev]);
      } else {
        await samplesApi.addHam(newSample);
        setHamSamples((prev) => [newSample, ...prev]);
      }
      setNewSample('');
    } catch {
      // handle error
    }
  };

  const handleDelete = async (message: string) => {
    if (!confirm('Delete this sample?')) return;
    try {
      if (activeTab === 'spam') {
        await samplesApi.deleteSpam(message);
        setSpamSamples((prev) => prev.filter((s) => s !== message));
      } else {
        await samplesApi.deleteHam(message);
        setHamSamples((prev) => prev.filter((s) => s !== message));
      }
    } catch {
      // handle error
    }
  };

  const currentSamples = activeTab === 'spam' ? spamSamples : hamSamples;

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-gray-900">Samples</h2>
        <p className="text-sm text-gray-500 mt-1">
          Training data for the spam classifier. Spam samples teach the bot to recognize spam, ham samples — legitimate messages.
        </p>
      </div>

      {/* tabs */}
      <div className="flex gap-2">
        <button
          onClick={() => setActiveTab('spam')}
          className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
            activeTab === 'spam'
              ? 'bg-red-100 text-red-700'
              : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
          }`}
        >
          Spam ({spamSamples.length})
        </button>
        <button
          onClick={() => setActiveTab('ham')}
          className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
            activeTab === 'ham'
              ? 'bg-green-100 text-green-700'
              : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
          }`}
        >
          Ham ({hamSamples.length})
        </button>
      </div>

      {/* add form */}
      <div className="bg-white rounded-xl border border-gray-200 p-6">
        <form onSubmit={handleAdd} className="flex gap-4">
          <textarea
            value={newSample}
            onChange={(e) => setNewSample(e.target.value)}
            rows={2}
            className="flex-1 px-4 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 resize-none"
            placeholder={`Add ${activeTab} sample...`}
          />
          <button
            type="submit"
            disabled={!newSample.trim()}
            className="self-end inline-flex items-center gap-2 bg-primary-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-primary-700 disabled:opacity-50 transition-colors"
          >
            <Plus size={16} />
            Add
          </button>
        </form>
      </div>

      {/* list */}
      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        <div className="divide-y divide-gray-200">
          {currentSamples.map((sample, i) => (
            <div key={i} className="flex items-start justify-between p-4 hover:bg-gray-50">
              <p className="text-sm text-gray-700 flex-1 whitespace-pre-wrap">{sample}</p>
              <button
                onClick={() => handleDelete(sample)}
                className="ml-4 text-red-500 hover:text-red-700 shrink-0 transition-colors"
              >
                <Trash2 size={16} />
              </button>
            </div>
          ))}
          {currentSamples.length === 0 && (
            <div className="px-4 py-12 text-center text-gray-400">
              <BookOpen size={48} className="mx-auto mb-3 opacity-30" />
              No {activeTab} samples yet
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
