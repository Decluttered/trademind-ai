import type { ExtensionMessage,ExtensionReply } from '../lib/messages';

export default defineBackground(()=>{
  const session=browser.storage.session;
  void session.setAccessLevel?.({accessLevel:'TRUSTED_CONTEXTS'});
  browser.runtime.onMessage.addListener((message:ExtensionMessage,_sender,sendResponse:(reply:ExtensionReply)=>void)=>{
    if(message?.type==='MINDBAY_CONFIGURE'){
      const apiBase=message.apiBase.trim().replace(/\/$/,'');
      if(!/^https:\/\//i.test(apiBase)&&!/^http:\/\/(?:localhost|127\.0\.0\.1)(?::\d+)?$/i.test(apiBase)){sendResponse({ok:false,message:'API-Basis muss HTTPS oder localhost sein.'});return false}
      void session.set({mindbayToken:message.token.trim(),mindbayApiBase:apiBase}).then(()=>sendResponse({ok:true}));return true;
    }
    if(message?.type==='MINDBAY_CAPTURE'){
      void (async()=>{const stored=await session.get(['mindbayToken','mindbayApiBase']);const token=String(stored.mindbayToken||'');const apiBase=String(stored.mindbayApiBase||'http://127.0.0.1:8080');if(!token){sendResponse({ok:false,message:'Extension ist nicht gekoppelt. Öffnen Sie die Optionen.'});return}try{const response=await fetch(`${apiBase}/v1/extension/captures`,{method:'POST',headers:{Authorization:`Bearer ${token}`,'Content-Type':'application/json','Idempotency-Key':message.idempotencyKey},body:JSON.stringify({url:message.url}),credentials:'omit',redirect:'error'});const envelope=await response.json() as {message?:string;data?:{snapshot?:{id:string;title:string}}};if(!response.ok){throw new Error(envelope.message||`Capture HTTP ${response.status}`)};sendResponse({ok:true,title:envelope.data?.snapshot?.title,snapshotId:envelope.data?.snapshot?.id})}catch(error){sendResponse({ok:false,message:error instanceof Error?error.message:String(error)})}})();return true;
    }
    return false;
  });
});
