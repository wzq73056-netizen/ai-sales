const count = async () => (await fetch('http://127.0.0.1:9333/json/list').then(r => r.json())).filter(p => p.type === 'page').length
console.log('跑之前:', await count())
