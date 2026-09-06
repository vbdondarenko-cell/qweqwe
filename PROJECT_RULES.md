# LinkUp — NON-NEGOTIABLE PROJECT RULES

Цей файл визначає головні незмінні правила розробки LinkUp. Вони мають вищий пріоритет за roadmap, приклади, історичні припущення та будь-які суперечливі формулювання в інших документах репозиторію.

## PROJECT IDENTITY

- Назва продукту: **LinkUp**.
- LinkUp створюється **з нуля (greenfield)** у цьому репозиторії.
- Старі версії, старі репозиторії, старий код і попередня архітектура LinkUp **не є джерелом правди та не повинні враховуватися**.
- Єдине джерело правди для продукту — **поточні файли цього репозиторію**, з урахуванням правил нижче.
- LinkUp — **повноцінна робоча соціальна мережа для Android та iOS**.
- Mobile architecture: **Kotlin Multiplatform (KMP)** для спільної бізнес-/domain-/data-логіки; **Jetpack Compose** для Android UI; **Swift/SwiftUI** для iOS UI та platform-specific API.
- **Render.com НЕ ВИКОРИСТОВУЄТЬСЯ.** Заборонено додавати Render-specific hosting, deployment configuration, URLs, documentation assumptions або runtime dependencies.
- **Google Cloud compute/build/deployment infrastructure НЕ ВИКОРИСТОВУЄТЬСЯ.** Заборонені Cloud Run, Cloud Deploy, Cloud Build, Artifact Registry, Google Cloud Logging/Monitoring, Secret Manager, Pub/Sub та інші GCP services як canonical LinkUp infrastructure, якщо користувач прямо не змінить це рішення пізніше. Firebase залишається дозволеним окремим platform service layer відповідно до правил нижче.
- **Canonical infrastructure stack: GitHub + Supabase + Firebase + окремий Ubuntu server.** GitHub є source-control authority; Supabase project `oavnrlwsfiiehluubwjk` є managed PostgreSQL/PostGIS infrastructure; Firebase дозволений для явно інтегрованих mobile/platform capabilities; окремий Ubuntu server є цільовим runtime/deployment environment для Go API після прямої команди користувача на deployment.
- **Supabase project `oavnrlwsfiiehluubwjk` є поточною managed PostgreSQL/PostGIS infrastructure для LinkUp.** Go API залишається єдиним application-facing authority для domain/auth/data flow; Supabase Auth/Data API/Realtime/Storage не є product authority або обов'язковою dependency `1.0.0`, якщо користувач прямо не змінить це пізніше. `service_role`, database credentials та інші server secrets ніколи не потрапляють у Android/iOS клієнти або Git history.
- **До окремої прямої команди користувача на deployment робота виконується тільки в GitHub-репозиторії.** Не підключатися до Ubuntu server, не копіювати туди код і не виконувати deploy/runtime changes завчасно.

## RULE 1 — EVERYTHING IN THE REPOSITORY MUST EXIST IN ITS TARGET RELEASE

Усе, що описано у файлах репозиторію як функція, поведінка, продуктова можливість, UX/UI-вимога, дизайн, анімація, архітектурна вимога, safety/privacy правило, тестова вимога або користувацький сценарій, **обов'язково має бути реально реалізовано в LinkUp у тій версії, до якої це віднесено canonical README roadmap**.

- Документація не є списком необов'язкових ідей.
- Не можна мовчки пропускати складні вимоги.
- Не можна замінювати вимоги декоративними макетами, fake/mock-даними або нефункціональними екранами.
- Якщо файли суперечать один одному, застосовуються ці NON-NEGOTIABLE PROJECT RULES; інший конфлікт має бути явно виправлений у репозиторії.
- Текст документації не обов'язково має відображатися в UI, але **вся продуктова поведінка, яку він вимагає для активної версії, повинна існувати та працювати до завершення цієї версії**.
- Вимоги майбутніх версій залишаються обов'язковим roadmap, але **не блокують Done попередньої версії**, якщо не є її прямою dependency.

## RULE 2 — ONLY A REAL WORKING SOCIAL NETWORK

Мета — не prototype, demo, showcase, skeleton або набір заготовок. Результат — **реально працююча end-to-end соціальна мережа LinkUp**.

Заборонено вважати функцію завершеною, якщо в ній є:

- `TODO`, `FIXME` або еквівалент незавершеної критичної роботи;
- fake/mock дані замість реального product flow;
- кнопки або екрани без робочої поведінки;
- hardcoded успішні відповіді замість реальної state/domain логіки;
- декоративні realtime/chat/map/auth/social interactions без функціонального end-to-end flow;
- формулювання на кшталт «зробимо потім», «закомітимо основу», «тимчасова заглушка» як спосіб оголосити роботу готовою.

Кожна завершена функція повинна мати необхідну domain/data/platform реалізацію, коректні стани помилок, реальну взаємодію та перевірки/тести відповідно до її ризику.

## RULE 3 — MANDATORY REPORT AFTER EVERY CHANGE

Після кожної виконаної роботи обов'язковий чіткий звіт у двох частинах.

### Я зробив:

- які саме файли, модулі, функції або конфігурації створено/змінено;
- що конкретно почало працювати;
- що виправлено;
- які перевірки та тести виконані і який їх результат.

### Треба ще:

- що реально залишилося незавершеним у **поточній активній версії**;
- який наступний обов'язковий крок;
- які є відомі блокери, ризики або залежності.

Заборонено писати лише «готово», якщо scope активної версії не завершений повністю.

## RULE 4 — ANDROID + IOS ARE FIRST-CLASS TARGETS; NO RENDER.COM

LinkUp розробляється як мобільна соціальна мережа для **Android та iOS**.

- Обидві платформи є first-class product targets.
- Shared business logic повинна використовувати KMP там, де це технічно доцільно та не погіршує native UX.
- Android UI: native **Jetpack Compose**.
- iOS UI: native **Swift/SwiftUI**.
- Platform-specific sensors, location, haptics, notifications, Live Activities, shaders та інші системні API реалізуються нативно для відповідної платформи.
- Не будувати WebView-first, web-first або desktop-first продукт замість Android/iOS застосунків.
- **Не використовувати Render.com ні для backend hosting, ні для deployment, ні як приховане припущення в документації чи коді.**

## RULE 5 — README MUST BE RE-READ BEFORE EVERY MAJOR WORK BLOCK

`README.md` є головним product/engineering contract LinkUp після цих NON-NEGOTIABLE PROJECT RULES.

Перед **кожним великим блоком реалізації** обов'язково потрібно перечитати актуальний `README.md` із поточного `main` і звірити заплановані зміни з ним.

Перед початком блоку потрібно перевірити щонайменше:

- **поточну active target version**;
- scope цієї версії та порядок залежностей;
- product behavior та domain contracts;
- privacy / anti-stalking / safety non-negotiables;
- API contract principles;
- required test matrix для цієї версії;
- Definition of Done;
- Android та iOS вимоги;
- realtime/offline/reconnect вимоги, якщо вони стосуються scope;
- design/product вимоги з інших актуальних файлів репозиторію.

Не можна покладатися лише на пам'ять про README або на старе прочитання. Якщо `README.md` змінився, для наступної роботи використовується **актуальна версія з `main`**.

Якщо active version вимагає фундаментальний dependency перед UI/feature scope, не можна перескакувати dependency лише заради швидшого видимого UI.

## RULE 6 — ALWAYS REPORT PRODUCTION READINESS

У кожному підсумковому статусі роботи по LinkUp потрібно вказувати **готовність активної цільової версії до production від 0% до 100%**.

- **0%** — фактично немає робочого scope активної версії.
- **100%** — весь scope активної версії реалізований, перевірений і реально готовий до production-релізу на Android та iOS.
- Відсоток не можна штучно підвищувати за документацію, scaffolding або декоративний UI; він зростає тільки за реально інтегровані та перевірені production capabilities.
- Ще не завершені capability-блоки README знижують readiness єдиної Version 1 пропорційно їхньому реальному production scope; readiness не можна рахувати лише за social foundation.

## RULE 7 — ALL DEVELOPMENT GOES DIRECTLY TO MAIN

Уся подальша розробка цього репозиторію виконується **тільки напряму в `main`**.

- Усі зміни коду, тестів, конфігурації, міграцій та документації потрібно комітити й пушити безпосередньо в `main`.
- Не створювати feature branches, agent branches, draft branches або Pull Requests для звичайної роботи.
- Не залишати завершені зміни в іншій гілці замість `main`.
- Окрему гілку або Pull Request дозволено створити **лише якщо користувач прямо попросив про це в поточному завданні**.
- Якщо інструмент або workflow за замовчуванням пропонує працювати через окрему гілку, це правило має пріоритет: використовувати `main`, якщо користувач явно не наказав інакше.

## RULE 8 — ONE PRODUCT VERSION; DELIVER BY DEPENDENCY-SAFE CAPABILITY BLOCKS

Активна і єдина product version зараз: **LinkUp Version 1 (`1.0.0`)**.

- Увесь product scope, описаний у поточному `README.md`, входить до **Version 1** і є обов'язковим до її production Done.
- Історичні semver-заголовки `1.0.0` → `2.12.0` у `README.md` зберігаються тільки як стабільні traceability IDs для capability-блоків; вони більше не означають окремі product releases.
- Вже реалізований social baseline не видаляється й залишається фундаментом Version 1: account → profile → PUBLIC + APPROVAL Slot → Pulse → REQUEST → APPROVE/REJECT → accepted-only temporary chat → START/COMPLETE або CANCEL → terminal chat purge → safety/privacy.
- Existing Instant access engine, realtime, City Context, outbox, locality та інші foundations зберігаються й розвиваються до повного scope Version 1.
- Об'єднання scope не дозволяє декоративні заглушки, fake/mock flows або передчасне оголошення Version 1 готовою.
- Реалізація виконується dependency-safe capability-блоками у порядку, визначеному `README.md`; складний unified scope не є причиною перескакувати фундаментальні backend/privacy dependencies.
- Кожний завершений блок має бути інтегрований і перевірений на Android та iOS у межах заявленої поведінки.
- Version 1 стає production-ready тільки після реалізації та перевірки **всього** README scope; частково завершені capability-блоки відображаються у звітах і readiness, але не створюють нових product version numbers.

## RULE 9 — PRESERVE EXISTING PRODUCT SCOPE

**Не прибирати наявний код, файли, migrations, тести, документацію, roadmap requirements або вже реалізовані/частково реалізовані feature blocks без прямої команди користувача.**

Якщо feature перенесена у майбутню version:

- вона залишається частиною LinkUp roadmap;
- уже написана реалізація зберігається;
- за потреби вона просто не входить у current release surface або контролюється feature flag;
- перенесення по версіях не є підставою для destructive cleanup;
- applied migrations залишаються forward-only;
- refactor/rename/move допускаються лише зі збереженням потрібної поведінки та даних.

Якщо колись виникне ситуація, де видалення справді необхідне, спочатку потрібно отримати **пряму команду користувача** на це конкретне видалення.

## RULE 10 — APK HANDOFF IS DIRECT TO CHAT; NEVER SUPABASE OR GOOGLE CLOUD

Android APK є release/test artifact, а не application data.

- **APK-файли заборонено зберігати в Supabase PostgreSQL, Supabase Storage або будь-якому іншому Supabase product.**
- APK-файли не комітяться в Git history.
- Готовий APK для користувача потрібно передавати **безпосередньо в активний ChatGPT-чат як файл**.
- **Google Cloud Build/Artifact Registry не використовуються навіть як тимчасовий transport layer для APK.** APK має збиратися лише в явно дозволеному development/build environment; Ubuntu server може використовуватися для build/runtime тільки після прямої команди користувача.
- **GitHub Actions artifacts для APK handoff не використовуються.**
- Для кожного APK handoff потрібно перевіряти підпис/цілісність та SHA-256 checksum.

## RULE 11 — CANONICAL INFRASTRUCTURE: GITHUB + SUPABASE + FIREBASE + UBUNTU; NO GOOGLE CLOUD

Canonical infrastructure LinkUp обмежується **GitHub + Supabase + Firebase + окремим Ubuntu server**.

- **GitHub використовується як source control / Git repository.** Уся розробка йде напряму в `main` відповідно до RULE 7.
- **GitHub Actions НЕ ВИКОРИСТОВУЄТЬСЯ** для build, test, lint, CI, CD, release, deploy, migration checks, APK/AAB handoff, artifact generation або будь-якого production/release gate, якщо користувач прямо не змінить це правило.
- Не створювати нові GitHub Actions workflows і не робити існуючі `.github/workflows/*` частиною canonical delivery process.
- Існуючі історичні GitHub Actions workflows, якщо вони залишаються в репозиторії, **не запускати, не вимагати їх green status і не вважати їх failures/limits blocker-ом LinkUp**.
- **Supabase** є canonical managed PostgreSQL/PostGIS infrastructure та production database layer відповідно до правил цього репозиторію.
- **Firebase** дозволений як canonical mobile/platform service layer тільки для реально інтегрованих capabilities LinkUp (наприклад push/device/test/crash services, якщо вони передбачені поточним product contract). Firebase не стає domain/data authority і не замінює Go API або PostgreSQL source of truth.
- **Окремий Ubuntu server** є canonical target runtime для Go API та server-side processes. Deployment на нього виконується лише після прямої команди користувача; до цього моменту сервер не чіпати.
- **Google Cloud compute/build/deployment services заборонені:** не використовувати Cloud Run, Cloud Deploy, Cloud Build, Artifact Registry, Google Cloud Logging/Monitoring, Secret Manager, Pub/Sub або інші GCP services як LinkUp infrastructure, CI/CD, runtime чи artifact transport, якщо користувач прямо не скасує цю заборону.
- Firebase дозволений цим правилом окремо; факт його належності Google не означає дозвіл на інші Google Cloud services.
- Release evidence та production readiness мають спиратися на реальні repository checks/tests, production Supabase, дозволені Firebase integrations, Ubuntu runtime після deployment та реальні Android/iOS/platform/device перевірки.
- Інший CI/CD, hosting або managed database provider не додається без прямої команди користувача.

## DEFINITION OF "DONE"

Слово «готово» дозволено тільки коли відповідний scope **активної версії**:

1. Реально реалізований.
2. Інтегрований у LinkUp.
3. Не залежить від fake/mock behavior для production flow.
4. Має коректні loading/content/empty/error/offline/reconnect стани там, де вони потрібні саме цій версії.
5. Перевірений відповідними unit/integration/platform/end-to-end тестами.
6. Не порушує privacy, safety та security правила репозиторію.
7. Працює на всіх заявлених для цього scope цільових платформах.
8. Відображений у звіті «Я зробив / Треба ще».
9. Відповідає Definition of Done відповідної версії в `README.md`.