from pathlib import Path

p = Path(__file__).parent / "internal" / "logic" / "auth" / "login_logic.go"
t = p.read_text(encoding="utf-8")
repls = [
    ("一\ufffd?)", "一种\")"),
    ("验证\ufffd?)", "验证码\")"),
    ("其中一\ufffd?)", "其中一种\")"),
    ("手机号\ufffd?)", "手机号）\")"),
    ("第三方登\ufffd?)", "第三方登录\")"),
    ("联系方式类\ufffd?)", "联系方式类型\")"),
]
for a, b in repls:
    t = t.replace(a, b)
p.write_text(t, encoding="utf-8")
print("ok")
