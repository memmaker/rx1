#!/usr/bin/env python3
"""Generate item pictures: python3 gen.py [model ...]  (default: flux2 zimage qwen-fast).
One model at a time over all of prompts.txt, so each model loads once. Writes out/<model>/<id>.png and
skips files that exist, so a killed run resumes where it stopped.
gemini-* models go to Google instead (needs GEMINI_API_KEY), e.g.: python3 gen.py gemini-2.5-flash-image
gpt-image-* models go to OpenAI (needs OPENAI_API_KEY), e.g.: python3 gen.py gpt-image-1"""
import base64, json, os, subprocess, sys, tempfile, time, urllib.parse, urllib.request

API = "http://10.0.0.142:8188/generate"
TEMPLATE = "High-res, gray-scale, pencil drawing in the style of 80s RPG manuals, NO TEXT, NO BACKGROUND, NO CHARACTERS, only the item:\n"
GEMINI = "https://generativelanguage.googleapis.com/v1beta/models/{}:generateContent"


def fetch(model, prompt):
    if model.startswith("gpt-image"):
        req = urllib.request.Request("https://api.openai.com/v1/images/generations",
                                     json.dumps({"model": model, "prompt": prompt, "size": "1024x1024", "output_format": "png"}).encode(),
                                     {"Content-Type": "application/json", "Authorization": "Bearer " + os.environ["OPENAI_API_KEY"]})
        return base64.b64decode(json.load(urllib.request.urlopen(req, timeout=300))["data"][0]["b64_json"])
    if not model.startswith("gemini"):
        return urllib.request.urlopen(API + "?" + urllib.parse.urlencode({"prompt": prompt, "model": model}), timeout=900).read()
    body = {"contents": [{"parts": [{"text": prompt}]}], "generationConfig": {"responseModalities": ["IMAGE"], "imageConfig": {"aspectRatio": "1:1"}}}
    req = urllib.request.Request(GEMINI.format(model), json.dumps(body).encode(),
                                 {"Content-Type": "application/json", "x-goog-api-key": os.environ["GEMINI_API_KEY"]})
    parts = json.load(urllib.request.urlopen(req, timeout=300))["candidates"][0]["content"]["parts"]
    img = next(p["inlineData"] for p in parts if "inlineData" in p)
    data = base64.b64decode(img["data"])
    return to_png(data)


def to_png(data):
    """The game decodes PNG only: anything else goes through macOS sips."""
    if data.startswith(b"\x89PNG"):
        return data
    with tempfile.TemporaryDirectory() as d:
        open(d + "/in", "wb").write(data)
        subprocess.run(["sips", "-s", "format", "png", d + "/in", "--out", d + "/out.png"], capture_output=True, check=True)
        return open(d + "/out.png", "rb").read()


HERE = os.path.dirname(os.path.abspath(__file__))

if __name__ == "__main__":  # pick.py imports TEMPLATE
    items = [l.rstrip("\n").split("\t") for l in open(os.path.join(HERE, "prompts.txt")) if l.strip() and not l.startswith("#")]
    for model in sys.argv[1:] or ["flux2", "zimage", "qwen-fast"]:
        os.makedirs(os.path.join(HERE, "out", model), exist_ok=True)
        for i, (id, line) in enumerate(items, 1):
            path = os.path.join(HERE, "out", model, id + ".png")
            if os.path.exists(path):
                continue
            t = time.time()
            for attempt in range(3):
                try:
                    png = fetch(model, TEMPLATE + line)
                    break
                except Exception as e:
                    print(f"  {id}: {e}, retrying", flush=True)
                    time.sleep(10)
            else:
                continue
            open(path + ".tmp", "wb").write(png)
            os.replace(path + ".tmp", path)
            print(f"{model} {i}/{len(items)} {id} {time.time() - t:.0f}s", flush=True)
