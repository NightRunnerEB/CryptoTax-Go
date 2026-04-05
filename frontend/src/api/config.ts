function normalizeBaseUrl(value: string): string {
  return value.replace(/\/+$/, '')
}

export const API_CONFIG = {
  gatewayBaseUrl: normalizeBaseUrl(import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'),
}
