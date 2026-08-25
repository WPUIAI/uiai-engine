// H5 local-host daemon trust — allow only loopback + Tailscale CGNAT, block public
export function isTrustedHost(url:string){ try{ const u=new URL(url); return u.hostname==="127.0.0.1"||u.hostname==="localhost"||u.hostname.endsWith(".ts.net"); }catch{return false} }
