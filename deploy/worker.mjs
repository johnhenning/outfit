// Both sites serve public static content. Never forward credentials upstream.
export default {
  async fetch(request) {
    const incoming = new URL(request.url);
    const general = incoming.hostname === 'getoutfit.dev';
    const dotfiles = incoming.hostname === 'johnhenning.dev' &&
      incoming.pathname === '/tools/dotfiles/install.sh';
    const personal = incoming.hostname === 'johnhenning.dev' &&
      (incoming.pathname === '/tools/outfit' || incoming.pathname.startsWith('/tools/outfit/'));
    if (!general && !personal && !dotfiles) {
      return new Response('Not found', { status: 404 });
    }
    if (request.method !== 'GET' && request.method !== 'HEAD') {
      return new Response('Method not allowed', { status: 405, headers: { Allow: 'GET, HEAD' } });
    }
    if (general && incoming.pathname === '/profiles') {
      return Response.redirect(incoming.origin + '/profiles/' + incoming.search, 308);
    }
    if (personal && incoming.pathname === '/tools/outfit') {
      return Response.redirect(incoming.origin + '/tools/outfit/' + incoming.search, 308);
    }
    if (general && (incoming.pathname === '/tools/outfit' || incoming.pathname.startsWith('/tools/outfit/'))) {
      return Response.redirect(incoming.origin + (incoming.pathname.slice('/tools/outfit'.length) || '/') + incoming.search, 308);
    }
    const upstream = dotfiles
      ? new URL('https://raw.githubusercontent.com/johnhenning/dotfiles/main/install.sh')
      : new URL('https://johnhenning.github.io/');
    if (!dotfiles) {
      let path = general ? incoming.pathname : incoming.pathname.slice('/tools/outfit'.length);
      if (personal) {
        path = path.replace(/^\/(v1\/)?install\.(sh|ps1)$/, '/$1run.$2');
        path = path.replace(/^\/(v1\/)?profiles\/mac\/everyday\.(sh|ps1)$/, '/$1profiles/john-mac-everyday.$2');
        path = path.replace(/^\/(v1\/)?profiles\/mac\/dev\.(sh|ps1)$/, '/$1profiles/john-mac-dev.$2');
      }
      upstream.pathname = (general ? '/outfit/tools/outfit' : '/outfit/personal') + path;
      upstream.search = incoming.search;
    }
    const response = await fetch(upstream.toString(), { method: request.method, redirect: 'manual' });
    const headers = new Headers(response.headers);
    headers.set('X-Content-Type-Options', 'nosniff');
    if (response.ok) headers.set('Cache-Control', 'public, max-age=300');
    return new Response(response.body, { status: response.status, headers });
  }
};
