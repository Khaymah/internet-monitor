# بحث السوق والمنافسين: Internet Monitor

> مستند مرجعي اتعمل في نوفمبر 2026، عن طريق وكيلين من sabry-kit: `senior-pm` (المنافسين والـ roadmap) و`growth-strategist` (السوق والتوزيع والنمو).
> كل الأرقام عليها مصادر في آخر المستند. واللي ما اتأكدش منه مكتوب جنبه **(غير موثّق)**.

---

## 1. الخلاصة التنفيذية

| البند | الخلاصة |
| --- | --- |
| **السوق الأساسي** | مصر. فيها حوالي 12.6–13.1 مليون اشتراك إنترنت أرضي. وفي 2025 اتصعّدت حوالي **79 ألف شكوى** عن الإنترنت الأرضي للجهاز القومي لتنظيم الاتصالات (34,261 في النص الأول و45,114 في النص التاني) |
| **أقوى Job-to-be-done** | "عايز أثبت إن النت بيفصل، عشان آخد حقي من WE أو فودافون أو أورانج أو e&، أو أصعّد للجهاز" |
| **الفجوة اللي محدش مالكها** | **مفيش ولا أداة بتطلع ملف شكوى جاهز** مظبوط على إجراءات الجهاز القومي. **ومفيش أداة بتفرّق** بين انقطاع الكهرباء أو الجهاز المقفول وبين عطل المزوّد. **ومفيش أداة عربية** مخصوصة لتوثيق الانقطاعات |
| **أكبر خطر على مصداقية التقرير** | انقطاع الكهرباء (تخفيف الأحمال) والجهاز النايم كانوا بيظهروا على إنهم انقطاع أو فترة "فاضية" ملهاش تفسير. ده بيضعّف الدليل قدام المزوّد |
| **أكبر leak في التبنّي** | الـ binary مش موقّع، فبيطلع تحذير SmartScreen لمستخدم مش تقني |
| **قيد خاص بمصر** | الإنترنت الأرضي بباقات (quota). اختبار سرعة 30 ثانية على 90 Mbps بياكل حوالي 340MB. ولو مجدول كل ساعة يوصل لأكتر من 100GB في الشهر |

---

## 2. خريطة المنافسين

| المنتج | الفئة | نقطة القوة | السعر | نقطة الضعف بالنسبة لنا |
| --- | --- | --- | --- | --- |
| PingPlotter | الدعم الفني والمستخدم المتقدم | traceroute مرئي، و**snapshots يقدر المزوّد يفتحها** | حوالي $7–29 في الشهر، أو $40–349 مرة واحدة | معقد، ومفيهوش عربي، ومفيهوش تقرير شهري "يتقدّم" |
| Net Uptime Monitor | ويندوز في البيت | فحص الـ gateway ("الشبكة المحلية سليمة") | حوالي $10 | واجهته قديمة، والـ log نص خام. **فيه قصة موثّقة لمستخدم فسخ عقده بيه** (Digital Trends) |
| UptimeLog / Network Logger | ماك | تقرير PDF/CSV للمزوّد | حوالي $10–19 | ماك بس، ومفيهوش تحليل لكل طبقة |
| Uptime Kuma / Gatus | self-hosted وDevOps | monitors كتير وقنوات تنبيه كتير | مجاني | Docker أو YAML. **ولما النت يقع ما يقدرش يبعت تنبيه** |
| SmokePing | مهندسين الشبكات | رسوم loss/jitter واضحة | مجاني | إعداده يدوي على Linux |
| speedtest-tracker | self-hosted | اختبارات سرعة مجدولة | مجاني | بيقيس السرعة بس، ومفيش uptime لكل طبقة |
| Fing | البيت | **بيكشف انقطاع النت والكهرباء** | $8.99–19.99 في الشهر | الكشف في الباقة المدفوعة بس، ومش موجّه للشكاوى |
| Ookla Speedtest | الكل | اختبار بضغطة واحدة | مجاني | عند الطلب بس، مفيش مراقبة مستمرة |
| UniFi + Network Optimizer | أصحاب أجهزة UniFi | بيطلع PDF "ISP Health" | مع الهاردوير | لازم hardware معيّن |
| My NTRA | المستهلك المصري | قياس سرعة + شكاوى | مجاني | مفيهوش مراقبة مستمرة ولا سجل انقطاعات |

**الخلاصة:** السوق فيه أدوات تشخيص قوية (للمحترفين) وأدوات تسجيل بسيطة (قديمة). **المساحة اللي في النص فاضية:** أداة سهلة وعربية بتطلع **دليل يتصدّق** ومظبوط على الإجراء الرسمي.

---

## 3. المستخدم المصري: إجراء الشكوى الرسمي

1. تشتكي للشركة الأول وتاخد **رقم شكوى** (الشركة ملزمة ترد خلال 72 ساعة).
2. لو ما اتحلتش، تصعّد للجهاز القومي لتنظيم الاتصالات عن طريق **155**، أو تطبيق **My NTRA**، أو بوابة الشكاوى، أو WhatsApp.
3. البيانات المطلوبة: الاسم، والرقم القومي، **ورقم الخط الأرضي**، واسم الشركة، **ورقم الشكوى**.
4. الجهاز بيبت في الشكوى خلال **5 أيام عمل**، وبينصح **تحتفظ بأرقام الشكاوى وتواريخها واسم الموظف**.
5. **مفيش شرط رسمي لشكل الدليل التقني.** يعني الأداة اللي تحدد شكل الدليل هي اللي هتمتلك الفئة دي.
6. التعويض موجود لكنه بيتحدد حالة بحالة. مثلاً بعد حريق سنترال رمسيس في يوليو 2025 العملاء خدوا 50% من الباقة أو خصم 30%. **ووقت الأزمة، اللي عنده سجل دقيق هو اللي يقدر يطالب.**

> **تحفّظ على الـ positioning:** ما لقيناش أي مصدر بيقول إن المزوّد أو الجهاز بيقبل logs المستخدم كدليل ملزم.
> الأدق إننا نقول **"اشتكي بدليل"** بدل **"اكسب النزاع"**. ده قرار تسويقي متروك لصاحب المشروع.

---

## 4. الشرائح (Pain × Reachability)

| الشريحة | الألم | سهولة الوصول | الحكم |
| --- | --- | --- | --- |
| مستخدم بيتي مصري في نزاع مع مزوّده | عالي جداً | عالية (محتوى عربي، جروبات، يوتيوب) | **Beachhead** |
| Freelancers والشغل من البيت | عالي | متوسطة | شريحة تانية بنفس الرسالة |
| Gamers وstreamers | متوسط | متوسطة | ثانوي (تنبيهات Discord بتخدمهم) |
| الكافيهات والشركات الصغيرة | متوسط | ضعيفة | ⚠️ **Freelancer-Trap**: ما تبيعلهمش خدمات تركيب |
| IT support وMSPs | عالي | ضعيفة لشخص واحد | الشريحة الوحيدة اللي ممكن تبقى مصدر دخل متكرر، **بعد ما الطلب يتأكد** |

---

## 5. خارطة الطريق (RICE)

| # | الميزة | الحكم | الحالة |
| --- | --- | --- | --- |
| 1 | **ملف الشكوى (Evidence Pack)**: بيانات المزوّد والباقة، والسرعة المقاسة قدام سرعة الباقة، وتصدير مضغوط فيه **SHA-256** | **Build** | ✅ اتنفّذ |
| 2 | **تغطية المراقبة**: فصل "الجهاز كان مقفول أو نايم أو الكهرباء قاطعة" عن عطل المزوّد | **Build** | ✅ اتنفّذ |
| 3 | **متابعة الشكاوى**: رقم الشكوى، وعدّاد الـ 72 ساعة، وتذكير بالتصعيد لـ 155 | **Build** | ✅ اتنفّذ |
| 4 | **تقدير استهلاك الباقة** للاختبارات المجدولة | Build (quick win) | ✅ اتنفّذ |
| 5 | Outage timeline في الداشبورد + ملخص أسبوعي | Build بعد اللي فوق | ⬜ |
| 6 | قنوات ntfy وEmail وwebhook عام | Build لما يبقى فيه وقت | ⬜ |
| 7 | Bufferbloat / loaded latency | Defer | ⬜ |
| 8 | traceroute تلقائي وقت الانقطاع | Defer | ⬜ |
| 9 | Remote Heartbeat (تنبيه من برة لما النت يقع) | Defer + **fake-door test** | ⬜ |
| 10 | SQLite وPWA | Defer | ⬜ |
| — | خريطة انقطاعات crowdsourced أو ترتيب المزوّدين | **Kill**: محتاجة telemetry، وده ضد وعد "100% local" | — |

**North Star:** عدد ملفات الشكوى اللي اتصدّرت في الشهر. **Guardrail:** الانقطاعات اللي اتنسبت للمزوّد غلط والجهاز كان مقفول.

---

## 6. النمو والتوزيع (90 يوم)

| الترتيب | القناة | ليه |
| --- | --- | --- |
| 1 | **Microsoft Store (MSIX)** | التسجيل للأفراد بقى مجاني من سبتمبر 2025، والـ MSIX بتوقّعه Microsoft. **وده بيحل مشكلة SmartScreen.** (ملحوظة: نسخة الـ Store لازم تقفل الـ auto-updater) |
| 2 | **winget** + Scoop | سهل، وبيبني ثقة |
| 3 | **محتوى عربي evergreen** | صفحة + فيديو: "إزاي تشتكي لجهاز تنظيم الاتصالات بدليل" |
| 4 | AlternativeTo وReddit | المنافسة ضعيفة في فئة home outage logger |
| 5 | Show HN وProduct Hunt | **آخر حاجة.** و⚠️ لازم نوحّد التسمية على **"source-available"** الأول، لأن ELv2 مش رخصة OSI |

**ملاحظات:**
- SignPath Foundation وhomebrew-core بيشترطوا رخصة OSI، فالمشروع **مش مؤهّل** ليهم بالرخصة الحالية.
- GitHub Sponsors متاح في مصر، وتكلفته صفر. فعّله.
- **الـ monetization:** مفيش أي حاجة مدفوعة للأفراد دلوقتي. ممكن نفتح "Business waitlist" نقيس بيها الطلب على نسخة multi-site، وأي ميزة SaaS تكون **opt-in ومنفصلة** عن الـ core.
- ❌ **ممنوع** أي "anonymous usage stats" أو "crowd map". عدم وجود telemetry هو الـ differentiator الأساسي.

**KPIs من غير telemetry:** عدد تحميلات الـ release لكل asset (GitHub API)، والـ Stars، والـ Store acquisitions (Partner Center)، والـ referrers في GitHub Traffic، والـ clicks في Search Console للصفحة العربية، واستبيان اختياري: "التقرير نفعك في شكوتك؟".

---

## 7. افتراضات لازم تتختبر قبل ما نصرف أكتر

1. **(الأهم)** المزوّد والجهاز هيتعاملوا مع التقرير؟ الاختبار: 5 مقابلات + شكوى حقيقية واحدة فيها ملف الشكوى، ونوثّق الرد.
2. تصنيف "الجهاز كان مقفول" مقابل "انقطاع" دقيق كفاية؟ الاختبار: 3 أجهزة لمدة أسبوعين.
3. الناس هتدفع على Remote Heartbeat؟ الاختبار: fake-door في الإعدادات.

> **تصحيح لملاحظة في تقرير النمو:** الـ Releases ظاهرة ومنشورة فعلاً (آخر إصدار `v2.0.2` بتاريخ 31 أكتوبر 2026).

---

## المصادر (مختارة)

- NTRA complaints H1/H2 2025: [Hapi Journal](https://hapijournal.com/?p=389371) · [مصراوي](https://www.masrawy.com/news/news_economy/details/2025/9/1/2846707/)
- حقوق المستخدمين وخطوات الشكوى: [الوطن 2026](https://www.elwatannews.com/news/details/8331551) · [الوطن: خطوات الشكوى](https://www.elwatannews.com/news/details/8297373)
- غرامة Telecom Egypt: [NTRA](https://www.tra.gov.eg/en/ntra-imposes-fine-of-20-million-pounds-on-telecom-egypt-for-service-breakdown/)
- حريق رمسيس: [Wikipedia](https://en.wikipedia.org/wiki/Ramses_Exchange_fire) · [Ahram](https://english.ahram.org.eg/News/549405.aspx)
- اشتراكات الإنترنت الأرضي: [Ahram](https://english.ahram.org.eg/News/575154.aspx) · [DataReportal Egypt 2025](https://datareportal.com/reports/digital-2025-egypt)
- شكاوى الباقات: [العربية 2026](https://www.alarabiya.net/technology/in-focus/2026/01/25/)
- VDSL بعد انقطاع الكهرباء: [TP-Link community](https://community.tp-link.com/en/home/forum/topic/176434)
- المنافسين: [PingPlotter pricing](https://www.capterra.com/p/143523/PingPlotter/pricing/) · [Digital Trends: NUM](https://www.digitaltrends.com/computing/this-tool-proved-my-internet-was-disconnecting/) · [Fing pricing](https://www.fing.com/pricing/) · [UptimeLog](https://alternativeto.net/software/uptimelog/about/)
- إرشادات تجميع الدليل: [speedtesthq](https://speedtesthq.com/guides/diagnostics/document-internet-problem-for-isp) · [Ofcom compensation](https://selectra.co.uk/tv-broadband/news/broadband-outage-compensation-ofcom-what-provider-owes)
- التوزيع: [Microsoft Store free registration](https://blogs.windows.com/windowsdeveloper/2025/09/10/free-developer-registration-for-individual-developers-on-microsoft-store/) · [Code signing options](https://learn.microsoft.com/en-us/windows/apps/package-and-deploy/code-signing-options) · [SignPath terms](https://signpath.org/terms) · [Homebrew license guidelines](https://docs.brew.sh/License-Guidelines) · [GitHub Sponsors regions](https://github.blog/news-insights/company-news/github-sponsors-available-in-30-new-regions-2/)
