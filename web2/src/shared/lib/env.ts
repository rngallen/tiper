/** Build-time environment (Vite `import.meta.env`). Only VITE_* keys reach the bundle. */
export const env = {
  appName: import.meta.env.VITE_APP_NAME || "TIPER DFMS",
  apiBase: "/api/v1",
  isDev: import.meta.env.DEV,
  isTest: import.meta.env.MODE === "test",
} as const;
