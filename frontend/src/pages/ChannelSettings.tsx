import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft, Save } from 'lucide-react';
import { channelsApi } from '@/api/channels';
import type { ChannelSettings as ChannelSettingsType } from '@/types';

export default function ChannelSettings() {
  const { gid } = useParams<{ gid: string }>();
  const navigate = useNavigate();
  const [settings, setSettings] = useState<ChannelSettingsType | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    if (!gid) return;
    const fetchSettings = async () => {
      setLoading(true);
      try {
        const response = await channelsApi.getSettings(gid);
        setSettings(response.data);
      } catch {
        // handle error
      } finally {
        setLoading(false);
      }
    };
    fetchSettings();
  }, [gid]);

  const handleSave = async () => {
    if (!gid || !settings) return;
    setSaving(true);
    try {
      await channelsApi.updateSettings(gid, settings);
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    } catch {
      // handle error
    } finally {
      setSaving(false);
    }
  };

  const updateField = <K extends keyof ChannelSettingsType>(key: K, value: ChannelSettingsType[K]) => {
    if (!settings) return;
    setSettings({ ...settings, [key]: value });
  };

  if (loading || !settings) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <button onClick={() => navigate('/channels')} className="p-2 hover:bg-gray-100 rounded-lg">
          <ArrowLeft size={20} />
        </button>
        <h2 className="text-2xl font-bold text-gray-900">Channel Settings: {gid}</h2>
      </div>

      <div className="grid gap-6">
        {/* classifier settings */}
        <Section title="Classifier">
          <NumberField label="Similarity Threshold" value={settings.similarity_threshold}
            onChange={(v) => updateField('similarity_threshold', v)} step={0.1} min={0} max={1} />
          <NumberField label="Min Message Length" value={settings.min_msg_len}
            onChange={(v) => updateField('min_msg_len', v)} />
          <NumberField label="Max Emoji" value={settings.max_emoji}
            onChange={(v) => updateField('max_emoji', v)} min={-1} />
          <NumberField label="Min Spam Probability (%)" value={settings.min_spam_probability}
            onChange={(v) => updateField('min_spam_probability', v)} min={0} max={100} />
          <NumberField label="First Messages Count" value={settings.first_messages_count}
            onChange={(v) => updateField('first_messages_count', v)} min={0} />
          <Toggle label="Paranoid Mode" value={settings.paranoid_mode}
            onChange={(v) => updateField('paranoid_mode', v)} />
        </Section>

        {/* integrations */}
        <Section title="Integrations">
          <Toggle label="CAS Enabled" value={settings.cas_enabled}
            onChange={(v) => updateField('cas_enabled', v)} />
          <Toggle label="OpenAI Enabled" value={settings.openai_enabled}
            onChange={(v) => updateField('openai_enabled', v)} />
          {settings.openai_enabled && (
            <>
              <TextField label="OpenAI Model" value={settings.openai_model}
                onChange={(v) => updateField('openai_model', v)} />
              <Toggle label="OpenAI Veto Mode" value={settings.openai_veto}
                onChange={(v) => updateField('openai_veto', v)} />
            </>
          )}
        </Section>

        {/* meta checks */}
        <Section title="Meta Checks">
          <NumberField label="Links Limit (-1 to disable)" value={settings.meta_links_limit}
            onChange={(v) => updateField('meta_links_limit', v)} min={-1} />
          <Toggle label="Links Only" value={settings.meta_links_only}
            onChange={(v) => updateField('meta_links_only', v)} />
          <Toggle label="Image Only" value={settings.meta_image_only}
            onChange={(v) => updateField('meta_image_only', v)} />
          <Toggle label="Video Only" value={settings.meta_video_only}
            onChange={(v) => updateField('meta_video_only', v)} />
          <Toggle label="Audio Only" value={settings.meta_audio_only}
            onChange={(v) => updateField('meta_audio_only', v)} />
          <Toggle label="Contact Only" value={settings.meta_contact_only}
            onChange={(v) => updateField('meta_contact_only', v)} />
          <Toggle label="Forward Detection" value={settings.meta_forward}
            onChange={(v) => updateField('meta_forward', v)} />
          <Toggle label="Keyboard Detection" value={settings.meta_keyboard}
            onChange={(v) => updateField('meta_keyboard', v)} />
          <Toggle label="Username Symbols" value={settings.meta_username_symbols}
            onChange={(v) => updateField('meta_username_symbols', v)} />
          <Toggle label="Giveaway Detection" value={settings.meta_giveaway}
            onChange={(v) => updateField('meta_giveaway', v)} />
        </Section>

        {/* duplicates */}
        <Section title="Duplicate Detection">
          <NumberField label="Threshold (0 to disable)" value={settings.duplicates_threshold}
            onChange={(v) => updateField('duplicates_threshold', v)} min={0} />
          <TextField label="Window" value={settings.duplicates_window}
            onChange={(v) => updateField('duplicates_window', v)} />
        </Section>

        {/* behavior */}
        <Section title="Behavior">
          <Toggle label="Training Mode" value={settings.training_mode}
            onChange={(v) => updateField('training_mode', v)} />
          <Toggle label="Dry Mode" value={settings.dry_mode}
            onChange={(v) => updateField('dry_mode', v)} />
          <Toggle label="Soft Ban" value={settings.soft_ban}
            onChange={(v) => updateField('soft_ban', v)} />
          <Toggle label="No Spam Reply" value={settings.no_spam_reply}
            onChange={(v) => updateField('no_spam_reply', v)} />
          <Toggle label="Aggressive Cleanup" value={settings.aggressive_cleanup}
            onChange={(v) => updateField('aggressive_cleanup', v)} />
          {settings.aggressive_cleanup && (
            <NumberField label="Cleanup Limit" value={settings.aggressive_cleanup_limit}
              onChange={(v) => updateField('aggressive_cleanup_limit', v)} min={1} />
          )}
          <Toggle label="Suppress Join Message" value={settings.suppress_join_message}
            onChange={(v) => updateField('suppress_join_message', v)} />
          <Toggle label="Delete Join Messages" value={settings.delete_join_messages}
            onChange={(v) => updateField('delete_join_messages', v)} />
          <Toggle label="Delete Leave Messages" value={settings.delete_leave_messages}
            onChange={(v) => updateField('delete_leave_messages', v)} />
        </Section>
      </div>

      {/* save button */}
      <div className="sticky bottom-6">
        <button
          onClick={handleSave}
          disabled={saving}
          className="inline-flex items-center gap-2 bg-primary-600 text-white px-6 py-3 rounded-lg text-sm font-medium hover:bg-primary-700 disabled:opacity-50 transition-colors shadow-lg"
        >
          <Save size={16} />
          {saving ? 'Saving...' : saved ? 'Saved!' : 'Save Settings'}
        </button>
      </div>
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="bg-white rounded-xl border border-gray-200 p-6">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">{title}</h3>
      <div className="grid gap-4">{children}</div>
    </div>
  );
}

function Toggle({ label, value, onChange }: { label: string; value: boolean; onChange: (v: boolean) => void }) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-sm text-gray-700">{label}</span>
      <button
        type="button"
        onClick={() => onChange(!value)}
        className={`relative w-11 h-6 rounded-full transition-colors ${
          value ? 'bg-primary-600' : 'bg-gray-200'
        }`}
      >
        <span className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full shadow transition-transform ${
          value ? 'translate-x-5' : 'translate-x-0'
        }`} />
      </button>
    </div>
  );
}

function NumberField({ label, value, onChange, step = 1, min, max }: {
  label: string; value: number; onChange: (v: number) => void;
  step?: number; min?: number; max?: number;
}) {
  return (
    <div className="flex items-center justify-between gap-4">
      <span className="text-sm text-gray-700">{label}</span>
      <input
        type="number"
        value={value}
        onChange={(e) => onChange(parseFloat(e.target.value) || 0)}
        step={step}
        min={min}
        max={max}
        className="w-32 px-3 py-1.5 border border-gray-300 rounded-lg text-sm text-right focus:outline-none focus:ring-2 focus:ring-primary-500"
      />
    </div>
  );
}

function TextField({ label, value, onChange }: {
  label: string; value: string; onChange: (v: string) => void;
}) {
  return (
    <div className="flex items-center justify-between gap-4">
      <span className="text-sm text-gray-700">{label}</span>
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-48 px-3 py-1.5 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
      />
    </div>
  );
}
