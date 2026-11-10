import { FormEvent, useEffect, useState } from 'react'

const API = `${window.location.port === '5173' ? 'http://127.0.0.1:8899' : ''}/api/v1`
type Ticket = { thread_id: string; access_token: string; subject?: string }
type Message = { id: number; thread_id: string; sender: string; content: string; created_at: string }
type Thread = { id: string; customer_name: string; customer_email: string; subject: string; status: string; updated_at: string }
type Deposit = { id: string; thread_id: string; name: string; email: string; amount: number; currency: string; method: string; status: string; admin_note: string; created_at: string }

async function request<T>(path: string, init: RequestInit = {}, adminKey?: string): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('Content-Type', 'application/json')
  if (adminKey) headers.set('X-Admin-Key', adminKey)
  const response = await fetch(`${API}${path}`, { ...init, headers })
  const data = await response.json().catch(() => ({}))
  if (!response.ok) throw new Error(data.error || `Request failed (${response.status})`)
  return data as T
}

const inputStyle = 'w-full rounded-lg border border-white/10 bg-[#101014] px-3 py-3 text-sm text-zinc-100 outline-none focus:border-indigo-400'
const buttonStyle = 'rounded-lg bg-indigo-500 px-4 py-3 text-sm font-semibold text-white transition hover:bg-indigo-400 disabled:cursor-not-allowed disabled:opacity-50'

export default function FundingSupport() {
  const [mode, setMode] = useState<'deposit' | 'support' | 'admin'>('deposit')
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [amount, setAmount] = useState('')
  const [currency, setCurrency] = useState('USD')
  const [method, setMethod] = useState('Bank transfer')
  const [message, setMessage] = useState('')
  const [subject, setSubject] = useState('Account support')
  const [ticket, setTicket] = useState<Ticket | null>(() => {
    try { const saved = localStorage.getItem('vantiqor-support-ticket'); return saved ? JSON.parse(saved) as Ticket : null } catch { return null }
  })
  const [messages, setMessages] = useState<Message[]>([])
  const [reply, setReply] = useState('')
  const [notice, setNotice] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [adminKey, setAdminKey] = useState('')
  const [threads, setThreads] = useState<Thread[]>([])
  const [deposits, setDeposits] = useState<Deposit[]>([])
  const [selectedThread, setSelectedThread] = useState('')
  const [adminMessages, setAdminMessages] = useState<Message[]>([])
  const [adminReply, setAdminReply] = useState('')

  useEffect(() => {
    if (!ticket) return
    const load = () => request<Message[]>(`/support/threads/${ticket.thread_id}/messages`, { headers: { 'X-Support-Token': ticket.access_token } })
      .then(setMessages).catch(() => setError('Could not load this support conversation. Check your saved support access code.'))
    void load()
    const timer = window.setInterval(() => { void load() }, 10000)
    return () => window.clearInterval(timer)
  }, [ticket])

  async function submitDeposit(e: FormEvent) {
    e.preventDefault(); setBusy(true); setError(''); setNotice('')
    try {
      const result = await request<Ticket & { application_id: string; message: string; status: string }>('/deposit-applications', { method: 'POST', body: JSON.stringify({ name, email, amount: Number(amount), currency, method, message }) })
      const next = { thread_id: result.thread_id, access_token: result.access_token, subject: 'Deposit application' }
      setTicket(next); localStorage.setItem('vantiqor-support-ticket', JSON.stringify(next)); setMessages([])
      setNotice(`Application ${result.application_id} submitted. Status: ${result.status}. Save your support access code in this browser to continue chatting.`)
      setAmount(''); setMessage('')
    } catch (err) { setError(err instanceof Error ? err.message : 'Could not submit application') }
    finally { setBusy(false) }
  }

  async function startSupport(e: FormEvent) {
    e.preventDefault(); setBusy(true); setError(''); setNotice('')
    try {
      const result = await request<Ticket>('/support/threads', { method: 'POST', body: JSON.stringify({ name, email, subject, message }) })
      const next = { ...result, subject }; setTicket(next); localStorage.setItem('vantiqor-support-ticket', JSON.stringify(next)); setMessages([]); setMessage(''); setNotice('Your support chat is open. Our team can reply here.')
    } catch (err) { setError(err instanceof Error ? err.message : 'Could not open support chat') }
    finally { setBusy(false) }
  }

  async function sendCustomerReply(e: FormEvent) {
    e.preventDefault(); if (!ticket || !reply.trim()) return
    setBusy(true); setError('')
    try { await request(`/support/threads/${ticket.thread_id}/messages`, { method: 'POST', headers: { 'X-Support-Token': ticket.access_token }, body: JSON.stringify({ content: reply }) }); setReply(''); const updated = await request<Message[]>(`/support/threads/${ticket.thread_id}/messages`, { headers: { 'X-Support-Token': ticket.access_token } }); setMessages(updated) }
    catch (err) { setError(err instanceof Error ? err.message : 'Could not send message') }
    finally { setBusy(false) }
  }

  async function loadAdmin() {
    setBusy(true); setError(''); setNotice('')
    try {
      const [t, d] = await Promise.all([request<Thread[]>('/admin/support/threads', {}, adminKey), request<Deposit[]>('/admin/deposit-applications', {}, adminKey)])
      setThreads(t); setDeposits(d); sessionStorage.setItem('vantiqor-admin-key', adminKey); setNotice('Admin inbox loaded.')
    } catch (err) { setError(err instanceof Error ? err.message : 'Admin access failed') }
    finally { setBusy(false) }
  }

  async function selectAdminThread(id: string) {
    setSelectedThread(id); setError('')
    try { setAdminMessages(await request<Message[]>(`/admin/support/threads/${id}/messages`, {}, adminKey)) }
    catch (err) { setError(err instanceof Error ? err.message : 'Could not load conversation') }
  }

  async function sendAdminReply(e: FormEvent) {
    e.preventDefault(); if (!selectedThread || !adminReply.trim()) return
    setBusy(true)
    try { await request(`/admin/support/threads/${selectedThread}/messages`, { method: 'POST', body: JSON.stringify({ content: adminReply }) }, adminKey); setAdminReply(''); await selectAdminThread(selectedThread) }
    catch (err) { setError(err instanceof Error ? err.message : 'Could not send reply') }
    finally { setBusy(false) }
  }

  async function updateDeposit(id: string, status: string) {
    setBusy(true); setError('')
    try { await request(`/admin/deposit-applications/${id}`, { method: 'PATCH', body: JSON.stringify({ status }) }, adminKey); await loadAdmin() }
    catch (err) { setError(err instanceof Error ? err.message : 'Could not update application') }
    finally { setBusy(false) }
  }

  return <div className="mx-auto w-full max-w-6xl space-y-5 p-2 pb-10 md:p-6">
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div><p className="text-xs font-semibold uppercase tracking-[.22em] text-indigo-300">VANTIQOR CLIENT SERVICES</p><h1 className="mt-2 text-2xl font-bold text-white md:text-3xl">Funding & support</h1><p className="mt-2 max-w-2xl text-sm text-zinc-400">Submit a deposit application for admin review or contact our support team through a private message thread.</p></div>
      <div className="rounded-xl border border-amber-400/20 bg-amber-400/5 px-4 py-3 text-xs text-amber-200">Applications are requests only. Funds are not credited until verified by an authorized administrator.</div>
    </div>

    <div className="flex flex-wrap gap-2 border-b border-white/10 pb-3">
      {(['deposit','support','admin'] as const).map((item) => <button key={item} onClick={() => { setMode(item); setError(''); setNotice('') }} className={`rounded-lg px-4 py-2 text-sm font-medium capitalize ${mode===item?'bg-indigo-500/15 text-indigo-200':'text-zinc-400 hover:bg-white/5 hover:text-white'}`}>{item==='deposit'?'Deposit application':item==='support'?'Support chat':'Admin inbox'}</button>)}
    </div>

    {notice && <div className="rounded-lg border border-emerald-500/20 bg-emerald-500/10 p-3 text-sm text-emerald-200">{notice}</div>}
    {error && <div className="rounded-lg border border-red-500/20 bg-red-500/10 p-3 text-sm text-red-200">{error}</div>}

    {mode==='deposit' && <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_360px]">
      <form onSubmit={submitDeposit} className="space-y-4 rounded-2xl border border-white/10 bg-[#101014] p-5 md:p-7">
        <div><h2 className="text-lg font-semibold text-white">Apply for a deposit</h2><p className="mt-1 text-sm text-zinc-400">Send your details to the admin team. They will contact you in the support chat to guide the next steps.</p></div>
        <div className="grid gap-3 sm:grid-cols-2"><label className="space-y-1.5 text-xs text-zinc-400">Full name<input required maxLength={120} value={name} onChange={e=>setName(e.target.value)} className={inputStyle} placeholder="Your full name" /></label><label className="space-y-1.5 text-xs text-zinc-400">Email address<input required type="email" maxLength={254} value={email} onChange={e=>setEmail(e.target.value)} className={inputStyle} placeholder="you@example.com" /></label></div>
        <div className="grid gap-3 sm:grid-cols-2"><label className="space-y-1.5 text-xs text-zinc-400">Amount (USD)<input required type="number" min="1" max="100000000" step="0.01" value={amount} onChange={e=>setAmount(e.target.value)} className={inputStyle} placeholder="e.g. 500" /></label><label className="space-y-1.5 text-xs text-zinc-400">Currency<select value={currency} onChange={e=>setCurrency(e.target.value)} className={inputStyle}><option value="USD">USD — US Dollar</option></select></label></div>
        <label className="block space-y-1.5 text-xs text-zinc-400">Preferred deposit method<select value={method} onChange={e=>setMethod(e.target.value)} className={inputStyle}><option>Bank transfer</option><option>Crypto</option><option>Other</option></select></label>
        <label className="block space-y-1.5 text-xs text-zinc-400">Message to admin (optional)<textarea rows={4} maxLength={2000} value={message} onChange={e=>setMessage(e.target.value)} className={inputStyle} placeholder="Add any details the admin team should know…" /></label>
        <button disabled={busy} className={buttonStyle}>{busy?'Submitting…':'Submit deposit application'}</button>
        <p className="text-xs leading-5 text-zinc-500">Do not send passwords, private keys, seed phrases, or full card details. Submitting this form does not transfer money or guarantee approval.</p>
      </form>
      <aside className="space-y-4"><div className="rounded-2xl border border-white/10 bg-[#101014] p-5"><h3 className="font-semibold text-white">What happens next?</h3><ol className="mt-4 space-y-4 text-sm text-zinc-400"><li className="flex gap-3"><span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-indigo-500/15 text-xs text-indigo-200">01</span><span>Submit your deposit application with your contact details and requested amount.</span></li><li className="flex gap-3"><span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-indigo-500/15 text-xs text-indigo-200">02</span><span>Our admin team reviews it and replies through your private support conversation.</span></li><li className="flex gap-3"><span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-indigo-500/15 text-xs text-indigo-200">03</span><span>Your application status is updated after the admin verifies the request.</span></li></ol></div>
      {ticket && <div className="rounded-2xl border border-indigo-400/20 bg-indigo-400/5 p-5"><p className="text-sm font-semibold text-white">Your support conversation is saved on this device</p><p className="mt-1 break-all text-xs text-zinc-400">Reference: {ticket.thread_id}</p><button onClick={()=>setMode('support')} className="mt-3 text-sm font-semibold text-indigo-300 hover:text-indigo-200">Open support chat →</button></div>}</aside>
    </div>}

    {mode==='support' && <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_300px]">
      <div className="flex min-h-[540px] flex-col rounded-2xl border border-white/10 bg-[#101014]">
        <div className="border-b border-white/10 p-5"><h2 className="font-semibold text-white">Support conversation</h2><p className="mt-1 text-xs text-zinc-400">Messages are saved to your private support ticket.</p></div>
        {ticket ? <><div className="flex-1 space-y-3 overflow-y-auto p-5">{messages.map(m=><div key={m.id} className={`max-w-[85%] rounded-xl p-3 text-sm ${m.sender==='admin'?'mr-auto bg-white/5 text-zinc-200':'ml-auto bg-indigo-500/15 text-indigo-100'}`}><p className="mb-1 text-[10px] font-semibold uppercase tracking-wider text-zinc-500">{m.sender==='admin'?'VANTIQOR Support':'You'}</p><p className="whitespace-pre-wrap">{m.content}</p><p className="mt-2 text-right text-[10px] text-zinc-500">{new Date(m.created_at).toLocaleString()}</p></div>)}{messages.length===0&&<p className="text-sm text-zinc-500">Your conversation will appear here.</p>}</div><form onSubmit={sendCustomerReply} className="flex gap-2 border-t border-white/10 p-4"><input value={reply} onChange={e=>setReply(e.target.value)} className={inputStyle} placeholder="Write a message…" maxLength={4000}/><button disabled={busy||!reply.trim()} className={buttonStyle}>Send</button></form></> : <form onSubmit={startSupport} className="space-y-4 p-5"><p className="text-sm text-zinc-400">Start a new chat with our support team.</p><label className="block space-y-1.5 text-xs text-zinc-400">Full name<input required value={name} onChange={e=>setName(e.target.value)} className={inputStyle}/></label><label className="block space-y-1.5 text-xs text-zinc-400">Email<input required type="email" value={email} onChange={e=>setEmail(e.target.value)} className={inputStyle}/></label><label className="block space-y-1.5 text-xs text-zinc-400">Subject<input required value={subject} onChange={e=>setSubject(e.target.value)} className={inputStyle}/></label><label className="block space-y-1.5 text-xs text-zinc-400">Message<textarea required rows={4} value={message} onChange={e=>setMessage(e.target.value)} className={inputStyle}/></label><button disabled={busy} className={buttonStyle}>Start support chat</button></form>}
      </div>
      <aside className="rounded-2xl border border-white/10 bg-[#101014] p-5"><h3 className="font-semibold text-white">Your ticket</h3>{ticket?<><p className="mt-3 text-xs text-zinc-500">Ticket reference</p><p className="mt-1 break-all font-mono text-xs text-indigo-200">{ticket.thread_id}</p><p className="mt-4 text-xs leading-5 text-zinc-400">Your access code is stored in this browser. Keep this device/browser to continue the conversation. Clearing browser storage may remove access.</p><button onClick={()=>{setTicket(null);setMessages([]);localStorage.removeItem('vantiqor-support-ticket');setNotice('This browser has been disconnected from the saved ticket. Keep your reference and access code safe before doing this.')}} className="mt-5 text-xs text-red-300 hover:text-red-200">Disconnect this ticket</button></>:<p className="mt-2 text-sm text-zinc-400">No open ticket on this browser yet.</p>}</aside>
    </div>}

    {mode==='admin' && <div className="space-y-5"><div className="rounded-2xl border border-white/10 bg-[#101014] p-5"><h2 className="font-semibold text-white">Administrator access</h2><p className="mt-1 text-xs text-zinc-400">Enter the private ADMIN_API_KEY configured on your server. Never share this key with customers.</p><div className="mt-4 flex flex-col gap-2 sm:flex-row"><input type="password" value={adminKey} onChange={e=>setAdminKey(e.target.value)} placeholder="Admin API key" className={inputStyle}/><button onClick={loadAdmin} disabled={busy||!adminKey} className={buttonStyle}>Load admin inbox</button></div></div>
      {threads.length>0||deposits.length>0?<div className="grid gap-5 xl:grid-cols-2"><section className="rounded-2xl border border-white/10 bg-[#101014] p-5"><h3 className="mb-4 font-semibold text-white">Deposit applications</h3><div className="space-y-3">{deposits.map(d=><div key={d.id} className="rounded-xl border border-white/10 p-3"><div className="flex flex-wrap items-start justify-between gap-2"><div><p className="font-medium text-white">{d.name} · ${d.amount.toLocaleString('en-US',{minimumFractionDigits:2})} {d.currency}</p><p className="text-xs text-zinc-500">{d.email} · {d.method}</p></div><span className="rounded-full bg-amber-400/10 px-2 py-1 text-[10px] uppercase text-amber-200">{d.status}</span></div><div className="mt-3 flex gap-2"><button onClick={()=>selectAdminThread(d.thread_id)} className="rounded-md border border-white/10 px-3 py-2 text-xs text-zinc-200">Open chat</button><button onClick={()=>updateDeposit(d.id,'approved')} className="rounded-md bg-emerald-500/15 px-3 py-2 text-xs text-emerald-200">Approve</button><button onClick={()=>updateDeposit(d.id,'rejected')} className="rounded-md bg-red-500/15 px-3 py-2 text-xs text-red-200">Reject</button></div></div>)}{deposits.length===0&&<p className="text-sm text-zinc-500">No deposit applications yet.</p>}</div></section>
      <section className="rounded-2xl border border-white/10 bg-[#101014] p-5"><h3 className="mb-4 font-semibold text-white">Support inbox</h3><div className="max-h-72 space-y-2 overflow-y-auto">{threads.map(t=><button key={t.id} onClick={()=>selectAdminThread(t.id)} className={`w-full rounded-lg border p-3 text-left ${selectedThread===t.id?'border-indigo-400/40 bg-indigo-400/10':'border-white/10 hover:bg-white/5'}`}><p className="text-sm font-medium text-white">{t.subject}</p><p className="text-xs text-zinc-400">{t.customer_name} · {t.customer_email}</p><p className="mt-1 text-[10px] uppercase text-zinc-500">{t.status}</p></button>)}</div>{selectedThread&&<><div className="mt-4 max-h-64 space-y-2 overflow-y-auto border-t border-white/10 pt-4">{adminMessages.map(m=><div key={m.id} className="rounded-lg bg-white/5 p-3 text-sm"><p className="mb-1 text-[10px] uppercase text-zinc-500">{m.sender}</p><p className="whitespace-pre-wrap text-zinc-200">{m.content}</p></div>)}</div><form onSubmit={sendAdminReply} className="mt-3 flex gap-2"><input value={adminReply} onChange={e=>setAdminReply(e.target.value)} className={inputStyle} placeholder="Reply to customer…"/><button disabled={busy||!adminReply.trim()} className={buttonStyle}>Reply</button></form></>}</section></div>:null}</div>}
  </div>
}
