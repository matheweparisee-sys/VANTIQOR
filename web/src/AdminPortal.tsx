import { useState } from 'react'

type Deposit = { id: string; thread_id: string; name: string; email: string; amount: number; currency: string; method: string; status: string; admin_note: string; created_at: string }
type Thread = { id: string; customer_name: string; customer_email: string; subject: string; status: string; created_at: string; updated_at: string }
type Message = { id: string; thread_id: string; sender: string; content: string; created_at: string }

export default function AdminPortal() {
  const [key, setKey] = useState('')
  const [connected, setConnected] = useState(false)
  const [deposits, setDeposits] = useState<Deposit[]>([])
  const [threads, setThreads] = useState<Thread[]>([])
  const [selected, setSelected] = useState('')
  const [messages, setMessages] = useState<Message[]>([])
  const [reply, setReply] = useState('')
  const [notes, setNotes] = useState<Record<string,string>>({})
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [busy, setBusy] = useState(false)
  async function request(path: string, method = 'GET', payload?: object) {
    const response = await fetch(`/api/v1/admin${path}`, { method, headers: { 'X-Admin-Key': key, ...(payload ? { 'Content-Type': 'application/json' } : {}) }, body: payload ? JSON.stringify(payload) : undefined, cache: 'no-store' })
    const result = await response.json()
    if (!response.ok) throw new Error(result.error || `Request failed (${response.status})`)
    return result
  }
  async function refresh() {
    setError(''); setBusy(true)
    try {
      const [d,t] = await Promise.all([request('/deposit-applications'),request('/support/threads')])
      setDeposits(d); setThreads(t); setConnected(true)
    } catch (e) { setError(e instanceof Error ? e.message : 'Unable to load admin data'); setConnected(false) }
    finally { setBusy(false) }
  }
  async function updateDeposit(id: string, status: string) {
    setError(''); setNotice(''); setBusy(true)
    try {
      await request(`/deposit-applications/${encodeURIComponent(id)}`, 'PATCH', { status, admin_note: notes[id] ?? '' })
      setNotice(`Request ${id} marked ${status}. This does not credit customer funds.`)
      await refresh()
    } catch (e) { setError(e instanceof Error ? e.message : 'Unable to update request') }
    finally { setBusy(false) }
  }
  async function openThread(id: string) {
    setError(''); setSelected(id)
    try { setMessages(await request(`/support/threads/${encodeURIComponent(id)}/messages`)) }
    catch (e) { setError(e instanceof Error ? e.message : 'Unable to load conversation') }
  }
  async function sendReply() {
    if (!reply.trim() || !selected) return
    setBusy(true); setError('')
    try { await request(`/support/threads/${encodeURIComponent(selected)}/messages`, 'POST', { content: reply.trim() }); setReply(''); await openThread(selected) }
    catch (e) { setError(e instanceof Error ? e.message : 'Unable to send reply') }
    finally { setBusy(false) }
  }
  const field = 'w-full rounded-lg border border-white/15 bg-zinc-900 p-3 text-white'
  return <main className="min-h-screen bg-[#09090b] text-white p-5 md:p-10"><div className="mx-auto max-w-6xl space-y-6">
    <header className="flex flex-wrap justify-between items-center gap-3 border-b border-white/10 pb-5"><div><div className="text-xl font-extrabold tracking-[.2em] text-indigo-400">VANTIQOR</div><h1 className="text-2xl font-bold">Administration</h1></div><a className="text-indigo-400 underline" href="/">Investor portal</a></header>
    <p className="text-amber-300 text-sm">Development administration only. Access uses the existing shared admin key until individual administrator accounts and MFA are implemented. Never use this interface to credit customer funds.</p>
    {!connected ? <section className="max-w-lg rounded-xl border border-white/10 bg-zinc-900 p-6 space-y-4"><h2 className="font-semibold text-xl">Administrator access</h2><label className="block text-sm">Admin API key<input className={`${field} mt-2`} type="password" autoComplete="off" value={key} onChange={e=>setKey(e.target.value)} /></label><p className="text-xs text-zinc-400">Key is held only in this page's memory, not local storage.</p><button disabled={!key || busy} onClick={refresh} className="rounded-lg bg-indigo-600 px-5 py-3 disabled:opacity-50">{busy?'Checking…':'Open admin dashboard'}</button></section> : <>
      <div className="flex flex-wrap items-center justify-between gap-3"><div><h2 className="text-xl font-semibold">Deposit request queue</h2><p className="text-sm text-zinc-400">Requests only — approval is not proof of payment or a balance credit.</p></div><div className="flex gap-3"><button onClick={refresh} disabled={busy} className="rounded-lg border border-white/20 px-4 py-2">Refresh</button><button onClick={()=>{setKey('');setConnected(false);setDeposits([]);setThreads([]);setMessages([])}} className="rounded-lg border border-white/20 px-4 py-2">Lock admin</button></div></div>
      <div className="grid gap-3 sm:grid-cols-3">{[['Requests',deposits.length],['Pending',deposits.filter(d=>d.status==='pending').length],['Support threads',threads.length]].map(([label,count])=><div key={label} className="rounded-xl border border-white/10 bg-zinc-900 p-5"><p className="text-zinc-400 text-sm">{label}</p><p className="text-3xl font-bold mt-2">{count}</p></div>)}</div>
      <section className="space-y-3">{deposits.length===0?<p className="text-zinc-400">No deposit requests.</p>:deposits.map(d=><article key={d.id} className="rounded-xl border border-white/10 bg-zinc-900 p-5 space-y-3"><div className="flex flex-wrap justify-between gap-3"><div><h3 className="font-semibold">{d.name} · {d.currency} {d.amount.toLocaleString()}</h3><p className="text-sm text-zinc-400 break-all">{d.email} · {d.method} · {d.created_at}</p><p className="text-xs text-zinc-500 break-all">Request ID: {d.id}</p></div><span className="capitalize text-amber-300">{d.status}</span></div><label className="block text-sm">Admin note<textarea maxLength={2000} className={`${field} mt-2`} value={notes[d.id] ?? d.admin_note ?? ''} onChange={e=>setNotes(p=>({...p,[d.id]:e.target.value}))} /></label><div className="flex flex-wrap gap-2"><button disabled={busy} onClick={()=>updateDeposit(d.id,'approved')} className="rounded-lg bg-indigo-600 px-4 py-2 disabled:opacity-50">Approve request</button><button disabled={busy} onClick={()=>updateDeposit(d.id,'rejected')} className="rounded-lg border border-red-500/50 px-4 py-2 disabled:opacity-50">Reject</button><button disabled={busy} onClick={()=>updateDeposit(d.id,'pending')} className="rounded-lg border border-white/20 px-4 py-2 disabled:opacity-50">Mark pending</button><button onClick={()=>openThread(d.thread_id)} className="rounded-lg border border-white/20 px-4 py-2">View conversation</button></div></article>)}</section>
      <section className="rounded-xl border border-white/10 bg-zinc-900 p-5 space-y-3"><h2 className="text-xl font-semibold">Customer support inbox</h2>{threads.map(t=><button key={t.id} className="block w-full text-left rounded-lg border border-white/10 p-3 hover:bg-white/5" onClick={()=>openThread(t.id)}><strong>{t.subject}</strong><p className="text-sm text-zinc-400">{t.customer_name} · {t.customer_email} · {t.status}</p></button>)}{selected&&<div className="space-y-3 border-t border-white/10 pt-4"><h3 className="font-semibold">Conversation</h3>{messages.map(m=><div key={m.id} className="rounded-lg bg-black/30 p-3"><p className="text-xs text-indigo-300">{m.sender} · {m.created_at}</p><p className="whitespace-pre-wrap break-words">{m.content}</p></div>)}<textarea className={field} maxLength={2000} value={reply} onChange={e=>setReply(e.target.value)} placeholder="Write a reply"/><button disabled={busy||!reply.trim()} onClick={sendReply} className="rounded-lg bg-indigo-600 px-5 py-2 disabled:opacity-50">Send reply</button></div>}</section>
    </>}
    {error&&<p role="alert" className="text-red-400">{error}</p>}{notice&&<p role="status" className="text-emerald-300">{notice}</p>}
  </div></main>
}
