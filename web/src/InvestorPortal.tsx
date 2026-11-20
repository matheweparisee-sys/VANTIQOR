import { useEffect, useState, type FormEvent } from 'react'

type User = { id: string; name: string; email: string }
type Deposit = { id: string; amount: number; currency: string; method: string; status: string; created_at: string }
const api = async (path: string, options?: RequestInit) => {
  const response = await fetch(`/api/v1${path}`, { credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, ...options })
  const data = await response.json()
  if (!response.ok) throw new Error(data.error || 'Request failed')
  return data
}
export default function InvestorPortal() {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [signup, setSignup] = useState(false)
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [acceptedPrivacy, setAcceptedPrivacy] = useState(false)
  const [showPrivacy, setShowPrivacy] = useState(false)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [amount, setAmount] = useState('')
  const [method, setMethod] = useState('Bank transfer')
  const [message, setMessage] = useState('')
  const [deposits, setDeposits] = useState<Deposit[]>([])
  const [confirmation, setConfirmation] = useState('')
  useEffect(() => { api('/auth/me').then(d => setUser(d.user)).catch(() => {}).finally(() => setLoading(false)) }, [])
  useEffect(() => { if (user) api('/my/deposit-applications').then(setDeposits).catch(() => {}) }, [user])
  async function submitAuth(e: FormEvent) {
    e.preventDefault(); setError('')
    if (signup && password !== confirmPassword) { setError('Passwords do not match'); return }
    if (signup && !acceptedPrivacy) { setError('Please review and acknowledge the privacy notice'); return }
    setBusy(true)
    try { const data = await api(signup ? '/auth/signup' : '/auth/login', { method: 'POST', body: JSON.stringify(signup ? { name, email, password } : { email, password }) }); setUser(data.user); setPassword(''); setConfirmPassword('') }
    catch (err) { setError(err instanceof Error ? err.message : 'Unable to sign in') }
    finally { setBusy(false) }
  }
  async function submitDeposit(e: FormEvent) {
    e.preventDefault(); setError('')
    setBusy(true); setConfirmation('')
    try {
      const data = await api('/deposit-applications', { method: 'POST', body: JSON.stringify({ name: user?.name, email: user?.email, amount: Number(amount), currency: 'USD', method, message }) })
      setConfirmation(`Request ${data.application_id} submitted. Status: pending. Your support access code is ${data.access_token}. Save this code securely.`)
      setAmount(''); setMessage(''); setDeposits(await api('/my/deposit-applications'))
    } catch (err) { setError(err instanceof Error ? err.message : 'Could not submit request') }
    finally { setBusy(false) }
  }
  if (showPrivacy) return <main className="min-h-screen bg-[#09090b] text-white p-6"><div className="max-w-2xl mx-auto space-y-5"><button className="text-indigo-400" onClick={()=>setShowPrivacy(false)}>← Back to registration</button><h1 className="text-3xl font-bold">Registration privacy notice</h1><p>VANTIQOR collects your name, email address and signup IP address when you register. The IP address is used for account security, abuse prevention and investigation of suspicious activity.</p><p>Account and security records are accessible only to authorized personnel. A production privacy policy must identify the legal entity, applicable legal bases, retention periods, international transfers and contact details before public registration is enabled.</p><p>This development notice does not replace the final legally reviewed privacy policy.</p></div></main>
  if (loading) return <div className="min-h-screen grid place-items-center">Loading VANTIQOR…</div>
  if (!user) return <div className="min-h-screen flex items-center justify-center p-5 bg-[#09090b] text-white"><div className="w-full max-w-md rounded-2xl border border-white/10 bg-[#15151b] p-8 shadow-2xl">
    <div className="text-indigo-400 font-extrabold tracking-[.24em] text-xl mb-2">VANTIQOR</div>
    <h1 className="text-3xl font-bold mb-2">{signup ? 'Create your account' : 'Welcome back'}</h1>
    <p className="text-zinc-400 mb-8">{signup ? 'Register to access your investor portal.' : 'Sign in to your investor account.'}</p>
    <form onSubmit={submitAuth} className="space-y-4">
      {signup && <label className="block text-sm">Full name<input required minLength={2} maxLength={120} value={name} onChange={e=>setName(e.target.value)} autoComplete="name" className="block w-full mt-2 p-3 rounded-lg bg-zinc-900 border border-white/10 text-white" /></label>}
      <label className="block text-sm">Email address<input required type="email" value={email} onChange={e=>setEmail(e.target.value)} autoComplete="email" className="block w-full mt-2 p-3 rounded-lg bg-zinc-900 border border-white/10 text-white" /></label>
      <label className="block text-sm">Password<input required type="password" minLength={signup ? 12 : 1} maxLength={128} value={password} onChange={e=>setPassword(e.target.value)} autoComplete={signup ? 'new-password' : 'current-password'} className="block w-full mt-2 p-3 rounded-lg bg-zinc-900 border border-white/10 text-white" /></label>
      {signup && <label className="block text-sm">Confirm password<input required type="password" minLength={12} maxLength={128} value={confirmPassword} onChange={e=>setConfirmPassword(e.target.value)} autoComplete="new-password" className="block w-full mt-2 p-3 rounded-lg bg-zinc-900 border border-white/10 text-white" /></label>}
      {signup && <div className="space-y-2 text-xs text-zinc-400"><p>Use at least 12 characters. Your signup IP address is recorded for account security.</p><button type="button" className="text-indigo-400 underline" onClick={()=>setShowPrivacy(true)}>Read registration privacy notice</button><label className="flex gap-2 items-start"><input type="checkbox" checked={acceptedPrivacy} onChange={e=>setAcceptedPrivacy(e.target.checked)} className="mt-0.5" required /><span>I acknowledge the registration privacy notice.</span></label></div>}
      {error && <p role="alert" className="text-red-400 text-sm">{error}</p>}
      <button disabled={busy} className="w-full rounded-lg bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 py-3 font-semibold">{busy ? 'Please wait…' : signup ? 'Create account' : 'Sign in'}</button>
    </form>
    <button className="mt-6 text-sm text-indigo-400" onClick={()=>{setSignup(!signup);setError('')}}>{signup ? 'Already registered? Sign in' : 'New to VANTIQOR? Create an account'}</button>
  </div></div>
  return <div className="min-h-screen bg-[#09090b] text-white p-5 md:p-10"><div className="max-w-5xl mx-auto space-y-8">
    <header className="flex justify-between items-center gap-4 border-b border-white/10 pb-5"><div><div className="text-indigo-400 font-extrabold tracking-[.2em] text-xl">VANTIQOR</div><p className="text-zinc-400 text-sm">Investor portal</p></div><button className="border border-white/20 rounded-lg px-4 py-2" onClick={async()=>{await api('/auth/logout',{method:'POST'});setUser(null);setDeposits([])}}>Sign out</button></header>
    <div><h1 className="text-3xl font-bold">Welcome, {user.name}</h1><p className="text-zinc-400">{user.email}</p></div>
    <div className="grid md:grid-cols-3 gap-4">{[['Account balance','Not available'],['Investment performance','Not available'],['Funding requests',String(deposits.length)]].map(([label,value])=><div key={label} className="rounded-xl bg-[#18181b] border border-white/10 p-6"><p className="text-zinc-400 text-sm">{label}</p><p className="text-xl font-bold mt-3">{value}</p></div>)}</div>
    <div className="grid md:grid-cols-2 gap-6"><section className="rounded-xl bg-[#18181b] border border-white/10 p-6"><h2 className="text-xl font-bold mb-2">Request a deposit</h2><p className="text-zinc-400 text-sm mb-5">This sends a request to the administrator. It does not transfer money or credit your account.</p>
      <form className="space-y-4" onSubmit={submitDeposit}><label className="block text-sm">Amount (USD)<input type="number" required min="1" max="100000000" step="0.01" value={amount} onChange={e=>setAmount(e.target.value)} className="block w-full mt-2 p-3 bg-zinc-900 border border-white/10 rounded-lg" /></label><label className="block text-sm">Preferred method<select value={method} onChange={e=>setMethod(e.target.value)} className="block w-full mt-2 p-3 bg-zinc-900 border border-white/10 rounded-lg"><option>Bank transfer</option><option>Crypto</option><option>Other</option></select></label><label className="block text-sm">Message (optional)<textarea maxLength={2000} value={message} onChange={e=>setMessage(e.target.value)} className="block w-full mt-2 p-3 bg-zinc-900 border border-white/10 rounded-lg" /></label>{error && <p role="alert" className="text-red-400">{error}</p>}{confirmation && <p role="status" className="text-emerald-300 break-words">{confirmation}</p>}<button disabled={busy} className="bg-indigo-600 rounded-lg px-5 py-3 font-semibold disabled:opacity-50">Submit deposit request</button></form>
    </section><section className="rounded-xl bg-[#18181b] border border-white/10 p-6"><div className="flex justify-between gap-2 items-center mb-5"><h2 className="text-xl font-bold">Your deposit requests</h2><button type="button" className="text-sm text-indigo-400 underline" onClick={async()=>{try{setDeposits(await api('/my/deposit-applications'))}catch(err){setError(err instanceof Error?err.message:'Unable to refresh')}}}>Refresh</button></div>{deposits.length===0?<p className="text-zinc-400">No requests yet.</p>:<div className="space-y-3">{deposits.map(d=><div key={d.id} className="rounded-lg border border-white/10 p-4"><div className="flex justify-between gap-3"><strong>{d.currency} {d.amount.toLocaleString()}</strong><span className="text-amber-300 capitalize">{d.status}</span></div><p className="text-xs text-zinc-400 mt-2">{d.method} · {d.created_at}</p></div>)}</div>}</section></div>
    <p className="text-zinc-500 text-sm">VANTIQOR investor access is in development. Investment services and real-money account crediting are not active.</p>
  </div></div>
}
