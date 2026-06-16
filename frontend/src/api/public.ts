import { apiClient, unwrap } from './client';

export interface PaymentProviderOption {
  label: string;
  value: string;
}

export const publicApi = {
  settings() {
    return unwrap<Record<string, string>>(apiClient.get('/public/settings'));
  },
  paymentProviders() {
    return unwrap<PaymentProviderOption[]>(apiClient.get('/public/payment-providers'));
  }
};
