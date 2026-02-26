import { useState } from 'react';
import { Search, Radio } from 'lucide-react';
import { spamApi } from '@/api/spam';
import { useChannelStore } from '@/store/channelStore';
import type { SpamCheck as SpamCheckType } from '@/types';

export default function SpamCheck() {
  const [text, setText] = useState('');
  const [results, setResults] = useState<SpamCheckType[] | null>(null);
  const [loading, setLoading] = useState(false);
  const selectedGid = useChannelStore((s) => s.selectedGid);

  const handleCheck = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!text.trim()) return;

    setLoading(true);
    try {
      const response = await spamApi.check(text, selectedGid || undefined);
      setResults(response.data.checks || []);
    } catch {
      // handle error
    } finally {
      setLoading(false);
    }
  };

  const isSpam = results?.some((r) => r.spam);

  if (!selectedGid) {
    return (
      <div className="flex flex-col items-center justify-center h-64 text-gray-400">
        <Radio size={48} className="mb-3 opacity-30" />
        <p className="text-sm">Select a channel in the sidebar to check messages</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold text-gray-900">Spam Check</h2>

      <div className="bg-white rounded-xl border border-gray-200 p-6">
        <form onSubmit={handleCheck} className="space-y-4">
          <div>
            <label htmlFor="message" className="block text-sm font-medium text-gray-700 mb-1">
              Message text
            </label>
            <textarea
              id="message"
              value={text}
              onChange={(e) => setText(e.target.value)}
              rows={4}
              className="w-full px-4 py-2.5 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent resize-none"
              placeholder="Enter message text to check..."
            />
          </div>
          <button
            type="submit"
            disabled={loading || !text.trim()}
            className="inline-flex items-center gap-2 bg-primary-600 text-white px-6 py-2.5 rounded-lg text-sm font-medium hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            <Search size={16} />
            {loading ? 'Checking...' : 'Check Message'}
          </button>
        </form>
      </div>

      {results && (
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <div className="flex items-center gap-3 mb-4">
            <div
              className={`w-10 h-10 rounded-xl flex items-center justify-center ${
                isSpam ? 'bg-red-100 text-red-600' : 'bg-green-100 text-green-600'
              }`}
            >
              <Search size={20} />
            </div>
            <div>
              <h3 className="font-semibold text-gray-900">
                {isSpam ? 'Spam Detected' : 'Not Spam'}
              </h3>
              <p className="text-sm text-gray-500">
                {results.filter((r) => r.spam).length} of {results.length} checks triggered
              </p>
            </div>
          </div>

          <div className="space-y-2">
            {results.map((check, i) => (
              <div
                key={i}
                className={`flex items-center justify-between p-3 rounded-lg ${
                  check.spam ? 'bg-red-50' : 'bg-gray-50'
                }`}
              >
                <div>
                  <p className="text-sm font-medium text-gray-900">{check.name}</p>
                  {check.details && (
                    <p className="text-xs text-gray-500 mt-0.5">{check.details}</p>
                  )}
                </div>
                <span
                  className={`px-2.5 py-1 rounded-full text-xs font-medium ${
                    check.spam
                      ? 'bg-red-100 text-red-700'
                      : 'bg-green-100 text-green-700'
                  }`}
                >
                  {check.spam ? 'SPAM' : 'OK'}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
