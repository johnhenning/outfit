// The tool namespace is public static content. Never forward credentials upstream.
export default {
  async fetch(request) {
    const incoming = new URL(request.url);
    if (!incoming.pathname.startsWith('/tools/outfit/')) {
      return new Response('Not found', { status: 404 });
    }
    if (request.method !== 'GET' && request.method !== 'HEAD') {
      return new Response('Method not allowed', { status: 405, headers: { Allow: 'GET, HEAD' } });
    }
    const upstream = new URL('https://johnhenning.github.io/');
    upstream.pathname = '/outfit' + incoming.pathname;
    upstream.search = incoming.search;
    const response = await fetch(upstream.toString(), { method: request.method, redirect: 'manual' });
    const headers = new Headers(response.headers);
    headers.set('X-Content-Type-Options', 'nosniff');
    if (response.ok) headers.set('Cache-Control', 'public, max-age=300');
    return new Response(response.body, { status: response.status, headers });
  }
};
