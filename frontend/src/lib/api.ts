export async function apiFetch<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = typeof window !== 'undefined' ? localStorage.getItem('tt_auth_token') : null;
  const headers = new Headers(options.headers || {});
  
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  if (!headers.has('Content-Type') && options.body && typeof options.body === 'string') {
    headers.set('Content-Type', 'application/json');
  }

  const res = await fetch(path, {
    ...options,
    headers,
  });

  if (!res.ok) {
    if (res.status === 401 && typeof window !== 'undefined') {
      window.dispatchEvent(new CustomEvent('tt-auth-required'));
    }
    const errText = await res.text();
    throw new Error(`API Error ${res.status}: ${errText}`);
  }

  return res.json() as Promise<T>;
}

export function subscribeEvents(onMessage: (event: { type: string; data: any }) => void): () => void {
  if (typeof window === 'undefined') return () => {};

  const token = localStorage.getItem('tt_auth_token');
  const url = token ? `/events?token=${encodeURIComponent(token)}` : '/events';
  const es = new EventSource(url);

  es.onmessage = (e) => {
    try {
      const data = JSON.parse(e.data);
      onMessage({ type: 'message', data });
    } catch {
      // Non-JSON or comment ping
    }
  };

  es.addEventListener('session.created', (e: any) => {
    try {
      const data = JSON.parse(e.data);
      onMessage({ type: 'session.created', data });
    } catch {}
  });

  es.addEventListener('session.updated', (e: any) => {
    try {
      const data = JSON.parse(e.data);
      onMessage({ type: 'session.updated', data });
    } catch {}
  });

  es.addEventListener('scan.progress', (e: any) => {
    try {
      const data = JSON.parse(e.data);
      onMessage({ type: 'scan.progress', data });
    } catch {}
  });

  return () => {
    es.close();
  };
}
