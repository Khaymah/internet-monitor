# Privacy Policy · سياسة الخصوصية

Internet Monitor · مراقب الإنترنت

[English](#privacy-policy) · [العربية](#سياسة-الخصوصية)

## Privacy Policy

Last updated: 5 November 2026

Internet Monitor runs on your computer and keeps your data there. This page lists everything it stores and every place it connects to.

### The short version

There's no account, no analytics, no ads and no tracking. The people who make Internet Monitor can't see your history, your settings or your alerts.

### What's stored on your computer

Everything is kept in the app's folder (next to the app, or in %LOCALAPPDATA%\InternetMonitor for the Microsoft Store version), except a few small preferences kept by your browser.

- config.json: your settings. This includes your webhook URL, Telegram token and chat ID, and ntfy topic and token, saved as plain text. Don't share this file. If you add the router's password in Settings → App, it's saved here too, in plain text, and only ever sent to your router.
- The logs folder: every check, outage, speed test and traceroute result, the daily seals that prove the logs weren't edited, plus app.log, a diagnostic log of what the app did. Files older than 90 days are deleted automatically.
- notification.wav: your custom alert sound, if you uploaded one.
- .session: the dashboard key for the current launch. It's replaced every time the app starts.
- Your browser remembers your theme and language, so the dashboard opens the way you left it, plus small flags such as the last What's new you saw and whether you clicked “I want this” on the outside-alerts card. None of this is sent anywhere.
- case.json: what you enter on the Complaints page, such as your provider, plan, price, line and account numbers, and your complaint tickets. It's only used to fill your complaint text and evidence pack, and never leaves your computer unless you share them.

### What leaves your computer

The app only connects to the places below. Each of them sees your IP address, as with any internet connection, and its own privacy policy applies.

- Connection checks go to the targets in Settings → Check targets. By default: Google and Cloudflare DNS servers (8.8.8.8 and 1.1.1.1), connectivitycheck.gstatic.com, and name lookups for www.google.com and www.cloudflare.com. It also checks your own router. If your router allows it (UPnP), the app also reads its line state, sync speed, uptime and traffic counters once a minute. That stays on your home network and is saved in the logs folder as router_DATE.jsonl. You can turn it off in Settings → App. With the router's password set, the app also signs in to the router's own page every few minutes to read the noise margin, attenuation and traffic counters, and signs out again. If the router offers TR-064 (a standard that some routers support), the app can also read the same figures from it, with or without the password. That also stays on your home network.
- Speed tests go to speed.cloudflare.com by default, and only when you start one or schedule them. The test data contains nothing that identifies your device.
- Update checks go to GitHub (api.github.com) every 6 hours (you can turn this off in Settings), or when you press “Check for updates”. Installing an update downloads it from GitHub.
- Alerts and the weekly summary go to Discord, Slack, a webhook of your choice, Telegram or ntfy only if you set them up. They contain the status, times, durations and causes of outages, or a speed test result. The app doesn't add your computer's name or IP address.
- Log timestamps go to freetsa.org only if you turn on “Timestamp the sealed logs” in Settings. It receives a 32-byte hash of the day's seal, never your logs.
- When an outage starts, the app runs a traceroute toward your first check target to see where the connection stops. The routers along the way see these packets, as with any traceroute.
- The tour videos stream from GitHub (github.com and its download servers) only when you press play on the Video tours page. Nothing is loaded from GitHub before that, and the subtitles come from the app itself.

### The dashboard

The dashboard is served by the app on this computer (127.0.0.1). Other devices on your network can't open it, unless you run the headless version with a network address on purpose. It uses one cookie, which holds the key for the current launch. It loads no outside scripts, fonts or images.

### You're in control

You can delete the logs folder at any time to erase your history. Deleting config.json resets your settings, and deleting case.json erases your complaint details. To remove the app completely, turn off “Start with Windows” in Settings → App, quit the app, and delete its folder. The Microsoft Store version is removed from Windows Settings → Apps, and then you can delete %LOCALAPPDATA%\InternetMonitor.

### Changes to this page

If a new version changes what the app stores or where it connects, this page is updated and the change is listed in What's new. The date at the top shows the last update.

### Questions

Reach us through any channel on the Contact page in the app, or open an issue at github.com/Khaymah/internet-monitor. Don't send your config.json, case.json or logs unless we ask, since they may contain your webhook, token, account details or history.

<div dir="rtl">

## سياسة الخصوصية

آخر تحديث: 5 نوفمبر 2026

مراقب الإنترنت بيشتغل على جهازك وبيحتفظ ببياناتك عليه. الصفحة دي بتذكر كل اللي بيحفظه وكل مكان بيتصل بيه.

### باختصار

مفيش حساب، ولا تحليلات، ولا إعلانات، ولا تتبّع. اللي عملوا مراقب الإنترنت مايقدروش يشوفوا سجلك ولا إعداداتك ولا تنبيهاتك.

### اللي بيتحفظ على جهازك

كل حاجة محفوظة في فولدر التطبيق (جنب التطبيق، أو في ‎%LOCALAPPDATA%\InternetMonitor لنسخة Microsoft Store)، ما عدا شوية تفضيلات صغيرة المتصفح بيحفظها.

- config.json: إعداداتك، ومنها رابط الـ webhook وتوكن Telegram ورقم المحادثة، وموضوع ntfy والتوكن بتاعه، محفوظة كنص عادي. ماتشاركش الملف ده. ولو حطيت باسورد الراوتر من الإعدادات ← التطبيق، بيتحفظ هنا كمان كنص عادي، ومابيتبعتش غير للراوتر بتاعك.
- فولدر logs: كل فحص وانقطاع ونتيجة قياس سرعة وtraceroute، والأختام اليومية اللي بتثبت إن السجلات ماتعدّلتش، بالإضافة لـ app.log وهو سجل تشخيصي بيوضّح التطبيق عمل إيه. الملفات الأقدم من 90 يوم بتتمسح تلقائياً.
- notification.wav: صوت التنبيه المخصص، لو كنت رفعت صوت.
- .session: مفتاح لوحة التحكم للتشغيل الحالي، وبيتبدّل مع كل مرة التطبيق يشتغل فيها.
- المتصفح بيفتكر الثيم واللغة، عشان لوحة التحكم تفتح زي ما سيبتها، وكمان علامات صغيرة زي آخر «إيه الجديد» شفته، ولو دوست «عايز ده» في كارت التنبيه من برّه. ومفيش حاجة من دي بتتبعت لأي مكان.
- case.json: اللي بتكتبه في صفحة الشكاوى، زي مزوّد الخدمة والباقة والسعر ورقم الخط ورقم الحساب وأرقام الشكاوى. بيتستخدم بس عشان يملا نص الشكوى وحزمة الأدلة، ومابيخرجش من جهازك غير لو إنت شاركتهم.

### اللي بيخرج من جهازك

التطبيق بيتصل بس بالأماكن دي. كل واحد فيهم بيشوف عنوان IP بتاعك زي أي اتصال بالإنترنت، وبتنطبق عليه سياسة الخصوصية بتاعته.

- فحوصات الاتصال بتروح للأهداف الموجودة في الإعدادات ← أهداف الفحص. افتراضياً: خوادم DNS من Google وCloudflare ‏(8.8.8.8 و1.1.1.1)، وconnectivitycheck.gstatic.com، والبحث عن www.google.com وwww.cloudflare.com. وبيفحص الراوتر بتاعك كمان. ولو الراوتر بيسمح (UPnP)، التطبيق كمان بيقرا حالة الخط وسرعة التزامن ومدة التشغيل وعدّادات الاستهلاك كل دقيقة. ده بيفضل جوه شبكة البيت، وبيتحفظ في فولدر logs باسم router_DATE.jsonl. وتقدر تقفله من الإعدادات ← التطبيق. ولو حطيت باسورد الراوتر، التطبيق كمان بيدخل صفحة الراوتر نفسها كل كام دقيقة يقرا هامش الإشارة والتوهين وعدّادات الاستهلاك، وبعدين يخرج. ولو الراوتر بيدعم TR-064 (معيار بعض الراوترات بتدعمه)، التطبيق ممكن يقرا نفس الأرقام منه كمان، بباسورد أو من غير. وده كمان بيفضل جوه شبكة البيت.
- قياس السرعة بيروح لـ speed.cloudflare.com افتراضياً، وبس لما تبدأه أو تجدوله. بيانات القياس مفيهاش أي حاجة تعرّف جهازك.
- البحث عن التحديثات بيروح لـ GitHub ‏(api.github.com) كل 6 ساعات (وتقدر تقفلها من الإعدادات)، أو لما تدوس «دوّر على تحديثات». وتثبيت التحديث بيتنزّل من GitHub.
- التنبيهات والملخص الأسبوعي بتروح لـ Discord أو Slack أو webhook من اختيارك أو Telegram أو ntfy بس لو فعّلتها. فيها الحالة، والأوقات، والمدد، وأسباب الانقطاع، أو نتيجة قياس السرعة. والتطبيق مابيضيفش اسم جهازك ولا عنوان IP.
- ختم السجلات بالوقت بيروح لـ freetsa.org بس لو فعّلت «ختم السجلات بتوقيت مستقل» من الإعدادات. بيوصله hash طوله 32 بايت لختم اليوم، مش السجلات نفسها.
- لما الانقطاع يبدأ، التطبيق بيعمل traceroute ناحية أول هدف فحص عشان يعرف الاتصال بيقف فين. الراوترات اللي في الطريق بتشوف الباكتات دي زي أي traceroute.
- فيديوهات الجولات بتشتغل من GitHub ‏(github.com وسيرفرات التنزيل بتاعته) بس لما تدوس تشغيل في صفحة «جولات بالفيديو». مفيش حاجة بتتحمّل من GitHub قبل كده، والترجمة جاية من التطبيق نفسه.

### لوحة التحكم

لوحة التحكم التطبيق بيقدّمها على الجهاز ده (127.0.0.1)، والأجهزة التانية على شبكتك مش بتقدر تفتحها، إلا لو شغّلت نسخة headless على عنوان شبكة بقصد. بتستخدم ملف تعريف ارتباط (cookie) واحد فيه مفتاح التشغيل الحالي، ومابتحمّلش أي سكربتات أو خطوط أو صور من بره.

### التحكم في إيدك

تقدر تمسح فولدر logs في أي وقت عشان تمسح سجلك، ومسح config.json بيرجّع الإعدادات للوضع الافتراضي، ومسح case.json بيمسح بيانات الشكاوى. ولإزالة التطبيق بالكامل: اقفل «التشغيل مع Windows» من الإعدادات ← التطبيق، وبعدين اقفل التطبيق وامسح الفولدر بتاعه. ونسخة Microsoft Store بتتشال من إعدادات Windows ← التطبيقات، وبعدها تقدر تمسح ‎%LOCALAPPDATA%\InternetMonitor.

### تغييرات الصفحة دي

لو إصدار جديد غيّر اللي التطبيق بيحفظه أو الأماكن اللي بيتصل بيها، الصفحة دي بتتحدّث والتغيير بيتذكر في «إيه الجديد». التاريخ فوق بيوضّح آخر تحديث.

### أسئلة

تواصل معانا من أي وسيلة في صفحة «تواصل معانا» في التطبيق، أو افتح issue على github.com/Khaymah/internet-monitor. وماتبعتش config.json ولا case.json ولا السجلات غير لو طلبناها، لأنها ممكن تحتوي على الـ webhook أو التوكن أو بيانات حسابك أو سجلك.

</div>
