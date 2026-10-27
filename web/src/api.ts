export type ApiUser = { id: string; username: string; displayName: string; createdAt: string };
export type ApiConversation = { id: string; kind: "direct" | "group" | "channel"; title?: string; createdAt: string };
export type ApiMessage = { id: string; conversationId: string; senderId: string; body: string; createdAt: string };
async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
 const response=await fetch(path,{...init,credentials:"include",headers:{"Content-Type":"application/json",...init.headers}});
 if(!response.ok){const body=await response.json().catch(()=>({})) as {error?:string};throw new Error(body.error??`Request failed (${response.status})`)}
 if(response.status===204)return undefined as T;return response.json() as Promise<T>;
}
export const api={
 me:()=>request<ApiUser>("/api/v1/auth/me"),
 register:(username:string,displayName:string,password:string)=>request<ApiUser>("/api/v1/auth/register",{method:"POST",body:JSON.stringify({username,displayName,password})}),
 login:(username:string,password:string)=>request<ApiUser>("/api/v1/auth/login",{method:"POST",body:JSON.stringify({username,password})}),
 logout:()=>request<void>("/api/v1/auth/logout",{method:"POST"}),
 conversations:()=>request<ApiConversation[]>("/api/v1/conversations"),
 createConversation:(kind:"group"|"channel",title:string)=>request<ApiConversation>("/api/v1/conversations",{method:"POST",body:JSON.stringify({kind,title})}),
 messages:(id:string)=>request<ApiMessage[]>(`/api/v1/conversations/${encodeURIComponent(id)}/messages`),
 sendMessage:(id:string,body:string)=>request<ApiMessage>(`/api/v1/conversations/${encodeURIComponent(id)}/messages`,{method:"POST",body:JSON.stringify({body})}),
};
