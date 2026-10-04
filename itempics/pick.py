#!/usr/bin/env python3
"""Choose one picture per item: python3 pick.py, open http://localhost:8199. Click a picture to pick it
(picks survive reloads in picks.json), or paste the path of a local file instead; the game's current picture is shown first and kept unless another is picked; "Confirm" copies every picked out/<model>/<id>.png to web/items/<id>.png."""
import http.server, json, os, subprocess, time, urllib.parse
from gen import TEMPLATE, to_png

HERE = os.path.dirname(os.path.abspath(__file__))
OUT, PICKS, DEST = os.path.join(HERE, "out"), os.path.join(HERE, "picks.json"), os.path.join(HERE, "..", "web", "items")
items = [l.split("\t") for l in open(os.path.join(HERE, "prompts.txt")) if l.strip() and not l.startswith("#")]
load = lambda: json.load(open(PICKS)) if os.path.exists(PICKS) else {}

PAGE = """<!doctype html><meta charset=utf-8><title>Item pictures</title><style>
body{background:#111;color:#ddd;font:14px sans-serif;margin:0 16px}h3{margin:18px 0 4px}h3 button{font-size:12px;padding:1px 8px}small{color:#888}
.row{display:flex;gap:8px;flex-wrap:wrap}figure{margin:0;cursor:pointer;border:4px solid transparent;text-align:center}
figure.on{border-color:#e9a63a}figcaption input{width:240px}.img{width:256px;height:256px;background:#333}img{width:256px;height:256px;object-fit:contain;background:#fff;display:block}
#bar{position:sticky;top:0;background:#111;padding:8px 0;z-index:1}button{font-size:16px;padding:4px 16px}</style>
<div id=bar><button onclick=confirmAll()>Confirm: copy picks into the game</button> <span id=n></span></div><div id=list></div><script>
let d, picks
async function load(){ d = await (await fetch("data")).json(); picks = d.picks; draw() }
function draw(){
  list.innerHTML = d.items.map(([id, line]) => `<h3>${id} <small>${line}</small> <button onclick="copy(this,'${id}')">copy prompt</button></h3><div class=row>` +
    (d.current.includes(id) ? `<figure class="${picks[id]?"":"on"}" onclick="keep('${id}')"><img loading=lazy src="local?p=${encodeURIComponent(d.dest + "/" + id + ".png")}&t=${d.t}"><figcaption>in the game</figcaption></figure>` : "") +
    d.models.map(m => d.have[m].includes(id) ? `<figure class="${picks[id]==m?"on":""}" onclick="pick('${id}','${m}')"><img loading=lazy src="out/${m}/${id}.png"><figcaption>${m}</figcaption></figure>` : "").join("") +
    `<figure class="${local(picks[id])?"on":""}">${local(picks[id])?`<img src="local?p=${encodeURIComponent(picks[id])}">`:"<div class=img></div>"}<figcaption><input placeholder="or paste a file path" value="${local(picks[id])?picks[id]:""}" onchange="setPath('${id}',this.value.trim())"></figcaption></figure></div>`).join("")
  n.textContent = `${Object.keys(picks).length} / ${d.items.length} picked`
}
function copy(b, id){ navigator.clipboard.writeText(d.template + d.items.find(i => i[0] == id)[1]); b.textContent = "copied" }
const local = p => p && p.includes("/") // a pick is a model name, or a pasted file path
async function save(){ draw(); await fetch("picks", {method: "POST", body: JSON.stringify(picks)}) }
function pick(id, m){ picks[id] == m ? delete picks[id] : picks[id] = m; save() }
function keep(id){ delete picks[id]; save() }
function setPath(id, p){ p ? picks[id] = p : delete picks[id]; save() }
async function confirmAll(){ alert(await (await fetch("confirm", {method: "POST"})).text()); load() }
load()</script>"""

class H(http.server.SimpleHTTPRequestHandler):
    def __init__(s, *a, **k): super().__init__(*a, directory=HERE, **k)
    def reply(s, body, type="text/plain"):
        s.send_response(200); s.send_header("Content-Type", type); s.end_headers(); s.wfile.write(body.encode())
    def do_GET(s):
        if s.path == "/": return s.reply(PAGE, "text/html")
        if s.path.startswith("/local?p="):  # a pasted path, previewed
            p = os.path.expanduser(urllib.parse.parse_qs(s.path.split("?", 1)[1])["p"][0])
            if not os.path.isfile(p): return s.send_error(404)
            s.send_response(200); s.end_headers(); s.wfile.write(open(p, "rb").read()); return
        if s.path == "/data":
            models = sorted(m for m in os.listdir(OUT) if os.path.isdir(os.path.join(OUT, m))) if os.path.isdir(OUT) else []
            have = {m: [f[:-4] for f in os.listdir(os.path.join(OUT, m)) if f.endswith(".png")] for m in models}
            return s.reply(json.dumps({"template": TEMPLATE, "items": [[i, l.strip()] for i, l in items], "models": models, "have": have, "picks": load(),
                                       "dest": os.path.abspath(DEST), "current": [f[:-4] for f in os.listdir(DEST) if f.endswith(".png")] if os.path.isdir(DEST) else [], "t": time.time()}))
        super().do_GET()
    def do_POST(s):
        if s.path == "/picks":
            open(PICKS, "w").write(s.rfile.read(int(s.headers["Content-Length"])).decode()); return s.reply("ok")
        if s.path == "/confirm":
            os.makedirs(DEST, exist_ok=True)
            picks = load()
            missing = []
            for id, m in picks.items():
                src = os.path.expanduser(m) if "/" in m else os.path.join(OUT, m, id + ".png")
                if not os.path.isfile(src):
                    missing.append(f"{id}: {m}"); continue
                open(os.path.join(DEST, id + ".png"), "wb").write(to_png(open(src, "rb").read()))
                # 512px is plenty for the ≤320pt the game shows (gfx/visible.go renderItem); full size made web/items 120MB
                subprocess.run(["sips", "-Z", "512", os.path.join(DEST, id + ".png")], capture_output=True, check=True)
            return s.reply(f"copied {len(picks) - len(missing)} pictures to web/items/" + "".join("\nmissing " + x for x in missing))

http.server.ThreadingHTTPServer(("localhost", 8199), H).serve_forever()
