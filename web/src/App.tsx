import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import { Archive, ArrowRightLeft, Bell, Check, CircleHelp, Copy, Globe, Hash, LockKeyhole, LogOut, MessageCircle, MoreHorizontal, Paperclip, Plus, Search, Send, Settings, ShieldCheck, Smile, Users } from "lucide-react";
import { api, ApiConversation, ApiGroupMember, ApiGroupSettings, ApiMessage, ApiUser } from "./api";

type Chat = { id: string; name: string; preview: string; time: string; kind: "direct" | "group" | "channel"; color: string };
const palette = ["#477da8", "#8a6dc1", "#3b9d89", "#bb7954", "#478eaa"];
const timeLabel = (value: string) => new Date(value).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
const toChat = (c: ApiConversation): Chat => ({ id: c.id, name: c.title || (c.kind === "direct" ? "Личная переписка" : "Без названия"), preview: "Откройте переписку", time: timeLabel(c.createdAt), kind: c.kind, color: palette[c.id.charCodeAt(0) % palette.length] });

export default function App() {
  const [user, setUser] = useState<ApiUser | null>(null);
  const [authMode, setAuthMode] = useState<"login" | "register">("login");
  const [username, setUsername] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [password, setPassword] = useState("");
  const [chats, setChats] = useState<Chat[]>([]);
  const [active, setActive] = useState("");
  const [messages, setMessages] = useState<ApiMessage[]>([]);
  const [search, setSearch] = useState("");
  const [filter, setFilter] = useState<"all" | "direct" | "group" | "channel">("all");
  const [draft, setDraft] = useState("");
  const [showEmojiPicker, setShowEmojiPicker] = useState(false);
  const [newChatMenu, setNewChatMenu] = useState(false);
  const [newTitle, setNewTitle] = useState("");
  const [newKind, setNewKind] = useState<"group" | "channel">("group");
  const [showCreate, setShowCreate] = useState(false);
  const [showUserSearch, setShowUserSearch] = useState(false);
  const [userQuery, setUserQuery] = useState("");
  const [userResults, setUserResults] = useState<ApiUser[]>([]);
  const [showInfo, setShowInfo] = useState(false);
  const [showMessageSearch, setShowMessageSearch] = useState(false);
  const [messageSearch, setMessageSearch] = useState("");
  const [groupSettings, setGroupSettings] = useState<ApiGroupSettings | null>(null);
  const [groupMembers, setGroupMembers] = useState<ApiGroupMember[]>([]);
  const [inviteCode, setInviteCode] = useState("");
  const [inviteLink, setInviteLink] = useState("");
  const [showJoinInvite, setShowJoinInvite] = useState(false);
  const [joinCode, setJoinCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const current = chats.find(c => c.id === active);
  const visibleMessages = useMemo(() => {
    const query = messageSearch.trim().toLocaleLowerCase();
    return messages.filter(message => !query || message.body.toLocaleLowerCase().includes(query));
  }, [messages, messageSearch]);
  const messageEndRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!showInfo || !current || current.kind === "direct") { setGroupSettings(null); setGroupMembers([]); setInviteCode(""); setInviteLink(""); return; }
    let cancelled = false;
    Promise.all([api.groupSettings(current.id), api.groupMembers(current.id)]).then(([settings, members]) => {
      if (!cancelled) { setGroupSettings(settings); setGroupMembers(members); }
    }).catch(e => { if (!cancelled) setError(e instanceof Error ? e.message : "Не удалось загрузить настройки группы"); });
    return () => { cancelled = true; };
  }, [showInfo, current?.id, current?.kind]);
  const filtered = useMemo(() => {
    const query = search.trim().toLocaleLowerCase();
    return chats.filter(c => (filter === "all" || c.kind === filter) &&
      (!query || c.name.toLocaleLowerCase().includes(query) || c.preview.toLocaleLowerCase().includes(query)));
  }, [chats, filter, search]);

  useEffect(() => {
    messageEndRef.current?.scrollIntoView({ behavior: "smooth", block: "end" });
  }, [messages]);

  async function refreshChats(preferredId?: string) {
    const result = await api.conversations();
    const mapped = result.map(toChat);
    setChats(mapped);
    const next = preferredId || (mapped.some(c => c.id === active) ? active : mapped[0]?.id || "");
    setActive(next);
    if (next) setMessages(await api.messages(next));
    else setMessages([]);
  }

  useEffect(() => {
    api.me().then(async found => { setUser(found); await refreshChats(); }).catch(() => undefined);
  }, []);

  useEffect(() => {
    if (!user) return;
    const code = new URLSearchParams(window.location.search).get("invite");
    if (!code) return;
    let cancelled = false;
    api.joinInvite(code).then(async joined => {
      if (cancelled) return;
      const url = new URL(window.location.href);
      url.searchParams.delete("invite");
      window.history.replaceState({}, "", url.pathname + url.search + url.hash);
      await refreshChats(joined.id);
    }).catch(e => {
      if (!cancelled) setError(e instanceof Error ? e.message : "Не удалось открыть приглашение");
    });
    return () => { cancelled = true; };
  }, [user?.id]);

  // WebSocket delivers new messages immediately; periodic REST refresh is the reconnect fallback.
  useEffect(() => {
    if (!user || !active) return;
    let disposed = false;
    let reconnectTimer: number | undefined;
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    let socket: WebSocket | undefined;
    const connect = () => {
      if (disposed) return;
      socket = new WebSocket(`${protocol}//${window.location.host}/api/v1/ws?conversationId=${encodeURIComponent(active)}`);
      socket.onmessage = event => {
        try {
          const incoming = JSON.parse(String(event.data)) as { type: string; message: ApiMessage };
          if (incoming.type !== "message" || incoming.message.conversationId !== active) return;
          const message = incoming.message;
          setMessages(previous => previous.some(item => item.id === message.id) ? previous : [...previous, message]);
          setChats(previous => {
            const updated = previous.map(chat => chat.id === active ? { ...chat, preview: message.body, time: timeLabel(message.createdAt) } : chat);
            return [...updated.filter(chat => chat.id === active), ...updated.filter(chat => chat.id !== active)];
          });
        } catch { /* ignore malformed events and keep the REST fallback available */ }
      };
      socket.onclose = () => {
        if (!disposed) reconnectTimer = window.setTimeout(connect, 2000);
      };
      socket.onerror = () => socket?.close();
    };
    connect();
    const syncMessages = async () => {
      try {
        const latest = await api.messages(active);
        if (!disposed) setMessages(previous => {
          const ids = new Set(previous.map(item => item.id));
          return [...previous, ...latest.filter(item => !ids.has(item.id))].sort((a,b) => a.createdAt.localeCompare(b.createdAt));
        });
      } catch { /* a temporary network failure should not interrupt typing */ }
    };
    const interval = window.setInterval(() => { void syncMessages(); }, 15000);
    return () => {
      disposed = true;
      window.clearInterval(interval);
      if (reconnectTimer !== undefined) window.clearTimeout(reconnectTimer);
      socket?.close();
    };
  }, [user, active]);

  async function submitAuth(event: FormEvent) {
    event.preventDefault(); setBusy(true); setError("");
    try {
      const found = authMode === "register"
        ? await api.register(username, displayName, password)
        : await api.login(username, password);
      setUser(found); setPassword(""); await refreshChats();
    } catch (e) { setError(e instanceof Error ? e.message : "Не удалось выполнить вход"); }
    finally { setBusy(false); }
  }

  async function selectChat(id: string) {
    setActive(id); setShowInfo(false); setError("");
    try { setMessages(await api.messages(id)); }
    catch (e) { setError(e instanceof Error ? e.message : "Не удалось загрузить сообщения"); }
  }

  async function sendMessage() {
    const value = draft.trim();
    if (!value || !current || busy) return;
    setBusy(true); setError("");
    try {
      const sent = await api.sendMessage(active, value);
      setMessages(prev => prev.some(message => message.id === sent.id) ? prev : [...prev, sent].sort((a, b) => a.createdAt.localeCompare(b.createdAt)));
      setChats(prev => {
        const updated = prev.map(c => c.id === active ? { ...c, preview: value, time: timeLabel(sent.createdAt) } : c);
        return [...updated.filter(c => c.id === active), ...updated.filter(c => c.id !== active)];
      });
      setDraft("");
    } catch (e) { setError(e instanceof Error ? e.message : "Не удалось отправить сообщение"); }
    finally { setBusy(false); }
  }

  async function createConversation(event: FormEvent) {
    event.preventDefault(); if (!newTitle.trim()) return;
    setBusy(true); setError("");
    try {
      const created = await api.createConversation(newKind, newTitle.trim());
      setShowCreate(false); setNewChatMenu(false); setNewTitle("");
      await refreshChats(created.id);
    } catch (e) { setError(e instanceof Error ? e.message : "Не удалось создать чат"); }
    finally { setBusy(false); }
  }

  async function findUsers(event: FormEvent) {
    event.preventDefault();
    if (userQuery.trim().length < 2) { setError("Введите минимум 2 символа для поиска"); return; }
    setBusy(true); setError("");
    try { setUserResults(await api.searchUsers(userQuery.trim())); }
    catch (e) { setError(e instanceof Error ? e.message : "Не удалось найти пользователей"); }
    finally { setBusy(false); }
  }

  async function startDirectChat(target: ApiUser) {
    setBusy(true); setError("");
    try {
      const created = await api.createDirectConversation(target.username);
      setShowUserSearch(false); setUserQuery(""); setUserResults([]);
      await refreshChats(created.id);
    } catch (e) { setError(e instanceof Error ? e.message : "Не удалось открыть личный чат"); }
    finally { setBusy(false); }
  }

  async function refreshGroupDetails() {
    if (!current || current.kind === "direct") return;
    const [settings, members] = await Promise.all([api.groupSettings(current.id), api.groupMembers(current.id)]);
    setGroupSettings(settings); setGroupMembers(members);
  }

  async function changeGroupVisibility(isPublic: boolean) {
    if (!current) return;
    try { await api.setGroupVisibility(current.id, isPublic); await refreshGroupDetails(); }
    catch (e) { setError(e instanceof Error ? e.message : "Не удалось изменить видимость"); }
  }

  async function changeMemberRole(member: ApiGroupMember, role: "admin" | "member") {
    if (!current) return;
    try { await api.setMemberRole(current.id, member.userId, role); await refreshGroupDetails(); }
    catch (e) { setError(e instanceof Error ? e.message : "Не удалось изменить роль участника"); }
  }

  async function transferGroupOwnership(member: ApiGroupMember) {
    if (!current || !window.confirm(`Передать владение группой «${current.name}» пользователю @${member.username}? Вы станете администратором.`)) return;
    try { await api.transferOwnership(current.id, member.userId); await refreshGroupDetails(); }
    catch (e) { setError(e instanceof Error ? e.message : "Не удалось передать владение"); }
  }

  async function createGroupInvite() {
    if (!current) return;
    try {
      const result = await api.createInvite(current.id);
      setInviteCode(result.code);
      const link = new URL(window.location.href); link.searchParams.set("invite", result.code);
      const shareUrl = link.toString(); setInviteLink(shareUrl);
      await refreshGroupDetails();
      try { await navigator.clipboard.writeText(shareUrl); }
      catch { /* the code remains visible so it can be copied manually */ }
    } catch (e) { setError(e instanceof Error ? e.message : "Не удалось создать приглашение"); }
  }

  async function joinByInvite(event: FormEvent) {
    event.preventDefault();
    const rawCode = joinCode.trim();
    if (!rawCode) return;
    let code = rawCode;
    try { const parsed = new URL(rawCode); code = parsed.searchParams.get("invite") || rawCode; } catch { /* treat input as a raw invite code */ }
    setBusy(true); setError("");
    try {
      const joined = await api.joinInvite(code);
      setShowJoinInvite(false); setJoinCode("");
      await refreshChats(joined.id);
    } catch (e) { setError(e instanceof Error ? e.message : "Не удалось вступить по приглашению"); }
    finally { setBusy(false); }
  }

  async function signOut() {
    try { await api.logout(); } catch { /* clear the local view even if the session has expired */ }
    setUser(null); setChats([]); setMessages([]); setActive(""); setShowEmojiPicker(false); setError("");
  }

  if (!user) return <main className="auth-shell">
    <form className="auth-card" onSubmit={submitAuth}>
      <div className="auth-logo"><MessageCircle size={27}/></div>
      <h1>Private Messenger</h1>
      <p>Независимый мессенджер для общения</p>
      <div className="auth-tabs"><button type="button" className={authMode === "login" ? "chosen" : ""} onClick={() => { setAuthMode("login"); setError(""); }}>Войти</button><button type="button" className={authMode === "register" ? "chosen" : ""} onClick={() => { setAuthMode("register"); setError(""); }}>Создать аккаунт</button></div>
      {authMode === "register" && <label>Отображаемое имя<input autoComplete="name" value={displayName} onChange={e => setDisplayName(e.target.value)} maxLength={80} required placeholder="Как вас называть"/></label>}
      <label>Имя пользователя<input autoComplete="username" value={username} onChange={e => setUsername(e.target.value)} minLength={3} maxLength={32} pattern="[A-Za-z0-9_]+" required placeholder="например, alex_123"/></label>
      <label>Пароль<input autoComplete={authMode === "login" ? "current-password" : "new-password"} type="password" value={password} onChange={e => setPassword(e.target.value)} minLength={10} maxLength={72} required placeholder="Не менее 10 символов"/></label>
      {error && <div className="inline-error" role="alert">{error}</div>}
      <button className="auth-submit" type="submit" disabled={busy}>{busy ? "Подождите…" : authMode === "login" ? "Войти" : "Зарегистрироваться"}</button>
      <small>Аккаунты хранятся на сервере. Используйте уникальный пароль.</small>
    </form>
  </main>;

  return <main className="app-shell">
    <aside className="sidebar">
      <header className="side-header"><div className="profile-mark">{user.displayName.slice(0,1).toUpperCase()}</div><div className="search-box"><Search size={17}/><input value={search} onChange={e => setSearch(e.target.value)} placeholder="Поиск чатов"/></div><button className="icon-button" aria-label="Настройки" onClick={() => setShowInfo(v => !v)}><Settings/></button></header>
      <nav className="filter-row" aria-label="Фильтр чатов"><button className={filter === "all" ? "filter active" : "filter"} onClick={() => setFilter("all")}><MessageCircle size={15}/> Все</button><button className={filter === "direct" ? "filter active" : "filter"} onClick={() => setFilter("direct")}><MessageCircle size={15}/> Личные</button><button className={filter === "group" ? "filter active" : "filter"} onClick={() => setFilter("group")}><Users size={15}/> Группы</button><button className={filter === "channel" ? "filter active" : "filter"} onClick={() => setFilter("channel")}><Hash size={15}/> Каналы</button></nav>
      <div className="chat-list">{filtered.map(chat => <button key={chat.id} className={"chat-item " + (active === chat.id ? "selected" : "")} onClick={() => { setShowEmojiPicker(false); void selectChat(chat.id); }}>
        <span className="avatar" style={{ background: chat.color }}>{chat.kind === "group" ? <Users/> : chat.kind === "channel" ? <Hash/> : chat.name.slice(0,1)}</span>
        <span className="chat-copy"><span className="chat-top"><strong>{chat.name}</strong><time>{chat.time}</time></span><span className="chat-bottom"><span>{chat.preview}</span></span></span>
      </button>)}
      {filtered.length === 0 && <div className="list-empty">{search ? "Ничего не найдено" : "Пока нет чатов. Создайте группу или канал."}</div>}</div>
      <div className="compose-wrap"><button className="compose" aria-label="Создать группу или канал" title="Создать группу или канал" onClick={() => setNewChatMenu(v => !v)}><Plus size={23}/></button>
      {newChatMenu && <div className="compose-menu"><button onClick={() => { setShowUserSearch(true); setNewChatMenu(false); setError(""); }}><MessageCircle size={16}/> Личный чат</button><button onClick={() => { setNewKind("group"); setShowCreate(true); setNewChatMenu(false); }}> <Users size={16}/> Создать группу</button><button onClick={() => { setNewKind("channel"); setShowCreate(true); setNewChatMenu(false); }}><Hash size={16}/> Создать канал</button><button onClick={() => { setShowJoinInvite(true); setNewChatMenu(false); setError(""); }}><Users size={16}/> Вступить по коду</button></div>}</div>
      <footer className="sidebar-footer"><span className="online-dot"/> {user.username} <button className="logout-button" onClick={() => void signOut()} title="Выйти" aria-label="Выйти"><LogOut size={15}/></button></footer>
    </aside>
    <section className="conversation">
      {current ? <>
        <header className="conversation-header"><span className="avatar small" style={{ background: current.color }}>{current.kind === "group" ? <Users/> : current.kind === "channel" ? <Hash/> : current.name.slice(0,1)}</span><div className="conversation-title"><strong>{current.name}</strong><span>{current.kind === "channel" ? "Канал" : current.kind === "group" ? "Группа" : "Личная переписка"} · серверная переписка</span></div><div className="header-actions"><button className="icon-button" aria-label="Поиск по сообщениям" aria-pressed={showMessageSearch} onClick={() => { setShowMessageSearch(value => !value); setMessageSearch(""); }}><Search/></button><button className="icon-button" aria-label="Уведомления" title="Уведомления пока не подключены"><Bell/></button><button className="icon-button" aria-label="Информация о чате" onClick={() => setShowInfo(!showInfo)}><MoreHorizontal/></button></div></header>
        {showMessageSearch && <div className="message-search"><Search size={16}/><input autoFocus value={messageSearch} onChange={e => setMessageSearch(e.target.value)} placeholder="Найти в переписке…" aria-label="Поиск по сообщениям"/><span>{messageSearch.trim() ? `${visibleMessages.length} найдено` : `${messages.length} сообщений`}</span><button type="button" onClick={() => { setShowMessageSearch(false); setMessageSearch(""); }} aria-label="Закрыть поиск">×</button></div>}
        <div className="message-area" aria-live="polite"><div className="date-pill">{messageSearch.trim() ? "Результаты поиска" : "История сообщений"}</div>{visibleMessages.map(m => <div className={"message-row " + (m.senderId === user.id ? "own" : "")} key={m.id}><div className="message-bubble">{m.body}<span className="message-meta">{timeLabel(m.createdAt)} {m.senderId === user.id && <Check size={13}/>}</span></div></div> )}{visibleMessages.length === 0 && <div className="welcome"><span className="welcome-icon"><MessageCircle/></span><strong>{messageSearch.trim() ? "Ничего не найдено" : "Начните переписку"}</strong><span>{messageSearch.trim() ? "Попробуйте изменить поисковый запрос." : "Отправьте первое сообщение — история сохранится на вашем сервере."}</span></div>}<div ref={messageEndRef} /></div>
        {error && <div className="chat-error" role="alert">{error}</div>}
        <form className="composer" onSubmit={e => { e.preventDefault(); void sendMessage(); }}><button type="button" className="icon-button" aria-label="Прикрепить файл" title="Загрузка файлов пока не подключена" onClick={() => setError("Загрузка вложений пока не реализована.")}><Paperclip/></button><textarea className="message-input" value={draft} onChange={e => setDraft(e.target.value)} onKeyDown={e => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); void sendMessage(); } }} placeholder="Написать сообщение…" aria-label="Сообщение" maxLength={4000} rows={1}/><div className="emoji-wrap"><button type="button" className="icon-button" aria-label="Эмодзи" aria-expanded={showEmojiPicker} onClick={() => setShowEmojiPicker(v => !v)}><Smile/></button>{showEmojiPicker && <div className="emoji-picker" role="group" aria-label="Выбор эмодзи">{["😀","😁","😂","🥹","😊","😍","😘","😎","😭","😡","👍","👎","❤️","🔥","🎉","🙏","🤔","💯","👋","✨","💙","🥰","😴","🤝"].map(emoji => <button key={emoji} type="button" onClick={() => { setDraft(value => value + emoji); setShowEmojiPicker(false); }}>{emoji}</button>)}</div>}</div><button type="submit" className="send-button" aria-label="Отправить" disabled={busy || !draft.trim()}><Send size={19}/></button></form>
      </> : <div className="empty-state"><MessageCircle size={35}/><strong>Ваши чаты</strong><span>Выберите чат или создайте группу либо канал.</span>{error && <span className="inline-error">{error}</span>}</div>}
    </section>
    {showInfo && <aside className="info-panel"><header><strong>Аккаунт и чат</strong><button className="icon-button" onClick={() => setShowInfo(false)} aria-label="Закрыть">×</button></header><div className="info-avatar" style={{ background: current?.color || "#477da8" }}>{(current?.name || user.displayName).slice(0,1)}</div><h2>{current?.name || user.displayName}</h2><p className="muted">@{user.username}</p>{current && current.kind !== "direct" ? <div className="group-management">
      <div className="group-section-title"><Users size={15}/> Участники · {groupMembers.length}</div>
      {groupMembers.map(member => <div className="group-member" key={member.userId}>
        <span className="result-avatar">{member.displayName.slice(0,1).toUpperCase()}</span>
        <div className="group-member-copy"><strong>{member.displayName}</strong><small>@{member.username} · {member.role === "owner" ? "Владелец" : member.role === "admin" ? "Администратор" : "Участник"}</small>
        {groupSettings?.currentRole === "owner" && member.role !== "owner" && <select aria-label={`Роль @${member.username}`} value={member.role} onChange={e => void changeMemberRole(member, e.target.value as "admin" | "member")}><option value="member">Участник</option><option value="admin">Администратор</option></select>}</div>
        {groupSettings?.currentRole === "owner" && member.userId !== user.id && <button className="mini-icon" title="Передать владение" aria-label={`Передать владение ${member.displayName}`} onClick={() => void transferGroupOwnership(member)}><ArrowRightLeft size={15}/></button>}
      </div>)}
      {groupSettings && <div className="group-visibility"><div><Globe size={15}/><span><strong>{groupSettings.isPublic ? "Публичная" : "Приватная"}</strong><small>{groupSettings.isPublic ? "Видна в списке публичных групп" : "Вступление по приглашению"}</small></span></div><button disabled={groupSettings.currentRole !== "owner"} onClick={() => void changeGroupVisibility(!groupSettings.isPublic)}>{groupSettings.isPublic ? "Сделать приватной" : "Сделать публичной"}</button></div>}
      {groupSettings?.currentRole === "owner" && <button className="invite-button" onClick={() => void createGroupInvite()}><Copy size={15}/> Создать / обновить код приглашения</button>}
      {inviteCode && <div className="invite-code"><small>Ссылка-приглашение (скопирована, если браузер разрешил)</small><code>{inviteLink || inviteCode}</code><button onClick={() => void navigator.clipboard?.writeText(inviteLink || inviteCode).catch(() => undefined)}><Copy size={13}/> Копировать ссылку</button></div>}
    </div> : <div className="info-actions"><button type="button" title="Настройки уведомлений появятся позже"><Bell size={16}/> Уведомления · скоро</button><button type="button" title="Архивирование чатов ещё не реализовано"><Archive size={16}/> Архив · скоро</button><button type="button" title="Политика безопасности сервера"><ShieldCheck size={16}/> Безопасность</button><button type="button" onClick={() => { setShowInfo(false); setError("Для помощи обратитесь к администратору сервера."); }}><CircleHelp size={16}/> Помощь</button></div>}<div className="info-note"><LockKeyhole size={16}/> Сессия защищена HttpOnly-cookie.</div><button className="info-link" onClick={() => void signOut()}><LogOut size={16}/> Выйти из аккаунта</button></aside>}
    {showUserSearch && <div className="modal-backdrop" onClick={() => setShowUserSearch(false)}><section className="create-modal" onClick={e => e.stopPropagation()}><h2>Новый личный чат</h2><p>Найдите пользователя по имени или имени пользователя.</p><form className="user-search-form" onSubmit={e => void findUsers(e)}><input autoFocus value={userQuery} onChange={e => setUserQuery(e.target.value)} maxLength={50} placeholder="Введите имя или username"/><button className="auth-submit" type="submit" disabled={busy || userQuery.trim().length < 2}>{busy ? "Поиск…" : "Найти"}</button></form>{error && <div className="inline-error">{error}</div>}<div className="user-results">{userResults.map(result => <button key={result.id} onClick={() => void startDirectChat(result)} disabled={busy}><span className="result-avatar">{result.displayName.slice(0,1).toUpperCase()}</span><span><strong>{result.displayName}</strong><small>@{result.username}</small></span><MessageCircle size={17}/></button>)}{userResults.length === 0 && <span className="results-hint">Введите поисковый запрос, чтобы найти людей.</span>}</div><div className="modal-actions"><button type="button" onClick={() => setShowUserSearch(false)}>Закрыть</button></div></section></div>}
    {showJoinInvite && <div className="modal-backdrop" onClick={() => setShowJoinInvite(false)}><form className="create-modal" onClick={e => e.stopPropagation()} onSubmit={e => void joinByInvite(e)}><h2>Вступить в группу или канал</h2><p>Вставьте код приглашения, который вам отправил владелец.</p><label>Код приглашения<input autoFocus value={joinCode} onChange={e => setJoinCode(e.target.value)} maxLength={64} required placeholder="Вставьте код приглашения"/></label>{error && <div className="inline-error">{error}</div>}<div className="modal-actions"><button type="button" onClick={() => setShowJoinInvite(false)}>Отмена</button><button className="auth-submit" type="submit" disabled={busy || !joinCode.trim()}>{busy ? "Подключение…" : "Вступить"}</button></div></form></div>}
    {showCreate && <div className="modal-backdrop" onClick={() => setShowCreate(false)}><form className="create-modal" onClick={e => e.stopPropagation()} onSubmit={e => void createConversation(e)}><h2>{newKind === "group" ? "Создать группу" : "Создать канал"}</h2><p>{newKind === "group" ? "Создайте пространство для общения." : "Создайте канал для публикаций."}</p><label>Название<input autoFocus value={newTitle} onChange={e => setNewTitle(e.target.value)} maxLength={120} required placeholder={newKind === "group" ? "Название группы" : "Название канала"}/></label>{error && <div className="inline-error">{error}</div>}<div className="modal-actions"><button type="button" onClick={() => setShowCreate(false)}>Отмена</button><button className="auth-submit" type="submit" disabled={busy || !newTitle.trim()}>{busy ? "Создание…" : "Создать"}</button></div></form></div>}
  </main>;
}
