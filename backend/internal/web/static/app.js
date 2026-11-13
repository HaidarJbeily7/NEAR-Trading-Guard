'use strict';
const $ = id => document.getElementById(id);
const money = value => new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 }).format(value);
const esc = value => String(value).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
let snapshot = null, selectedID = '', busy = false, tokens = [], quoteVersion = 0, statusVersion = 0, quoteExpiryTimer, toastTimer;
const statuses = {
 healthy: ['Healthy', 'safe'], rescue_planned: ['Rescue needed', 'warning'], insufficient_reserve: ['Low reserve', 'danger'],
 liquidatable: ['Liquidation risk', 'danger'], liquidated: ['Liquidated', 'danger'], paused: ['Paused', 'neutral']
};
function statusHTML(status) { const [label, style] = statuses[status] || [status, 'neutral']; return `<span class="status ${style}">${esc(label)}</span>`; }
function selected() { return snapshot?.positions.find(p => p.id === selectedID); }
function toast(message) { clearTimeout(toastTimer); $('toast').textContent = message; $('toast').hidden = false; toastTimer = setTimeout(() => $('toast').hidden = true, 4000); }
function error(id, message = '') { $(id).textContent = message; $(id).hidden = !message; }
async function api(path, body) {
 const controller = new AbortController(), timer = setTimeout(() => controller.abort(), 45000);
 try {
  const response = await fetch(path, { signal: controller.signal, cache: 'no-store', ...(body === undefined ? {} : { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }) });
  if (!response.ok) { const e = new Error((await response.text()).trim() || `Request failed (${response.status})`); e.status = response.status; throw e; }
  return await response.json();
 } catch (e) { if (e.name === 'AbortError') throw new Error('The request timed out. Please retry.'); throw e; }
 finally { clearTimeout(timer); }
}
function accept(data) {
 snapshot = data;
 if (!snapshot.positions.some(p => p.id === selectedID)) selectedID = snapshot.positions[0]?.id || '';
 render();
}
function render() {
 if (!snapshot) return;
 const active = snapshot.positions.filter(p => !p.liquidated);
 $('total-collateral').textContent = money(active.reduce((n,p) => n + p.collateral, 0));
 $('total-reserve').textContent = money(active.reduce((n,p) => n + p.reserve, 0));
 $('lowest-health').textContent = active.length ? Math.min(...active.map(p => p.health)).toFixed(2) : '—';
 $('rescued-total').textContent = money(snapshot.positions.reduce((n,p) => n + p.rescued_usd, 0));
 $('position-count').textContent = active.length;
 $('table-count').textContent = snapshot.positions.length;
 $('nav-count').textContent = snapshot.positions.length;
 $('position-rows').innerHTML = snapshot.positions.map(p => {
  const coinClass = p.symbol === 'ETH' ? 'eth' : p.symbol === 'BTC' ? 'btc' : '';
  const icon = p.symbol === 'NEAR' ? 'И' : p.symbol === 'ETH' ? '♦' : p.symbol === 'BTC' ? '₿' : esc(p.symbol.slice(0,1));
  const color = p.health <= 1 ? '#c5594d' : p.health < p.trigger ? '#c2a252' : '#64a88c';
  return `<tr data-id="${esc(p.id)}" class="${p.id === selectedID ? 'selected' : ''}"><td><button class="asset-select" data-select="${esc(p.id)}" aria-pressed="${p.id === selectedID}"><span class="asset-cell"><span class="coin ${coinClass}">${icon}</span><span><span class="asset-name">${esc(p.name)}</span><span class="asset-subtitle" style="display:block">${esc(p.symbol)} collateral · Demo market</span></span></span></button></td><td class="numeric">${money(p.collateral)}<div class="debt">${money(p.debt)} debt</div></td><td><div class="health-cell"><span class="health-number numeric" style="color:${color}">${p.health.toFixed(2)}</span><div class="mini-track"><div class="mini-fill" style="width:${Math.min(100, p.health / 2.5 * 100)}%;background:${color}"></div></div></div></td><td class="numeric">${money(p.reserve)}</td><td><span class="protection"><span class="dot ${p.auto_protect && !p.paused && !p.liquidated ? 'green' : 'amber'}"></span>${p.liquidated ? 'Closed' : p.paused ? 'Paused' : p.auto_protect ? 'Automatic' : 'Manual'}</span></td><td>${statusHTML(p.status)}</td></tr>`;
 }).join('');
 renderDetail(); renderActivity();
 $('reset-demo').disabled = busy;
}
function renderDetail() {
 const p = selected(); if (!p) return;
 $('selected-symbol').textContent = p.symbol;
 $('health-description').textContent = `${p.name} · ${money(p.debt)} borrowed`;
 $('selected-health').textContent = p.health.toFixed(2);
 const [label, style] = statuses[p.status] || [p.status, 'neutral'];
 $('selected-status').textContent = label; $('selected-status').className = `status ${style}`;
 $('target-value').textContent = p.target.toFixed(2); $('trigger-value').textContent = p.trigger.toFixed(2);
 $('auto-toggle').setAttribute('aria-checked', String(p.auto_protect)); $('auto-toggle').disabled = busy || p.liquidated;
 $('pause-button').textContent = p.paused ? 'Resume guard' : 'Pause guard';
 $('pause-button').disabled = busy || p.liquidated;
 $('crash-button').disabled = busy || p.liquidated;
 $('crash-button').textContent = busy ? 'Applying scenario…' : '↘ Simulate price drop';
 $('rescue-button').disabled = busy || p.status !== 'rescue_planned';
 $('liquidate-button').hidden = p.health > 1 || p.liquidated;
 $('liquidate-button').disabled = busy;
 const notes = {
  healthy: `Guard triggers below ${p.trigger.toFixed(2)}. ${money(p.reserve)} is available to restore health to ${p.target.toFixed(2)}.`,
  rescue_planned: `A ${money(p.rescue_amount)} collateral top-up can restore health to ${p.target.toFixed(2)}.`,
  insufficient_reserve: 'Insufficient reserve to reach the target. The guard will not spend a partial amount.',
  liquidatable: 'The position has crossed 1.00. Rescue cannot be assumed to beat liquidation.',
  liquidated: 'This simulated position is closed. Reset the demo to start again.',
  paused: 'Protection is paused. Price drops will not trigger an automatic rescue.'
 };
 $('scenario-note').textContent = notes[p.status] || '';
 chart(p);
}
function chart(p) {
 const points = p.history;
 const W=660,H=180,left=36,right=16,top=15,bottom=28;
 const max=Math.max(2.1,...points.map(x=>x.health))*1.08,min=Math.min(.7,...points.map(x=>x.health))*.9;
 const y=v=>top+(max-v)/(max-min)*(H-top-bottom);
 const x=i=>left+i/Math.max(1,points.length-1)*(W-left-right);
 const data=points.map((d,i)=>`${x(i)},${y(d.health)}`);
 const path=data.join(' ');
 const area=`${left},${H-bottom} ${path} ${x(points.length-1)},${H-bottom}`;
 const grid=[1,1.5,2].filter(v=>v>=min&&v<=max).map(v=>`<line x1="${left}" x2="${W-right}" y1="${y(v)}" y2="${y(v)}" stroke="${v===1?'#d8a69e':'#edf1ef'}" stroke-dasharray="${v===1?'4 5':'0'}"/><text x="0" y="${y(v)+3}" fill="#9ba8a0" font-size="9">${v.toFixed(1)}</text>`).join('');
 const dots=points.map((d,i)=>`<circle cx="${x(i)}" cy="${y(d.health)}" r="3.5" fill="${d.health<=1?'#c5594d':'#379573'}" stroke="white" stroke-width="2"><title>${esc(d.label)}: ${d.health.toFixed(2)}</title></circle>`).join('');
 const labels=points.map((d,i)=>i===0||i===points.length-1||points.length<=5?`<text x="${x(i)}" y="${H-7}" text-anchor="${i===0?'start':i===points.length-1?'end':'middle'}" fill="#9ba8a0" font-size="8">${esc(d.label)}</text>`:'').join('');
 const annotation=points.length===1?`<text x="${W/2}" y="${H/2}" text-anchor="middle" fill="#94a89b" font-size="10">Simulate a price drop to build the history</text>`:'';
 $('health-chart').innerHTML=`<svg viewBox="0 0 ${W} ${H}" preserveAspectRatio="none" aria-hidden="true"><defs><linearGradient id="area" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#75b994" stop-opacity=".14"/><stop offset="100%" stop-color="#75b994" stop-opacity="0"/></linearGradient></defs>${grid}<polygon points="${area}" fill="url(#area)"/><polyline points="${path}" fill="none" stroke="#559f7d" stroke-width="2" stroke-linejoin="round"/>${dots}${labels}${annotation}</svg>`;
 $('health-chart').setAttribute('aria-label', `${p.name} health history: ${points.map(d=>`${d.label} ${d.health.toFixed(2)}`).join(', ')}`);
}
function renderActivity() {
 const icons={success:'✓',warning:'↘',danger:'!',info:'·'};
 $('activity-list').innerHTML=snapshot.events.slice(0,15).map(e=>`<li><span class="event-icon ${esc(e.kind)}">${icons[e.kind]||'·'}</span><div class="event-body"><div class="event-heading"><strong>${esc(e.title)}</strong><time datetime="${esc(e.at)}">${new Date(e.at).toLocaleTimeString([], {hour:'2-digit',minute:'2-digit'})}</time></div><p>${esc(e.detail)}</p></div></li>`).join('');
}
async function act(action) {
 if(busy||!snapshot) return;
 if(action==='reset'&&!confirm('Reset all simulated positions and their activity? Mainnet swaps are unaffected.'))return;
 if(action==='liquidate'&&!confirm('Close this position in the simulation? No real funds will move.'))return;
 busy=true;render();error('global-error');
 try {
  const data=await api('/api/positions/action',{id:selectedID,action,revision:snapshot.revision,drop_percent:Number($('drop').value)});
  accept(data);
  if(action==='crash') {
   const p=selected(); toast(p.history.at(-1)?.label==='Rescued'?'Price drop detected. Position rescued in simulation.':'Scenario applied. Review the position status.');
  } else toast(action==='reset'?'Demo reset.':'Position updated.');
 }catch(e){error('global-error',e.message);if(e.status===409){try{accept(await api('/api/positions'));}catch{}}}
 finally{busy=false;render();}
}
$('position-rows').addEventListener('click',event=>{const row=event.target.closest('[data-id]');if(row&&!busy){selectedID=row.dataset.id;render();}});
$('drop').addEventListener('input',()=>$('drop-output').textContent=`−${$('drop').value}%`);
for(const [id,action] of [['crash-button','crash'],['rescue-button','rescue'],['pause-button','pause'],['auto-toggle','auto'],['liquidate-button','liquidate'],['reset-demo','reset']]) $(id).addEventListener('click',()=>act(action));
$('add-open').addEventListener('click',()=>{error('add-error');$('add-dialog').showModal();});
$('add-close').addEventListener('click',()=>$('add-dialog').close());
$('add-form').addEventListener('submit',async event=>{
 event.preventDefault();if(busy)return;busy=true;render();$('add-submit').disabled=true;error('add-error');
 const data=Object.fromEntries(new FormData(event.target));
 for(const key of ['collateral','debt','threshold','reserve'])data[key]=Number(data[key]);data.threshold/=100;
 try{const result=await api('/api/positions',data);selectedID=result.positions.at(-1).id;accept(result);$('add-dialog').close();event.target.reset();toast('Simulated position added.');}
 catch(e){error('add-error',e.message);}finally{busy=false;$('add-submit').disabled=false;render();}
});
function invalidateQuote(){quoteVersion++;clearTimeout(quoteExpiryTimer);$('quote-result').hidden=true;error('quote-error');}
$('quote-form').addEventListener('input',invalidateQuote);
$('quote-form').addEventListener('change',invalidateQuote);
function toUnits(value,decimals){
 if(!/^\d+(\.\d+)?$/.test(value)||!Number.isInteger(decimals)||decimals<0||decimals>77)throw new Error('Enter a valid positive decimal amount.');
 const [whole,fraction='']=value.split('.');if(fraction.length>decimals)throw new Error(`This asset supports at most ${decimals} decimal places.`);
 const amount=BigInt(whole+fraction.padEnd(decimals,'0'));
 if(amount<=0n||amount.toString().length>78)throw new Error('Amount is outside the supported range.');
 return amount.toString();
}
async function loadTokens(){
 $('retry-tokens').hidden=true;$('quote-button').disabled=true;
 try{
  tokens=await api('/api/intents/tokens');
  if(!tokens.length)throw new Error('No supported NEAR assets returned.');
  const options=tokens.map(t=>`<option value="${esc(t.assetId)}">${esc(t.symbol)} · NEAR</option>`).join('');
  $('origin-asset').innerHTML=options;$('destination-asset').innerHTML=options;
  const near=tokens.find(t=>t.assetId==='nep141:wrap.near'),usdc=tokens.find(t=>t.symbol==='USDC');
  if(near)$('origin-asset').value=near.assetId;if(usdc)$('destination-asset').value=usdc.assetId;
  $('quote-button').disabled=false;error('quote-error');
 }catch(e){error('quote-error',`Live assets unavailable: ${e.message}`);$('retry-tokens').hidden=false;}
}
$('retry-tokens').addEventListener('click',loadTokens);
$('quote-form').addEventListener('submit',async event=>{
 event.preventDefault();invalidateQuote();const version=quoteVersion;
 const origin=tokens.find(t=>t.assetId===$('origin-asset').value),destination=tokens.find(t=>t.assetId===$('destination-asset').value);
 if(!origin||!destination){error('quote-error','Select both assets first.');return;}
 $('quote-button').disabled=true;$('quote-button').textContent='Requesting quote…';
 try{
  const deadline=new Date(Date.now()+5*60*1000).toISOString();
  const result=await api('/api/intents/quote',{dry:true,swapType:'EXACT_INPUT',slippageTolerance:50,originAsset:origin.assetId,depositType:'ORIGIN_CHAIN',destinationAsset:destination.assetId,amount:toUnits($('quote-amount').value.trim(),origin.decimals),recipient:$('quote-account').value.trim(),recipientType:'DESTINATION_CHAIN',refundTo:$('quote-account').value.trim(),refundType:'ORIGIN_CHAIN',deadline});
  if(version!==quoteVersion)return;
  const q=result.quote, minimum=formatUnits(q.minAmountOut,destination.decimals);
  $('quote-result').className='quote-result';$('quote-result').hidden=false;
  $('quote-result').innerHTML=`<div class="quote-tag">LIVE PREVIEW · NO FUNDS MOVED</div><div class="quote-value">${esc(q.amountOutFormatted)} ${esc(destination.symbol)}</div><p>Minimum ${esc(minimum)} ${esc(destination.symbol)} · estimated ${esc(q.timeEstimate)} seconds after deposit confirmation.</p><p>Recipient: ${esc($('quote-account').value.trim())}</p><p>Preview only. Use the local CLI to prepare a deposit quote, then fund it from your wallet.</p>`;
  quoteExpiryTimer=setTimeout(()=>{if(version===quoteVersion){$('quote-result').classList.add('stale');$('quote-result').innerHTML='<strong>Preview expired.</strong><p>Request a fresh quote before planning a transfer.</p>';}},Math.max(0,new Date(deadline)-Date.now()));
 }catch(e){if(version===quoteVersion)error('quote-error',e.message);}finally{$('quote-button').disabled=!tokens.length;$('quote-button').textContent='Get live quote ↗';}
});
function formatUnits(amount,decimals){const value=String(amount).padStart(decimals+1,'0');if(!decimals)return value;return `${value.slice(0,-decimals)}.${value.slice(-decimals)}`.replace(/\.?0+$/,'');}
const swapLabels={PENDING_DEPOSIT:['Awaiting deposit','neutral','No deposit detected. A prepared quote has not moved funds.'],KNOWN_DEPOSIT_TX:['Deposit detected','warning','The provider has detected the deposit transaction.'],PROCESSING:['Swap processing','warning','Settlement is in progress. Do not send a second deposit.'],SUCCESS:['Swap completed','safe','Provider reports delivery. Verify the destination wallet balance and receipt.'],INCOMPLETE_DEPOSIT:['Deposit incomplete','danger','The provider reports an incomplete deposit. Review the saved quote and transaction.'],REFUNDED:['Refund reported','neutral','Provider reports a refund. Verify the refund wallet before retrying.'],FAILED:['Swap failed','danger','A failure does not prove a refund. Review the transaction and provider details.']};
$('status-form').addEventListener('submit',async event=>{
 event.preventDefault();const version=++statusVersion;error('status-error');$('status-button').disabled=true;$('status-button').textContent='Checking…';
 try{
  const params=new URLSearchParams({depositAddress:$('deposit-address').value.trim()});if($('deposit-memo').value.trim())params.set('depositMemo',$('deposit-memo').value.trim());
  const result=await api(`/api/intents/status?${params}`);if(version!==statusVersion)return;
  const [label,style,detail]=swapLabels[result.status]||[`Unknown status: ${result.status}`,'neutral','This state is not treated as success. Check provider details.'];
  $('swap-status').innerHTML=`<div class="swap-result"><span class="status ${style}">${esc(label)}</span><p>${esc(detail)}</p><p>Checked ${esc(new Date().toLocaleTimeString())} · Refresh to get the latest status.</p></div>`;
 }catch(e){if(version===statusVersion){$('swap-status').innerHTML='';error('status-error',e.message);}}finally{$('status-button').disabled=false;$('status-button').textContent='Check status';}
});
$('status-form').addEventListener('input',()=>{statusVersion++;$('swap-status').innerHTML='';error('status-error');});
for(const link of document.querySelectorAll('.nav-link'))link.addEventListener('click',()=>{document.querySelector('.nav-link.active')?.classList.remove('active');link.classList.add('active');});
async function refreshPositions(){if(busy||document.hidden)return;try{const data=await api('/api/positions');if(!busy&&(!snapshot||data.revision>snapshot.revision))accept(data);error('global-error');}catch(e){error('global-error',`Positions unavailable: ${e.message}`);}}
refreshPositions();loadTokens();setInterval(refreshPositions,5000);
