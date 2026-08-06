import { api } from "@/lib/api";
import type { User } from "@/stores/auth-store";

interface AuthResponse {
  access_token: string;
  refresh_token: string;
  user: User;
}

export function login(email: string, password: string) {
  return api.post<{ data: AuthResponse }>("/auth/login", { email, password }).then((r) => r.data.data);
}

export function register(name: string, email: string, password: string) {
  return api.post<{ data: AuthResponse }>("/auth/register", { name, email, password }).then((r) => r.data.data);
}

export function logout() {
  return api.post("/auth/logout");
}
