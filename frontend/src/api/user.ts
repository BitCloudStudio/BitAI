import { apiClient, unwrap } from './client';
import type { APIKey, PaymentIntent, PaymentOrder, UsageLog, User } from '../types';

export interface CreateKeyPayload {
  name: string;
  group_id?: number;
  quota_limit_micros?: number;
  expires_at?: string;
}

export interface UpdateProfilePayload {
  display_name: string;
  avatar_url?: string;
}

export const userApi = {
  updateProfile(payload: UpdateProfilePayload) {
    return unwrap<User>(apiClient.patch('/user/profile', payload));
  },
  uploadAvatar(file: File) {
    const form = new FormData();
    form.append('file', file);
    return unwrap<{ avatar_url: string }>(
      apiClient.post('/user/avatar', form, {
        headers: { 'Content-Type': 'multipart/form-data' }
      })
    );
  },
  keys() {
    return unwrap<APIKey[]>(apiClient.get('/user/api-keys'));
  },
  createKey(payload: CreateKeyPayload) {
    return unwrap<{ key: string; api_key: APIKey }>(apiClient.post('/user/api-keys', payload));
  },
  deleteKey(id: number) {
    return unwrap<{ deleted: boolean }>(apiClient.delete(`/user/api-keys/${id}`));
  },
  usage() {
    return unwrap<UsageLog[]>(apiClient.get('/user/usage'));
  },
  orders() {
    return unwrap<PaymentOrder[]>(apiClient.get('/user/orders'));
  },
  createOrder(payload: { amount_micros: number; provider?: string }) {
    return unwrap<PaymentIntent>(apiClient.post('/user/orders', payload));
  },
  redeem(code: string) {
    return unwrap(apiClient.post('/user/redeem', { code }));
  }
};
