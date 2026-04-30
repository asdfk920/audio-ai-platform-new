"""Normalize handlers to import internal/logic only (package logic)."""
import pathlib
import re

h = pathlib.Path(__file__).parent / "internal" / "handler"
base = "github.com/jacklau/audio-ai-platform/services/user/internal/logic"

for p in sorted(h.glob("*.go")):
    t = p.read_text(encoding="utf-8")
    orig = t
    t = re.sub(r'"github\.com/jacklau/audio-ai-platform/services/user/internal/logic/auth"\s*\n', "", t)
    t = re.sub(r'"github\.com/jacklau/audio-ai-platform/services/user/internal/logic/member"\s*\n', "", t)
    t = re.sub(r'\tuserlogic "github\.com/jacklau/audio-ai-platform/services/user/internal/logic/user"\s*\n', "", t)
    t = t.replace("auth.New", "logic.New")
    t = t.replace("member.New", "logic.New")
    t = t.replace("userlogic.New", "logic.New")
    if f'"{base}"' not in t and ("logic.New" in t or "NewLoginLogic" in t):
        needle = "import (\n"
        if needle in t:
            a, b = t.split(needle, 1)
            t = a + needle + f'\t"{base}"\n' + b
    if t != orig:
        p.write_text(t, encoding="utf-8")
        print("ok", p.name)

print("done")
