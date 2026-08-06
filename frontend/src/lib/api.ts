import axios from "axios";
import { useAuthStore } from "@/stores/auth-store";

export const api = axios.create({ baseURL: "/api/v1" });

api.interceptors.request.use((config) => {
  const token = useAuthStore.getState().accessToken;
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

let refreshing: Promise<string | null> | null = null;

api.interceptors.response.use(
  (res) => res,
  async (error) => {
    const original = error.config;
    if (error.response?.status !== 401 || original._retried) {
      throw error;
    }
    original._retried = true;

    refreshing ??= refreshAccessToken();
    const token = await refreshing;
    refreshing = null;

    if (!token) {
      useAuthStore.getState().clear();
      throw error;
    }

    original.headers.Authorization = `Bearer ${token}`;
    return api(original);
  },
);

async function refreshAccessToken(): Promise<string | null> {
  const refreshToken = useAuthStore.getState().refreshToken;
  if (!refreshToken) return null;

  try {
    const { data } = await axios.post("/api/v1/auth/refresh", { refresh_token: refreshToken });
    useAuthStore.getState().setTokens(data.data.access_token, data.data.refresh_token);
    return data.data.access_token as string;
  } catch {
    return null;
  }
}
