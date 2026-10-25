import {test} from 'node:test';
import assert from 'node:assert/strict';
import worker from '../deploy/worker.mjs';
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
