# Microsoft Store listing

Text for Partner Center → Store listings. The package declares English and Arabic, so
both listings are needed. "What's new in this version" stays empty on the first
submission. Each product feature is at most 200 characters.

Screenshots: `docs/store/ar/` (Arabic listing) and `docs/store/en/` (English listing),
1920 × 1080, in upload order: overview, complaints, monthly report, history. They show
2.0.4-rc.6 with sample data.

## Remaining Partner Center fields

| Section | Value |
| ------- | ----- |
| Pricing and availability → Markets | All markets |
| Pricing and availability → Pricing | Free |
| Pricing and availability → Visibility | Public audience, discoverable in the Store |
| Properties → Category | Utilities + tools (subcategory: none) |
| Properties → Privacy policy URL | `https://github.com/Khaymah/internet-monitor/blob/master/PRIVACY.md` |
| Properties → Website | `https://github.com/Khaymah/internet-monitor` |
| Properties → Support contact info | `https://github.com/Khaymah/internet-monitor/issues` |
| Properties → Product declarations | Leave all unchecked |
| Properties → System requirements | Leave every hardware item unchecked and every value "Not specified" |
| Age ratings | Questionnaire: no violence, no user-generated content shared with others, no purchases, no location sharing. "Does the app share the user's data with third parties?" — No. Expected rating: 3+ / Everyone |
| Packages | The `.msix` from the `msix` CI artifact of a final (non-rc) version |
| Submission options → Notes for certification | "Desktop (Win32) app with runFullTrust. It runs a local monitoring service, a tray icon, and the system's tracert and netsh tools. No account or sign-in is needed." |
| Restricted capability `runFullTrust` | Same text as the certification notes |

---

## Arabic

### Description

مراقب الإنترنت يراقب اتصالك بالإنترنت على مدار الساعة، ويسجّل كل انقطاع وكل بطء بالوقت والمدة، ثم يحوّل ذلك إلى دليل تقدّمه لمزوّد الخدمة في شكواك.

بدل أن تقول "النت بيقطع كتير"، تقول مثلاً: "انقطع الإنترنت 14 مرة هذا الشهر بإجمالي 6 ساعات، أطولها 47 دقيقة يوم 12 الساعة 9:40 مساءً، والسرعة الفعلية 38% من سرعة باقتي".

كيف يعمل:
• يفحص الاتصال كل بضع ثوانٍ بثلاث طرق مختلفة (TCP وHTTP وDNS)، فلا يخلط بين تعطّل موقع واحد وانقطاع حقيقي.
• يفرّق بين انقطاع من المزوّد ومشكلة في الراوتر أو الكابل أو الواي فاي أو انقطاع الكهرباء.
• يقيس السرعة ويقارنها بسرعة باقتك، ويقيس التأخير وقت الضغط على الخط.
• يجهّز نص الشكوى، والتقرير الشهري، وملف أدلة مختوماً يثبت أن السجلات لم تُعدَّل.

يعمل بهدوء في الخلفية من شريط المهام، وينبّهك فور انقطاع الاتصال وعودته.

خصوصيتك أولاً: لا حساب، ولا إعلانات، ولا تتبّع. كل بياناتك تبقى على جهازك.

### Product features

1. فحص الاتصال كل بضع ثوانٍ بثلاث طرق (TCP وHTTP وDNS) لتجنّب الإنذارات الكاذبة
2. سجلّ لكل انقطاع بوقت البداية والنهاية والمدة والسبب المرجّح
3. نص شكوى جاهز من بياناتك الفعلية، مع مشاركته عبر واتساب أو البريد
4. ملف أدلة بسجلات مختومة يومياً، مع ختم زمني مستقل اختياري يثبت أنها لم تُعدَّل
5. قياس السرعة ومقارنتها بباقتك، وتقييم التأخير تحت الضغط من A+ إلى F
6. تقرير شهري: نسبة التوفّر، وأطول انقطاع، وساعات الذروة، وقيمة الانقطاع من سعر باقتك
7. يحدّد مكان العطل: الراوتر أم شبكة المزوّد، بتتبّع المسار لحظة الانقطاع
8. يسجّل إن كنت على كابل أم واي فاي وقوة الإشارة، ويكشف انقطاع الكهرباء
9. تنبيهات على سطح المكتب وTelegram وDiscord وSlack وntfy، وملخّص أسبوعي
10. واجهة بالعربية والإنجليزية، بوضع داكن وفاتح
11. بلا حساب ولا تتبّع: كل البيانات تبقى على جهازك

### Search terms

مراقبة الإنترنت · انقطاع النت · قياس السرعة · شكوى مزود الخدمة · سرعة الإنترنت

---

## English

### Description

Internet Monitor watches your internet connection around the clock, records every outage and slowdown with its time and duration, and turns that into evidence you can send your provider with a complaint.

Instead of "my internet keeps dropping", you can say, for example: "It went down 14 times this month, 6 hours in total. The longest was 47 minutes on the 12th at 9:40 pm, and my real speed is 38% of my plan."

How it works:
• It checks the connection every few seconds in three different ways (TCP, HTTP and DNS), so one website being down isn't mistaken for an outage.
• It tells a provider outage apart from a router, cable, Wi-Fi or power problem.
• It measures your speed against your plan, and the lag when the line is busy.
• It writes the complaint text, a monthly report, and a sealed evidence pack that shows the logs weren't edited.

It runs quietly in the background from the system tray and alerts you the moment the connection drops and comes back.

Privacy first: no account, no ads, no tracking. All your data stays on your computer.

### Product features

1. Checks the connection every few seconds three ways (TCP, HTTP, DNS) to avoid false alarms
2. A log of every outage with start, end, duration and likely cause
3. A ready-to-send complaint written from your real data, shareable by WhatsApp or email
4. An evidence pack of daily-sealed logs, with an optional independent timestamp proving they weren't edited
5. Speed tests compared with your plan, plus a lag-under-load grade from A+ to F
6. Monthly report: uptime, longest outage, peak hours, and what the downtime is worth from your plan price
7. Shows where it broke, your router or the provider's network, with a route trace at the moment of the outage
8. Records cable vs Wi-Fi and signal strength, and detects power cuts
9. Alerts on the desktop, Telegram, Discord, Slack and ntfy, plus a weekly summary
10. English and Arabic interface, dark and light themes
11. No account, no tracking: all data stays on your computer

### Search terms

internet monitor · outage log · speed test · ISP complaint · connection monitor
