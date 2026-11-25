// Resolve a file in public/ against the deploy base path.
const base = import.meta.env.BASE_URL.replace(/\/?$/, '/');

export const asset = (path: string) => `${base}${path.replace(/^\//, '')}`;
