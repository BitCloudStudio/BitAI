import { defineStore } from 'pinia';
import { publicApi, type PaymentProviderOption } from '../api/public';

interface PublicConfigState {
  loaded: boolean;
  settings: Record<string, string>;
  paymentProviders: PaymentProviderOption[];
}

export const usePublicConfigStore = defineStore('publicConfig', {
  state: (): PublicConfigState => ({
    loaded: false,
    settings: {},
    paymentProviders: []
  }),
  getters: {
    enabled: (state) => (key: string, fallback = true) => {
      const value = state.settings[key];
      if (value === undefined || value === '') return fallback;
      return ['1', 'true', 'yes', 'on', 'enabled'].includes(String(value).toLowerCase());
    },
    text: (state) => (key: string, fallback = '') => state.settings[key] || fallback
  },
  actions: {
    async load() {
      const [settings, paymentProviders] = await Promise.all([publicApi.settings(), publicApi.paymentProviders()]);
      this.settings = settings;
      this.paymentProviders = paymentProviders;
      this.loaded = true;
    }
  }
});
