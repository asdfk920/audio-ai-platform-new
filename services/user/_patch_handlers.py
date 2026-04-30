import re
import pathlib

h = pathlib.Path(__file__).parent / "internal" / "handler"
base = "github.com/jacklau/audio-ai-platform/services/user/internal/logic"

auth_news = {
    "NewLoginLogic", "NewRegisterLogic", "NewResetPasswordLogic", "NewRefreshTokenLogic",
    "NewRebindContactLogic", "NewBindContactLogic", "NewSendVerifyCodeLogic", "NewLogoutLogic",
    "NewOauthUnbindLogic", "NewOauthWechatCallbackLogic", "NewOauthGoogleCallbackLogic",
    "NewOauthWechatStartLogic", "NewOauthGoogleStartLogic", "NewChangePasswordLogic",
}
member_news = {
    "NewGetUserMemberBenefitsLogic", "NewCreateMemberOrderLogic", "NewMemberPayCallbackLogic",
    "NewPayMemberOrderLogic", "NewAdminMemberListLogic",
}


def rep(m: re.Match) -> str:
    name = m.group(1)
    if name in auth_news:
        return "auth." + name
    if name in member_news:
        return "member." + name
    return "userlogic." + name


for p in sorted(h.glob("*.go")):
    t = p.read_text(encoding="utf-8")
    if "logic.New" not in t:
        continue
    names = re.findall(r"logic\.(New\w+)", t)
    if not names:
        continue
    need_auth = any(n in auth_news for n in names)
    need_member = any(n in member_news for n in names)
    need_user = any(n not in auth_news and n not in member_news for n in names)
    t2 = re.sub(r"logic\.(New\w+)", rep, t)
    lines = []
    for line in t2.splitlines(keepends=True):
        if f'"{base}"' in line:
            continue
        lines.append(line)
    body = "".join(lines)
    imps = []
    if need_auth:
        imps.append(f'\t"{base}/auth"')
    if need_member:
        imps.append(f'\t"{base}/member"')
    if need_user:
        imps.append(f'\tuserlogic "{base}/user"')
    if imps:
        needle = "import (\n"
        if needle in body:
            a, b = body.split(needle, 1)
            body = a + needle + "\n".join(imps) + "\n" + b
    p.write_text(body, encoding="utf-8")
    print("patched", p.name)

print("done")
