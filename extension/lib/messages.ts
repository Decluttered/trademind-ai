export type CaptureRequest={type:'MINDBAY_CAPTURE';url:string;idempotencyKey:string};
export type ConfigureRequest={type:'MINDBAY_CONFIGURE';token:string;apiBase:string};
export type ExtensionMessage=CaptureRequest|ConfigureRequest;
export type ExtensionReply={ok:true;title?:string;snapshotId?:string}|{ok:false;message:string};
export function isAmazonProductURL(raw:string){try{const u=new URL(raw);return(u.hostname==='amazon.de'||u.hostname==='www.amazon.de')&&/\/(?:dp|gp\/product)\/[A-Z0-9]{10}/i.test(u.pathname)}catch{return false}}
