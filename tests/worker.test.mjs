import {test} from 'node:test';
import assert from 'node:assert/strict';
import worker from '../deploy/worker.mjs';
test('dotfiles overview canonicalizes its URL and serves the dedicated static page',async()=>{
 const redirect=await worker.fetch(new Request('https://johnhenning.dev/tools/dotfiles?from=tools'));
 assert.equal(redirect.status,308);
 assert.equal(redirect.headers.get('Location'),'https://johnhenning.dev/tools/dotfiles/?from=tools');
 const original=globalThis.fetch;
 try{
  for(const method of ['GET','HEAD']){
   globalThis.fetch=async(input,options)=>{
    assert.equal(String(input),'https://johnhenning.github.io/outfit/dotfiles/');
    assert.deepEqual(options,{method,redirect:'manual'});
    return new Response(method==='HEAD'?null:'<h1>John’s dotfiles</h1>',{headers:{'Content-Type':'text/html'}});
   };
   const response=await worker.fetch(new Request('https://johnhenning.dev/tools/dotfiles/?from=tools',{method,headers:{Authorization:'secret',Cookie:'secret'}}));
   assert.equal(response.status,200);
   assert.equal(response.headers.get('Content-Type'),'text/html');
  }
 }finally{globalThis.fetch=original;}
 assert.equal((await worker.fetch(new Request('https://johnhenning.dev/tools/dotfiles/',{method:'POST'}))).status,405);
 assert.equal((await worker.fetch(new Request('https://johnhenning.dev/tools/dotfiles-other'))).status,404);
});
test('serves the standalone dotfiles installer without forwarding credentials or queries',async()=>{
 const original=globalThis.fetch;
 try{
  for(const method of ['GET','HEAD']){
   globalThis.fetch=async(input,options)=>{
    assert.equal(String(input),'https://raw.githubusercontent.com/johnhenning/dotfiles/main/install.sh');
    assert.deepEqual(options,{method,redirect:'manual'});
    return new Response(method==='HEAD'?null:'#!/bin/sh\necho dotfiles\n',{status:200});
   };
   const response=await worker.fetch(new Request('https://johnhenning.dev/tools/dotfiles/install.sh?ignored=1',{method,headers:{Authorization:'secret',Cookie:'session=secret'}}));
   assert.equal(response.status,200);
   assert.equal(response.headers.get('X-Content-Type-Options'),'nosniff');
   assert.equal(response.headers.get('Cache-Control'),'public, max-age=300');
   assert.equal(await response.text(),method==='HEAD'?'':'#!/bin/sh\necho dotfiles\n');
  }
 }finally{globalThis.fetch=original;}
});
test('dotfiles route is restricted to its exact installer and read methods',async()=>{
 assert.equal((await worker.fetch(new Request('https://johnhenning.dev/tools/dotfiles/install.sh',{method:'POST'}))).status,405);
 for(const url of ['https://johnhenning.dev/tools/dotfiles/other.sh','https://johnhenning.dev/tools/dotfiles/install.sh/extra','https://unrelated.dev/tools/dotfiles/install.sh']){
  assert.equal((await worker.fetch(new Request(url))).status,404);
 }
});
test('only serves the Outfit namespace and read methods',async()=>{
 assert.equal((await worker.fetch(new Request('https://johnhenning.dev/'))).status,404);
 assert.equal((await worker.fetch(new Request('https://johnhenning.dev/tools/outfit/run.sh',{method:'POST'}))).status,405);
});
test('maps the general domain without forwarding credentials',async()=>{
 const original=globalThis.fetch;
 try{
  globalThis.fetch=async(input,options)=>{assert.equal(String(input),'https://johnhenning.github.io/outfit/tools/outfit/install.sh');assert.equal(options.method,'GET');assert.equal(options.headers,undefined);return new Response('#!/bin/sh\n',{status:200});};
  const response=await worker.fetch(new Request('https://getoutfit.dev/install.sh',{headers:{Authorization:'secret',Cookie:'session=secret'}}));assert.equal(response.status,200);assert.equal(response.headers.get('X-Content-Type-Options'),'nosniff');assert.equal(await response.text(),'#!/bin/sh\n');
 }finally{globalThis.fetch=original;}
});
test('personal Mac URLs use personal assets, never general profiles',async()=>{
 const original=globalThis.fetch;
 try{
  globalThis.fetch=async(input)=>{assert.equal(String(input),'https://johnhenning.github.io/outfit/personal/profiles/john-mac-dev.sh');return new Response('personal bootstrap');};
  const response=await worker.fetch(new Request('https://johnhenning.dev/tools/outfit/profiles/mac/dev.sh'));
  assert.equal(await response.text(),'personal bootstrap');
 }finally{globalThis.fetch=original;}
});
test('personal install endpoint runs personal setup rather than CLI-only installer',async()=>{
 const original=globalThis.fetch;
 try{
  globalThis.fetch=async(input)=>{assert.equal(String(input),'https://johnhenning.github.io/outfit/personal/run.sh');return new Response('personal setup');};
  const response=await worker.fetch(new Request('https://johnhenning.dev/tools/outfit/install.sh'));
  assert.equal(await response.text(),'personal setup');
 }finally{globalThis.fetch=original;}
});
test('unknown hosts and methods are rejected',async()=>{
 assert.equal((await worker.fetch(new Request('https://unrelated.dev/tools/outfit/install.sh'))).status,404);
 assert.equal((await worker.fetch(new Request('https://getoutfit.dev/install.sh',{method:'POST'}))).status,405);
});
test('legacy general-domain namespace redirects to canonical root paths',async()=>{
 const response=await worker.fetch(new Request('https://getoutfit.dev/tools/outfit/profiles/dev.sh?preview=1'));
 assert.equal(response.status,308);
 assert.equal(response.headers.get('Location'),'https://getoutfit.dev/profiles/dev.sh?preview=1');
});
test('profile browser canonicalizes its directory without dropping platform selection',async()=>{
 const original=globalThis.fetch;
 try {
  globalThis.fetch=async()=>new Response('unexpected upstream request');
  const response=await worker.fetch(new Request('https://getoutfit.dev/profiles?platform=windows-amd64'));
  assert.equal(response.status,308);
  assert.equal(response.headers.get('Location'),'https://getoutfit.dev/profiles/?platform=windows-amd64');
 }finally{globalThis.fetch=original;}
});
