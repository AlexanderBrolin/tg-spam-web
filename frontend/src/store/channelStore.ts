import { create } from 'zustand';
import type { Channel } from '@/types';
import { channelsApi } from '@/api/channels';

interface ChannelState {
  channels: Channel[];
  selectedGid: string | null;
  isLoading: boolean;

  fetchChannels: () => Promise<void>;
  selectChannel: (gid: string) => void;
}

export const useChannelStore = create<ChannelState>((set) => ({
  channels: [],
  selectedGid: localStorage.getItem('selected_gid'),
  isLoading: false,

  fetchChannels: async () => {
    set({ isLoading: true });
    try {
      const response = await channelsApi.list();
      const channels = response.data || [];
      set((state) => {
        const selectedGid = state.selectedGid && channels.some(c => c.gid === state.selectedGid)
          ? state.selectedGid
          : channels[0]?.gid || null;
        if (selectedGid) {
          localStorage.setItem('selected_gid', selectedGid);
        }
        return { channels, selectedGid, isLoading: false };
      });
    } catch {
      set({ isLoading: false });
    }
  },

  selectChannel: (gid: string) => {
    localStorage.setItem('selected_gid', gid);
    set({ selectedGid: gid });
  },
}));
