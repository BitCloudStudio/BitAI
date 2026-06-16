import { defineStore } from 'pinia';
import { authApi, type AuthPayload } from '../api/auth';
import { userApi, type UpdateProfilePayload } from '../api/user';
import type { User } from '../types';

const ACCESS_TOKEN_KEY = 'bitapi.access_token';
const REFRESH_TOKEN_KEY = 'bitapi.refresh_token';
const USER_KEY = 'bitapi.user';

interface AuthState {
  user: User | null;
  accessToken: string;
  refreshToken: string;
}

function readStoredUser() {
  const raw = localStorage.getItem(USER_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as User;
  } catch {
    localStorage.removeItem(USER_KEY);
    return null;
  }
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    user: readStoredUser(),
    accessToken: localStorage.getItem(ACCESS_TOKEN_KEY) || '',
    refreshToken: localStorage.getItem(REFRESH_TOKEN_KEY) || ''
  }),
  getters: {
    isAuthenticated: (state) => Boolean(state.accessToken),
    isAdmin: (state) => ['owner', 'admin', 'operator'].includes(state.user?.role || '')
  },
  actions: {
    persist(accessToken: string, refreshToken: string) {
      this.accessToken = accessToken;
      this.refreshToken = refreshToken;
      localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
      localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
    },
    setUser(user: User | null) {
      this.user = user;
      if (user) {
        localStorage.setItem(USER_KEY, JSON.stringify(user));
      } else {
        localStorage.removeItem(USER_KEY);
      }
    },
    syncFromStorage() {
      this.accessToken = localStorage.getItem(ACCESS_TOKEN_KEY) || '';
      this.refreshToken = localStorage.getItem(REFRESH_TOKEN_KEY) || '';
      this.user = this.accessToken ? readStoredUser() : null;
    },
    async login(payload: AuthPayload) {
      const pair = await authApi.login(payload);
      this.persist(pair.access_token, pair.refresh_token);
      this.setUser(pair.user);
    },
    async register(payload: AuthPayload) {
      const pair = await authApi.register(payload);
      this.persist(pair.access_token, pair.refresh_token);
      this.setUser(pair.user);
    },
    async loadMe() {
      if (!this.accessToken) return;
      this.setUser(await authApi.me());
    },
    async updateProfile(payload: UpdateProfilePayload) {
      this.setUser(await userApi.updateProfile(payload));
      return this.user;
    },
    logout() {
      this.setUser(null);
      this.accessToken = '';
      this.refreshToken = '';
      localStorage.removeItem(ACCESS_TOKEN_KEY);
      localStorage.removeItem(REFRESH_TOKEN_KEY);
    }
  }
});
