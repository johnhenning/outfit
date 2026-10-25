import {test} from 'node:test';
import assert from 'node:assert/strict';
import worker from '../deploy/worker.mjs';
test('only serves the Outfit namespace and read methods',async()=>{
 assert.equal((await worker.fetch(new Request('https://johnhenning.dev/'))).status,404);
 assert.equal((await worker.fetch(new Request('https://johnhenning.dev/tools/outfit/run.sh',{method:'POST'}))).status,405);
});
test('maps the public site without forwarding credentials',async()=>{
 const original=globalThis.fetch;
 try{
  globalThis.fetch=async(input,options)=>{assert.equal(String(input),'https://johnhenning.github.io/outfit/tools/outfit/install.sh');assert.equal(options.method,'GET');assert.equal(options.headers,undefined);return new Response('#!/bin/sh\n',{status:200});};
  const response=await worker.fetch(new Request('https://johnhenning.dev/tools/outfit/install.sh',{headers:{Authorization:'secret',Cookie:'session=secret'}}));assert.equal(response.status,200);assert.equal(response.headers.get('X-Content-Type-Options'),'nosniff');assert.equal(await response.text(),'#!/bin/sh\n');
 }finally{globalThis.fetch=original;}
});
