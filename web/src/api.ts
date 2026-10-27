export type ApiUser = { id: string; username: string; displayName: string; createdAt: string };
export type ApiConversation = { id: string; kind: "direct" | "group" | "channel"; title?: string; createdAt: string };
export type ApiMessage = { id: string; conversationId: string; senderId: string; body: string; createdAt: string };
export type ApiGroupMember = { userId: string; username: string; displayName: string; role: "owner" | "admin" | "member"; joinedAt: string };
export type ApiGroupSettings = { id: string; kind: "group" | "channel"; title: string; isPublic: boolean; currentRole: "owner" | "admin" | "member"; ownerUserId: string; hasInvite: boolean };
export type ApiPublicConversation = { id: string; kind: "group" | "channel"; title: string; isPublic: boolean; memberCount: number; createdAt: string };
async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
 const response=await fetch(path,{...init,credentials:"include",headers:{"Content-Type":"application/json",...init.headers}});
 if(!response.ok){const body=await response.json().catch(()=>({})) as {error?:string};throw new Error(body.error??`Request failed (${response.status})`)}
 if(response.status===204)return undefined as T;return response.json() as Promise<T>;
}
export const api={
 me:()=>request<ApiUser>("/api/v1/auth/me"),
 searchUsers:(q:string)=>request<ApiUser[]>(`/api/v1/users/search?q=${encodeURIComponent(q)}`),
 createDirectConversation:(username:string)=>request<ApiConversation>("/api/v1/conversations/direct",{method:"POST",body:JSON.stringify({username})}),
 register:(username:string,displayName:string,password:string)=>request<ApiUser>("/api/v1/auth/register",{method:"POST",body:JSON.stringify({username,displayName,password})}),
 login:(username:string,password:string)=>request<ApiUser>("/api/v1/auth/login",{method:"POST",body:JSON.stringify({username,password})}),
 logout:()=>request<void>("/api/v1/auth/logout",{method:"POST"}),
 conversations:()=>request<ApiConversation[]>("/api/v1/conversations"),
 createConversation:(kind:"group"|"channel",title:string)=>request<ApiConversation>("/api/v1/conversations",{method:"POST",body:JSON.stringify({kind,title})}),
 groupSettings:(id:string)=>request<ApiGroupSettings>(`/api/v1/conversations/${encodeURIComponent(id)}/settings`),
 groupMembers:(id:string)=>request<ApiGroupMember[]>(`/api/v1/conversations/${encodeURIComponent(id)}/members`),
 setMemberRole:(id:string,userId:string,role:"admin"|"member")=>request<{userId:string;role:string}>(`/api/v1/conversations/${encodeURIComponent(id)}/members/${encodeURIComponent(userId)}`,{method:"PATCH",body:JSON.stringify({role})}),
 transferOwnership:(id:string,userId:string)=>request<{ownerUserId:string}>(`/api/v1/conversations/${encodeURIComponent(id)}/transfer-owner`,{method:"POST",body:JSON.stringify({userId})}),
 setGroupVisibility:(id:string,isPublic:boolean)=>request<{isPublic:boolean}>(`/api/v1/conversations/${encodeURIComponent(id)}/visibility`,{method:"PATCH",body:JSON.stringify({isPublic})}),
 createInvite:(id:string)=>request<{code:string}>(`/api/v1/conversations/${encodeURIComponent(id)}/invite`,{method:"POST",body:JSON.stringify({})}),
 joinInvite:(code:string)=>request<ApiConversation>(`/api/v1/invites/${encodeURIComponent(code)}/join`,{method:"POST",body:JSON.stringify({})}),
 publicConversations:()=>request<ApiPublicConversation[]>("/api/v1/conversations/public"),
 messages:(id:string)=>request<ApiMessage[]>(`/api/v1/conversations/${encodeURIComponent(id)}/messages`),
 sendMessage:(id:string,body:string)=>request<ApiMessage>(`/api/v1/conversations/${encodeURIComponent(id)}/messages`,{method:"POST",body:JSON.stringify({body})}),
};
