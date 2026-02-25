import { Settings as SettingsIcon } from 'lucide-react';

export default function Settings() {
  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold text-gray-900">Settings</h2>

      <div className="bg-white rounded-xl border border-gray-200 p-6">
        <div className="flex items-center gap-4 mb-6">
          <div className="w-12 h-12 bg-primary-100 rounded-xl flex items-center justify-center">
            <SettingsIcon size={24} className="text-primary-600" />
          </div>
          <div>
            <h3 className="font-semibold text-gray-900">TG-Spam Corporate Edition</h3>
            <p className="text-sm text-gray-500">Anti-Spam Management System</p>
          </div>
        </div>

        <div className="grid gap-3">
          <InfoRow label="Version" value="1.0.0-corp" />
          <InfoRow label="Database" value="SQLite" />
          <InfoRow label="API Version" value="v2" />
        </div>
      </div>
    </div>
  );
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between py-2 border-b border-gray-100 last:border-0">
      <span className="text-sm text-gray-500">{label}</span>
      <span className="text-sm font-medium text-gray-900">{value}</span>
    </div>
  );
}
