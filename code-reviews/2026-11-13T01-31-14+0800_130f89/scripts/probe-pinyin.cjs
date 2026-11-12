// Probe: reproduce the pinyin key derivation used by frontend/src/state.ts.
const {pinyin} = require('pinyin-pro');
const samples = ['微信','剪映专业版','Visual Studio Code','7-Zip','Visual-Studio Code','微信 WeChat','网易云音乐','OBS Studio','C++ Builder','123 播放器','开发工具','a-b','MPC-HC'];
for (const s of samples) {
  const arr = pinyin(s, {toneType:'none', type:'array', nonZh:'consecutive'});
  const full = arr.join('').toLocaleLowerCase().replace(/[^a-z0-9]+/g,'');
  const initials = arr.map(syl => syl.split(/[^a-z0-9]+/i).filter(Boolean).map(w=>w[0]).join('')).join('').toLocaleLowerCase();
  console.log(JSON.stringify(s), '=>', JSON.stringify(arr), '| full=', full, '| initials=', initials);
}
