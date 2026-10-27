import { useState } from "react";
import { Archive, Bell, Check, ChevronDown, CircleHelp, File, Hash, LockKeyhole, MessageCircle, MoreHorizontal, Paperclip, Phone, Plus, Search, Send, Settings, ShieldCheck, Smile, Users, Video } from "lucide-react";

type Chat = { id: number; name: string; preview: string; time: string; unread?: number; kind: "direct" | "group" | "channel"; color: string };
const initialChats: Chat[] = [
  { id: 1, name: "Избранное", preview: "Личные заметки и сохранённые сообщения", time: "17:42", kind: "direct", color: "#477da8" },
  { id: 2, name: "Команда", preview: "Добро пожаловать в Private Messenger", time: "17:35", unread: 2, kind: "group", color: "#8a6dc1" },
  { id: 3, name: "Обновления", preview: "Новости и объявления проекта", time: "16:58", kind: "channel", color: "#3b9d89" },
];
export default function App() {
  const [chats, setChats] = useState(initialChats);
  const [active, setActive] = useState(2);
  const [search, setSearch] = useState("");
  const [draft, setDraft] = useState("");
  const [messages, setMessages] = useState<Record<number, string[]>>({ 2: ["Добро пожаловать! Это стартовый интерфейс проекта.", "Пока сообщения хранятся только в состоянии страницы — серверная синхронизация ещё не подключена."] });
  const [showInfo, setShowInfo] = useState(false);
  const current = chats.find(c => c.id === active);
  const filtered = chats.filter(c => c.name.toLowerCase().includes(search.toLowerCase()) || c.preview.toLowerCase().includes(search.toLowerCase()));
  function sendMessage() {
    const value = draft.trim();
    if (!value || !current) return;
    setMessages(prev => ({ ...prev, [active]: [...(prev[active] ?? []), value] }));
    setChats(prev => [{ ...prev.find(c => c.id === active)!, preview: value, time: new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }) }, ...prev.filter(c => c.id !== active)]);
    setDraft("");
  }
  return <main className="app-shell">
    <aside className="sidebar">
      <header className="side-header"><button className="icon-button" aria-label="Меню"><MoreHorizontal /></button><div className="search-box"><Search size={17}/><input value={search} onChange={e => setSearch(e.target.value)} placeholder="Поиск"/></div><button className="icon-button" aria-label="Настройки"><Settings /></button></header>
      <nav className="filter-row"><button className="filter active"><MessageCircle size={15}/> Все чаты</button><button className="filter"><Users size={15}/> Группы</button><button className="filter"><Hash size={15}/> Каналы</button></nav>
      <div className="chat-list">{filtered.map(chat => <button key={chat.id} className={"chat-item " + (active === chat.id ? "selected" : "")} onClick={() => setActive(chat.id)}>
        <span className="avatar" style={{ background: chat.color }}>{chat.kind === "group" ? <Users/> : chat.kind === "channel" ? <Hash/> : chat.name.slice(0, 1)}</span>
        <span className="chat-copy"><span className="chat-top"><strong>{chat.name}</strong><time>{chat.time}</time></span><span className="chat-bottom"><span>{chat.preview}</span>{chat.unread ? <i>{chat.unread}</i> : null}</span></span>
      </button>)}</div>
      <button className="compose" aria-label="Создать чат" title="Создать чат" onClick={() => { const id = Date.now(); const chat: Chat = { id, name: "Новый чат", preview: "Черновик нового чата", time: "сейчас", kind: "direct", color: "#477da8" }; setChats(prev => [chat, ...prev]); setActive(id); }}> <Plus size={23}/></button>
      <footer className="sidebar-footer"><span className="online-dot"/> Независимый мессенджер <span className="version">v0.1</span></footer>
    </aside>
    <section className="conversation">
      {current ? <>
        <header className="conversation-header"><span className="avatar small" style={{ background: current.color }}>{current.kind === "group" ? <Users/> : current.kind === "channel" ? <Hash/> : current.name.slice(0,1)}</span><div className="conversation-title"><strong>{current.name}</strong><span>{current.kind === "channel" ? "Канал" : current.kind === "group" ? "Группа" : "Личная переписка"} · локальный демонстрационный режим</span></div><div className="header-actions"><button className="icon-button" aria-label="Аудиозвонок"><Phone/></button><button className="icon-button" aria-label="Видеозвонок"><Video/></button><button className="icon-button" aria-label="Уведомления"><Bell/></button><button className="icon-button" aria-label="Информация о чате" onClick={() => setShowInfo(!showInfo)}><MoreHorizontal/></button></div></header>
        <div className="message-area"><div className="date-pill">Сегодня</div>{(messages[active] ?? []).map((message, index) => <div className="message-row own" key={index}><div className="message-bubble">{message}<span className="message-meta">{new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })} <Check size={13}/></span></div></div>)}{(messages[active] ?? []).length === 0 && <div className="welcome"><span className="welcome-icon"><MessageCircle/></span><strong>Начните переписку</strong><span>Это демонстрационный интерфейс. Для реальных аккаунтов и доставки сообщений нужно подключить backend.</span></div>}</div>
        <form className="composer" onSubmit={e => { e.preventDefault(); sendMessage(); }}><button type="button" className="icon-button" aria-label="Прикрепить файл"><Paperclip/></button><input value={draft} onChange={e => setDraft(e.target.value)} placeholder="Написать сообщение…" aria-label="Сообщение"/><button type="button" className="icon-button" aria-label="Эмодзи"><Smile/></button><button type="submit" className="send-button" aria-label="Отправить"><Send size={19}/></button></form>
      </> : <div className="empty-state">Выберите чат слева</div>}
    </section>
    {showInfo && current && <aside className="info-panel"><header><strong>Информация о чате</strong><button className="icon-button" onClick={() => setShowInfo(false)} aria-label="Закрыть">×</button></header><div className="info-avatar" style={{ background: current.color }}>{current.name.slice(0,1)}</div><h2>{current.name}</h2><p className="muted">Демо-профиль · сервер ещё не подключён</p><div className="info-actions"><button><Bell size={16}/> Уведомления</button><button><Archive size={16}/> Архивировать</button><button><ShieldCheck size={16}/> Конфиденциальность</button><button><CircleHelp size={16}/> Помощь</button></div><div className="info-note"><LockKeyhole size={16}/> Не отправляйте настоящие секреты: это пока только интерфейс.</div><div className="info-note"><File size={16}/> Файлы и звонки ещё не подключены.</div><button className="add-member"><Plus size={16}/> Добавить участника (позже)</button><button className="info-link"><ChevronDown size={16}/> Дополнительные настройки</button></aside>}
  </main>;
}
