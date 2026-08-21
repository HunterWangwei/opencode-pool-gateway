# OpenCode Pool Gateway

褰撳墠鐗堟湰锛歚0.7.3`

## API 杞彂缃戝叧

绗笁鏂瑰鎴风浣跨敤鈥滀护鐗岀鐞嗏€濋〉闈㈢敓鎴愮殑鏈珯浠ょ墝锛岃姹傛湰鏈烘垨鏈嶅姟鍣ㄤ笂鐨?OpenCode Pool Gateway 鍦板潃銆傛湰绔欎护鐗屽彲鏀惧湪 `Authorization: Bearer` 鎴?`x-api-key` 涓€侽penCode Pool Gateway 瀹屾暣淇濈暀璇锋眰璺緞銆佹煡璇㈠弬鏁般€佽姹備綋鍜屼笟鍔¤姹傚ご锛屼粎灏嗕笂娓搁壌鏉冩浛鎹负閫変腑鍑瘉鐨?OpenCode API Key銆?
鏀寔璺敱锛?
- `/zen/go/v1/chat/completions`
- `/zen/go/v1/responses`
- `/zen/go/v1/messages`
- `/zen/go/v1/models`
- `/zen/v1/responses`
- `/zen/v1/messages`
- `/zen/v1/chat/completions`
- `/zen/v1/models`
- `/zen/v1/models/<model-id>`锛圙emini 鍘熺敓鍗忚锛?
Zen 妯″瀷鍗忚瀵瑰簲鍏崇郴锛?
| 鍗忚 | 妯″瀷绫诲瀷 | 璺敱 |
| --- | --- | --- |
| OpenAI Responses | GPT銆丟rok | `/zen/v1/responses` |
| Anthropic Messages | Claude銆丵wen | `/zen/v1/messages` |
| Google Generative Language | Gemini | `/zen/v1/models/<model-id>` |
| OpenAI-compatible Chat Completions | DeepSeek銆丮iniMax銆丟LM銆並imi 绛?| `/zen/v1/chat/completions` |

Go 妯″瀷鍗忚瀵瑰簲鍏崇郴锛?
| 鍗忚 | 妯″瀷绫诲瀷 | 璺敱 |
| --- | --- | --- |
| OpenAI Responses | GPT 5.6 Luna | `/zen/go/v1/responses` |
| Anthropic Messages | MiniMax M3/M2.7/M2.5銆丵wen 3.8/3.7/3.6 | `/zen/go/v1/messages` |
| OpenAI-compatible Chat Completions | Grok銆丟LM銆並imi銆丏eepSeek銆丮iMo銆丠y3 绛?| `/zen/go/v1/chat/completions` |
| Models | Go 瀹屾暣妯″瀷鐩綍 | `/zen/go/v1/models` |

璁剧疆椤靛彲鍒囨崲杞璐熻浇鍧囪　涓庝紭鍏堢骇鏁呴殰杞Щ銆傛渶澶у皾璇曟鏁颁负 `0` 鏃讹紝姣忔璇锋眰鏈€澶氬皾璇曟墍鏈夊彲鐢ㄥ嚟璇佷竴娆°€備唬鐞嗘敮鎸?`http://`銆乣https://`銆乣socks5://` 鍜?`socks5h://`锛屽苟鏀寔鐢ㄦ埛鍚嶅瘑鐮佽璇併€?
SOCKS5 绀轰緥锛歚socks5://username:password@127.0.0.1:1080`銆傜敤鎴峰悕鎴栧瘑鐮佸寘鍚壒娈婂瓧绗︽椂闇€瑕佽繘琛?URL 缂栫爜銆?
## 浠ｇ悊姹?
宸︿晶鈥滀唬鐞嗘睜鈥濋〉闈㈡敮鎸佷互琛ㄦ牸绠＄悊澶氭潯浠ｇ悊锛屾樉绀轰唬鐞嗗湴鍧€銆佺粦瀹氬嚟璇併€佽姹傛垚鍔熸鏁板拰璇锋眰澶辫触娆℃暟銆?
- 鏀寔鎵归噺绮樿创浠ｇ悊鍦板潃锛屾瘡琛屼竴鏉★紝鑷姩鍘婚噸鍚庡姞鍏ヤ唬鐞嗘睜銆?- 鏀寔鍚敤鎴栧叧闂唬鐞嗘睜锛涘叧闂悗涓嶄細浣跨敤姹犲唴浠ｇ悊銆?
- 姣忎釜鍑瘉鏍规嵁鑷韩 ID 绋冲畾缁戝畾浠ｇ悊姹犱腑鐨勪竴鏉′唬鐞嗐€?- 鐩稿悓鍑瘉涓嶄細鍦ㄦ瘡娆¤姹傛椂杞崲浠ｇ悊銆?- 璋冩暣浠ｇ悊姹犻『搴忔垨澧炲垹浠ｇ悊鍚庝細閲嶆柊璁＄畻绋冲畾鏄犲皠銆?- 璐﹀彿姹犲崱鐗囦細鏄剧ず褰撳墠鍑瘉瀹為檯缁戝畾鐨勪唬鐞嗐€?- 璇锋眰缁熻鏉ヨ嚜缃戝叧璇锋眰鏃ュ織锛屾垚鍔熺姸鎬佷负鏃犳湰鍦伴敊璇笖 HTTP 鐘舵€佺爜灏忎簬 400銆?
浠ｇ悊浣跨敤浼樺厛绾э細

```text
鍑瘉鐙珛浠ｇ悊 > 浠ｇ悊姹犲浐瀹氫唬鐞?> 鍏ㄥ眬浠ｇ悊
```

`socks5h://` 浼氬皢鐩爣鍩熷悕浜ょ粰 SOCKS5 浠ｇ悊瑙ｆ瀽锛岄€傚悎闇€瑕侀伩鍏嶆湰鍦?DNS 瑙ｆ瀽鐨勫満鏅€?
### 瀹㈡埛绔帴鍏?
1. 鍦ㄢ€滀护鐗岀鐞嗏€濋〉闈㈠垱寤烘湰绔欒闂护鐗屻€?2. 灏嗙涓夋柟瀹㈡埛绔殑 Base URL 鎸囧悜 OpenCode Pool Gateway锛屼緥濡?`http://127.0.0.1:8787/zen/v1`銆?3. 灏嗘湰绔欎护鐗屽～鍐欎负瀹㈡埛绔?API Key銆傚鎴风鍙互鍙戦€?`Authorization: Bearer <鏈珯浠ょ墝>` 鎴?`x-api-key: <鏈珯浠ょ墝>`銆?4. 缃戝叧楠岃瘉鏈珯浠ょ墝鍚庨€夋嫨璐﹀彿姹犱腑鐨?OpenCode API Key锛屽苟鎸夌洰鏍囧崗璁嚜鍔ㄧ敓鎴愪笂娓搁壌鏉冨ご锛涚涓夋柟浠ょ墝涓嶄細鍙戦€佺粰 OpenCode銆?
Responses 绀轰緥锛?
```bash
curl http://127.0.0.1:8787/zen/v1/responses \
  -H "Authorization: Bearer gq_your_gateway_token" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-5.2","input":"Hello"}'
```

Anthropic Messages 绀轰緥锛?
```bash
curl http://127.0.0.1:8787/zen/v1/messages \
  -H "x-api-key: gq_your_gateway_token" \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-sonnet-4","max_tokens":256,"messages":[{"role":"user","content":"Hello"}]}'
```

Gemini 瀹㈡埛绔簲淇濈暀妯″瀷璺緞鍜屽姩浣滃悗缂€锛屼緥濡?`/zen/v1/models/gemini-3.6-flash:generateContent`銆傛煡璇㈠弬鏁般€佽姹備綋鍙婇櫎閴存潈澶栫殑涓氬姟璇锋眰澶村潎鍘熸牱閫忎紶銆?
閴存潈杞崲瑙勫垯锛?
| 璇锋眰鏂瑰悜 | 璺敱 | 閴存潈澶?|
| --- | --- | --- |
| 绗笁鏂瑰鎴风 鈫?鏈珯 | 鍏ㄩ儴缃戝叧璺敱 | `Authorization: Bearer <鏈珯浠ょ墝>` 鎴?`x-api-key: <鏈珯浠ょ墝>` |
| 鏈珯 鈫?OpenCode | `/zen/v1/messages`銆乣/zen/go/v1/messages` | `x-api-key: <OpenCode API Key>` |
| 鏈珯 鈫?OpenCode | 鍏朵粬鏀寔璺敱 | `Authorization: Bearer <OpenCode API Key>` |

OpenCode Pool Gateway 鏄竴涓娇鐢?Go 鏍囧噯搴撴瀯寤虹殑 OpenCode Go / Zen 璐﹀彿姹犵鐞嗕笌 API 杞彂缃戝叧銆傚畠涓嶄緷璧?OpenCode CLI锛屽彲闆嗕腑绠＄悊澶氫釜 Workspace锛屾煡璇㈤搴︿笌妯″瀷锛屽苟鍚戠涓夋柟瀹㈡埛绔彁渚涘甫閴存潈銆佽皟搴︺€佹晠闅滆浆绉诲拰浠ｇ悊鏀寔鐨勫吋瀹规帴鍙ｃ€?
## 鍔熻兘

- 鎵归噺绠＄悊澶氫釜 OpenCode Workspace銆?- 鍒嗗埆璇嗗埆 Go 璁㈤槄棰濆害涓?Zen 鎸夐噺浠樿垂浣欓銆?- 灞曠ず Go 鐨?5 灏忔椂銆佹瘡鍛ㄣ€佹瘡鏈堥搴︾獥鍙ｅ拰閲嶇疆鏃堕棿銆?- 鍒嗗埆鏌ヨ Go 涓?Zen 鍙敤妯″瀷锛屼笉娣风敤妯″瀷鐩綍銆?- 鏍囪 Zen 宸插純鐢ㄦā鍨嬪苟鏄剧ず瀹樻柟寮冪敤鏃ユ湡銆?- 璐﹀彿鍗＄墖鏀寔鎼滅储銆佺姸鎬佺瓫閫夈€佺紪杈戙€佸埛鏂板拰鍒犻櫎銆?- Windows 鎺у埗鍙版敮鎸佹墦寮€缃戦〉銆佸埛鏂般€佸府鍔╁拰瀹夊叏閫€鍑恒€?- 鏁版嵁涓庡彲鎵ц鏂囦欢鏀惧湪鍚屼竴鐩綍锛屼究浜庤縼绉诲拰澶囦唤銆?- 鏈嶅姟绔櫥褰曢獙璇併€佷細璇濈鐞嗗拰鐧诲綍澶辫触闄愭祦銆?- 浣跨敤 Workspace ID 涓?Cookie 鑷姩璇嗗埆璐﹀彿閭鍜屾湰浜?API Key銆?- 涓€涓处鍙峰瓨鍦ㄥ涓?API Key 鏃跺彲鍦ㄦ坊鍔犻〉闈㈤€夋嫨浣跨敤鐨?Key銆?- 鏀寔杞璐熻浇鍧囪　鍜屾寜鏁板瓧鍗囧簭鐨勫嚟璇佷紭鍏堢骇妯″紡銆?- 涓婃父澶辫触鏃惰嚜鍔ㄥ垏鎹㈠叾浠栧嚟璇侊紝姣忎釜鍑瘉鍦ㄥ崟娆¤姹備腑鏈€澶氬皾璇曚竴娆°€?- 鏀寔鍏ㄥ眬浠ｇ悊涓庡嚟璇佺嫭绔嬩唬鐞嗭紝鐙珛浠ｇ悊浼樺厛銆?- 鏈珯璁块棶浠ょ墝浠呬繚瀛樻憳瑕侊紝绗笁鏂逛护鐗屼笉浼氶€忎紶缁?OpenCode銆?- 璇锋眰鏃ュ織璁板綍妯″瀷銆佸嚟璇併€乀oken銆佺紦瀛樸€佽€楁椂鍜岃劚鏁忛敊璇姤鏂囥€?- 璇锋眰鏃ュ織鏀寔鏅€?JSON銆乬zip 鍜?SSE 娴佸紡鍝嶅簲鐨勭敤閲忚В鏋愶紝鑰楁椂鑷姩鏄剧ず涓烘绉掋€佺鎴栧垎绉掋€?- 鑷姩鍖哄垎 Free銆丟o銆乑en 鍜?Go + Zen 璐﹀彿绫诲瀷锛屽苟鎸夋ā鍨嬫潈鐩婄瓫閫夎浆鍙戝嚟璇併€?
## 妯″瀷鏉冮檺

- Free 璐﹀彿涓嶈兘鐢ㄤ簬 Go 璺敱銆?- Free 璐﹀彿鍙互浣跨敤瀹樻柟瀹氫环椤垫爣璁颁负鍏嶈垂鐨?Zen 妯″瀷銆?- Zen 闈炲厤璐规ā鍨嬪彧浼氳皟搴﹀凡寮€鍚?Zen 璁¤垂鐨勫嚟璇併€?- Go 璺敱鍙細璋冨害宸插紑閫?Go 鐨勫嚟璇併€?- 鈥滄ā鍨嬧€濈獥鍙ｄ細鏄剧ず姣忎釜妯″瀷瀵瑰綋鍓嶅嚟璇佹槸鍚﹀彲鐢紝浠ュ強涓嶅彲鐢ㄥ師鍥犮€?
Zen 鍏嶈垂妯″瀷娓呭崟渚濇嵁 OpenCode 瀹樻柟瀹氫环椤电淮鎶ゃ€傚畼鏂瑰彲鑳借皟鏁村厤璐规湡闄愭垨妯″瀷鍒楄〃锛屽崌绾х増鏈椂搴斿悓姝ユ鏌ャ€?
褰撳墠棰濆璇嗗埆涓?Free 鐨勬ā鍨嬪寘鎷?`hy3-free` 鍜?`nemotron-3.5-lightning-free`銆?
## 蹇€熷紑濮?
### Windows

涓嬭浇 `opencode-pool-gateway-0.7.3-windows-amd64.exe`锛屽弻鍑昏繍琛岋紝鐒跺悗璁块棶锛?
```text
http://localhost:8787
```

鎺у埗鍙板懡浠わ細

```text
O  鎵撳紑缃戦〉
R  鍒锋柊鍏ㄩ儴璐﹀彿
Q  瀹夊叏閫€鍑?H  鏄剧ず甯姪
```

棣栨鍚姩鏃剁▼搴忎細鍒涘缓 `data/auth.json`锛屾帶鍒跺彴鏄剧ず涓€娆￠殢鏈哄垵濮嬪瘑鐮併€傜櫥褰曞悗璇峰湪鈥滆缃?鈫?淇敼鐧诲綍鍑瘉鈥濅腑淇敼鐢ㄦ埛鍚嶅拰瀵嗙爜銆?
### Linux

```bash
chmod +x opencode-pool-gateway-0.7.3-linux-amd64
./opencode-pool-gateway-0.7.3-linux-amd64
```

ARM64 Ubuntu 浣跨敤 `opencode-pool-gateway-0.7.3-linux-arm64`銆傛湇鍔″櫒閮ㄧ讲鍙?systemd 閰嶇疆瑙?[docs/ubuntu.md](docs/ubuntu.md)銆?
## 鐧诲綍瀹夊叏

鐧诲綍閰嶇疆淇濆瓨鍦ㄥ彲鎵ц鏂囦欢鏃佺殑 `data/auth.json`锛屼紭鍏堢骇楂樹簬鍏朵粬鏉ユ簮銆傚瘑鐮佷娇鐢ㄩ殢鏈虹洂鍜?PBKDF2-HMAC-SHA256 鍝堝笇淇濆瓨锛屼笉鍐欏叆鏄庢枃銆?
鏂囦欢涓嶅瓨鍦ㄦ椂锛岀▼搴忎細鍒涘缓鐢ㄦ埛鍚?`admin` 鍜岄殢鏈哄瘑鐮侊紝骞跺湪鎺у埗鍙版垨 systemd 鏃ュ織涓樉绀轰竴娆″垵濮嬪瘑鐮併€傜櫥褰曞悗鍙湪缃戦〉璁剧疆涓慨鏀癸紱淇濆瓨浼氱珛鍗崇儹鏇存柊閰嶇疆锛屽苟娉ㄩ攢鍏ㄩ儴鏃т細璇濄€?
HTTPS 閮ㄧ讲鍙缃?`OPG_COOKIE_SECURE=1`锛屽己鍒舵祻瑙堝櫒鍙€氳繃 HTTPS 鍙戦€佷細璇?Cookie銆?
璁よ瘉淇濇姢瑕嗙洊缃戦〉銆侀潤鎬佽祫婧愬拰鍏ㄩ儴 `/api/` 鎺ュ彛銆備細璇濇湁鏁堟湡涓?24 灏忔椂锛汣ookie 浣跨敤 `HttpOnly` 鍜?`SameSite=Strict`銆?
## 娣诲姞璐﹀彿

姣忎釜 Workspace 鍙坊鍔犱竴娆★紝绋嬪簭浼氬悓鏃舵娴?Go 涓?Zen锛?
娣诲姞椤甸潰鏀寔涓夌鏂瑰紡锛?
- `Workspace + Cookie`锛氬彲鏌ヨ璐﹀彿閭銆丄PI Key銆丟o 棰濆害銆乑en 浣欓鍜屽畬鏁磋处鍙风被鍨嬨€?- `璐﹀彿瀵嗙爜鐧诲綍`锛氬悗鍙版墽琛屽唴缃?GitHub OAuth 鍗忚鑴氭湰锛岃嚜鍔ㄥ彇寰?Workspace ID銆乤uth Cookie銆佽处鍙烽偖绠卞拰鏈汉 API Key銆傞渶瑕?Python 3 涓?`requests`銆?- `浠?API Key`锛氶€傚悎鍙嬁鍒?API Key 鐨勫満鏅€傜▼搴忎細浣跨敤 `deepseek-v4-flash` 鍒嗗埆璇锋眰 `/zen/go/v1/chat/completions` 涓?`/zen/v1/chat/completions`锛涜嫢鍝嶅簲杩斿洖 `CreditsError`锛屼細浠庝粯娆炬柟寮忛摼鎺ヨ嚜鍔ㄦ彁鍙?Workspace ID锛屽苟灏嗗叾浣滀负榛樿鏄剧ず鍚嶇О銆?
璐﹀彿瀵嗙爜鐧诲綍浼氭寜鑴氭湰瀹氫箟鐨?`璐﹀彿----瀵嗙爜----閭瀵嗙爜` 鍙傛暟鍗忚杩愯銆傜▼搴忓湪鍙墽琛屾枃浠跺悓绾х殑 `temp/opencode-auth-*` 鐩綍璇诲彇鑴氭湰鐢熸垚鐨?Account銆乄orkspace ID 鍜?Auth Cookie銆傛瘡娆¤繍琛岀殑鑴氭湰銆佹爣鍑嗚緭鍑哄拰缁撴灉鏂囦欢浼氫繚鐣欙紝渚夸簬鎺掓煡鎻愬彇澶辫触锛涚櫥褰曞瘑鐮佷笉浼氬啓鍏ヨ处鍙烽厤缃€?
瀹夎鑴氭湰渚濊禆锛?
```bash
python3 -m pip install requests
```

Windows 涔熷彲浠ヤ娇鐢細

```powershell
py -3 -m pip install requests
```

姣忔鍗忚鐧诲綍鐨勮皟璇曟枃浠朵繚瀛樺湪锛?
```text
temp/opencode-auth-闅忔満瀛楃/
鈹溾攢鈹€ opencode_auth_extractor.py
鈹溾攢鈹€ opencode_璐﹀彿.json
鈹溾攢鈹€ stdout.log
鈹斺攢鈹€ stderr.log
```

杩欎簺鐩綍涓嶄細鑷姩鍒犻櫎锛岀‘璁や笉鍐嶉渶瑕佸悗鍙墜鍔ㄦ竻鐞嗐€俙temp/` 宸茶 Git 蹇界暐銆?
API Key-only 鏂板鐣岄潰鍙姹傚～鍐?API Key銆佷紭鍏堢骇鍜岀嫭绔嬩唬鐞嗐€傛帰娴嬭姹傚浐瀹氫娇鐢?`deepseek-v4-flash` 鍜?`max_tokens: 1`锛屽彲鑳戒骇鐢熸瀬灏戦噺 Token 娑堣€椼€傚叕寮€妯″瀷鐩綍涓嶄細鐢ㄤ簬楠岃瘉 API Key銆傝嫢涓婃父鎴愬姛鍝嶅簲娌℃湁杩斿洖 Workspace ID锛屽嚟璇佷粛浼氫繚瀛樹负 API Key-only锛岄搴︿俊鎭繚鎸佷笉鍙煡璇紝涓斿悗缁埛鏂颁笉浼氶噸澶嶆墽琛屾潈鐩婃帰娴嬨€?
浠?API Key 鍙互纭閮ㄥ垎 Go/Zen 璇锋眰鏉冪泭锛屼絾鏃犳硶璇诲彇 Workspace 椤甸潰涓殑 Go 5 灏忔椂銆佹瘡鍛ㄣ€佹瘡鏈堥搴︾獥鍙ｆ垨 Zen 浣欓銆傚崱鐗囦細灏嗚繖绉嶆儏鍐垫爣璁颁负鈥滈渶 Cookie 鏌ヨ鈥濓紝涓嶄細璇垽涓衡€滄湭寮€閫氣€濄€?
鎺㈡祴妯″瀷杩斿洖 `RegionError` 琛ㄧず璇ユā鍨嬮渶瑕佸湪 Workspace 涓槑纭€夋嫨鎵樼鍖哄煙锛屽苟涓嶄唬琛?API Key 鏃犳晥鎴栨暣涓?Go/Zen 鏈嶅姟涓嶅彲鐢ㄣ€傜▼搴忎細浠庨敊璇摼鎺ユ彁鍙?Workspace ID锛屽苟灏嗗搴旀湇鍔¤涓哄凡閫氳繃閴存潈銆傚彧鏈?HTTP 401 鎴栨槑纭殑璁よ瘉閿欒鎵嶄細鎷掔粷 API Key銆?
绋嬪簭鍚姩鏃跺彧璇诲彇璐﹀彿閰嶇疆锛屼笉浼氭寜鍚嶇О銆乄orkspace ID銆丆ookie銆佽处鍙风被鍨嬫垨瀛楁瀹屾暣搴﹁嚜鍔ㄥ垹闄ゆ垨鍚堝苟璐﹀彿銆傝处鍙蜂粎鑳界敱绠＄悊椤甸潰鐨勫垹闄ゆ搷浣滅Щ闄ゃ€?
| 瀛楁 | 鐢ㄩ€?| 鏄惁蹇呭～ |
| --- | --- | --- |
| 鏄剧ず鍚嶇О | 鏈湴璇嗗埆璐﹀彿锛涚暀绌鸿嚜鍔ㄤ娇鐢ㄩ偖绠?| 鍚?|
| Workspace ID | 褰㈠ `wrk_...` | 鏄?|
| OpenCode API Key | 鏌ヨ Go/Zen 妯″瀷鐩綍锛涚暀绌鸿嚜鍔ㄨ幏鍙?| 鍚?|
| `auth` Cookie | 鏌ヨ Workspace Go 棰濆害鍜?Zen 浣欓 | 鏄?|

鐐瑰嚮鈥滆嚜鍔ㄨ瘑鍒处鍙封€濆悗锛岀▼搴忎細璇诲彇褰撳墠璐﹀彿閭鍜岃鐢ㄦ埛鏈汉鎷ユ湁鐨勫畬鏁?API Key銆傚瓨鍦ㄥ涓?Key 鏃堕渶瑕侀€夋嫨涓€涓紱鍏朵粬鎴愬憳鐨勬帺鐮?Key 涓嶄細浣滀负鍊欓€夐」銆?
API Key 涓庣綉绔?Cookie 鐢ㄩ€斾笉鍚屻€備粎鏈?API Key 涓嶈兘鏌ヨ Workspace 椤甸潰涓殑 Go 棰濆害鍜?Zen 浣欓銆傝嚜鍔ㄨ瘑鍒け璐ユ椂鍙鏌?Cookie 鍚庨噸璇曪紝涔熷彲鎵嬪姩濉啓鍚嶇О鍜?API Key銆?
### 鑾峰彇 Workspace ID 鍜?Cookie

1. 浣跨敤 GitHub 鎴?Google 鐧诲綍 [OpenCode Zen](https://opencode.ai/zen)銆?2. 杩涘叆宸ヤ綔鍖哄悗锛屼粠鍦板潃鏍忓鍒?`workspace/wrk_...` 涓殑 Workspace ID銆?3. 浠庢祻瑙堝櫒寮€鍙戣€呭伐鍏风殑 Cookie 鍒楄〃澶嶅埗 `opencode.ai` 鐨勫畬鏁?`auth` Cookie銆?
OpenCode 鐧诲綍閲囩敤 GitHub/Google OAuth锛屾渶缁堜細璇?Cookie 涓?HttpOnly銆侽penCode Pool Gateway 涓嶄繚瀛?GitHub/Google 瀵嗙爜锛屼篃鏃犳硶鍦ㄧ函鏈嶅姟绔垨 Ubuntu 閮ㄧ讲涓洿鎺ヨ鍙栨祻瑙堝櫒 Cookie锛屽洜姝ょ洰鍓嶄粛闇€鎵嬪姩鎻愪緵 Workspace ID 鍜?Cookie銆?
## 鏁版嵁鐩綍

閰嶇疆鍥哄畾淇濆瓨鍦ㄥ彲鎵ц鏂囦欢鏃侊細

```text
data/accounts.json
data/auth.json
data/gateway.json
data/tokens.json
data/requests.jsonl
```

`accounts.json` 鍖呭惈 OpenCode 鏁忔劅鍑瘉锛沗auth.json` 鍖呭惈绠＄悊璐﹀彿鍜屽瘑鐮佸搱甯岋紱`tokens.json` 鍙繚瀛樻湰绔欎护鐗屾憳瑕侊紱`requests.jsonl` 淇濆瓨鑴辨晱璇锋眰鏃ュ織銆傝鍕夸笂浼犮€佸垎浜垨鎻愪氦 `data/`锛涗粨搴撳凡榛樿鎺掗櫎璇ョ洰褰曘€?
寤鸿鏉冮檺锛?
```bash
chmod 700 data
chmod 600 data/accounts.json
```

## 鏋勫缓

瑕佹眰 Go 1.22 鎴栨洿楂樼増鏈€?
Windows PowerShell锛?
```powershell
.\scripts\build.ps1
```

Linux/macOS锛?
```bash
chmod +x scripts/build.sh
./scripts/build.sh
```

浜х墿鐢熸垚鍦?`dist/`锛屽寘鍚?Windows AMD64銆丩inux AMD64銆丩inux ARM64 鍜?`SHA256SUMS.txt`銆?
鐗堟湰鍙风敱鏍圭洰褰?`VERSION` 绠＄悊锛屽苟閫氳繃鏋勫缓鍙傛暟鍐欏叆绋嬪簭锛?
```bash
./opencode-pool-gateway-0.7.3-linux-amd64 --version
```

## 鎺ュ彛

| 鎺ュ彛 | 鏂规硶 | 璇存槑 |
| --- | --- | --- |
| `/api/accounts` | GET/POST | 鏌ヨ鎴栨坊鍔犺处鍙凤紙鍚嶇О鍜?API Key 鍙嚜鍔ㄨ幏鍙栵級 |
| `/api/accounts/discover` | POST | 浣跨敤 Workspace ID 涓?auth Cookie 鑷姩璇诲彇閭鍜屾湰浜?API Key |
| `/api/accounts/login-extract` | POST | 杩愯璐﹀彿瀵嗙爜鐧诲綍鍗忚骞惰嚜鍔ㄨ鍙?Workspace銆丆ookie銆侀偖绠卞拰 API Key |
| `/api/accounts/{id}` | PUT/DELETE | 缂栬緫鎴栧垹闄よ处鍙?|
| `/api/accounts/{id}/refresh` | POST | 鍒锋柊鍗曚釜璐﹀彿 |
| `/api/accounts/{id}/models` | GET | 鍒嗗埆鏌ヨ Go/Zen 妯″瀷 |
| `/api/refresh` | POST | 鍒锋柊鍏ㄩ儴璐﹀彿 |
| `/api/settings` | GET/PUT | 鏌ヨ鎴栫儹鏇存柊璋冨害銆侀噸璇曞拰鍏ㄥ眬浠ｇ悊閰嶇疆 |
| `/api/tokens` | GET/POST | 鏌ヨ鎴栧垱寤烘湰绔欒闂护鐗?|
| `/api/tokens/{id}` | PUT/DELETE | 鍚仠銆侀噸鍛藉悕鎴栧垹闄や护鐗?|
| `/api/logs` | GET/DELETE | 鏌ヨ鎴栨竻绌鸿姹傛棩蹇?|
| `/api/version` | GET | 鏌ヨ鐗堟湰淇℃伅 |
| `/api/shutdown` | POST | 瀹夊叏閫€鍑虹▼搴?|

## 瀹夊叏璇存槑

- 鍗充娇宸叉湁搴旂敤鐧诲綍锛屼篃寤鸿閫氳繃 HTTPS 鍙嶅悜浠ｇ悊璁块棶锛屼笉瑕侀€氳繃鍏綉鏄庢枃 HTTP 杈撳叆瀵嗙爜銆?- Ubuntu 閮ㄧ讲寤鸿閰嶅悎闃茬伀澧欐垨 VPN锛岄伩鍏嶆棤鍏虫潵婧愯闂櫥褰曞叆鍙ｃ€?- Cookie 鍜?API Key 鍧囪涓鸿处鍙峰嚟璇侊紱娉勬紡鍚庡簲绔嬪嵆鎾ら攢鎴栭噸鏂扮櫥褰曘€?- OpenCode Pool Gateway 鏄潪瀹樻柟椤圭洰锛孫penCode 椤甸潰鎴栨帴鍙ｇ粨鏋勫彉鍖栧彲鑳藉鑷撮噰闆嗘殏鏃跺け鏁堛€?- 绗笁鏂瑰鎴风搴斾娇鐢ㄦ湰绔欎护鐗岋紝鍙斁鍦?`Authorization: Bearer <token>` 鎴?`x-api-key: <token>` 涓紝鍒囧嬁鐩存帴鏆撮湶 OpenCode API Key銆?
## 鐗堟湰绠＄悊

椤圭洰閬靛惊 [Semantic Versioning](https://semver.org/)銆傚彂甯冩柊鐗堟湰鏃讹細

1. 鏇存柊 `VERSION`銆?2. 鏇存柊 `CHANGELOG.md`銆?3. 杩愯娴嬭瘯涓庢瀯寤鸿剼鏈€?4. 鍒涘缓 `vX.Y.Z` Git 鏍囩鍜?GitHub Release銆?
## License

[MIT](LICENSE)



