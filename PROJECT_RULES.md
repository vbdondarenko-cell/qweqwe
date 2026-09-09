# LinkUp — NON-NEGOTIABLE PROJECT RULES

Цей файл визначає головні незмінні правила розробки LinkUp. Вони мають вищий пріоритет за roadmap, приклади, історичні припущення та будь-які суперечливі формулювання в інших документах репозиторію.

## PROJECT IDENTITY

- Назва продукту: **LinkUp**.
- LinkUp створюється **з нуля (greenfield)** у цьому репозиторії.
- Старі версії, старі репозиторії, старий код і попередня архітектура LinkUp **не є джерелом правди та не повинні враховуватися**.
- Єдине джерело правди для продукту — **поточні файли цього репозиторію**, з урахуванням правил нижче.
- **Поточний активний product target — Android. iOS залишається майбутньою платформою, але повністю заморожений до окремої прямої команди користувача.**
- **Поточний дизайн, уже записаний у репозиторії, є canonical visual/UI contract. Його не переробляти, не редизайнити, не замінювати іншою дизайн-системою і не використовувати архітектурний refactor як причину змінювати його зовнішній вигляд або UX.**
- Увесь новий production functionality після цього правила реалізується **Kotlin для Android/client-side logic** та **Go для backend/server-side logic**. Нову product/domain functionality на React/TypeScript не переносити і не будувати там як альтернативну production implementation.
- Android UI та Android platform integration реалізуються на **Kotlin + Jetpack Compose** там, де потрібна production Android implementation, із збереженням уже затвердженого дизайну без самовільного redesign.
- **iOS/Swift/SwiftUI код, конфігурацію, проєктні файли, signing, build settings, тести та platform integrations не створювати, не змінювати, не видаляти і не рефакторити до прямої команди користувача “працювати над iOS” або еквівалентної.**
- Kotlin Multiplatform допускається лише там, де це не потребує змін iOS target і не створює iOS work block. До активації iOS Android-first Kotlin implementation має пріоритет.
- **Render.com НЕ ВИКОРИСТОВУЄТЬСЯ.** Заборонено додавати Render-specific hosting, deployment configuration, URLs, documentation assumptions або runtime dependencies.
- **Google Cloud compute/build/deployment infrastructure НЕ ВИКОРИСТОВУЄТЬСЯ.** Заборонені Cloud Run, Cloud Deploy, Cloud Build, Artifact Registry, Google Cloud Logging/Monitoring, Secret Manager, Pub/Sub та інші GCP services як canonical LinkUp infrastructure, якщо користувач прямо не змінить це рішення пізніше. Firebase залишається дозволеним окремим platform service layer відповідно до правил нижче.
- **Canonical infrastructure stack: GitHub + Supabase + Firebase + окремий Ubuntu server.** GitHub є source-control authority; Supabase project `oavnrlwsfiiehluubwjk` є managed PostgreSQL/PostGIS infrastructure; Firebase дозволений для явно інтегрованих mobile/platform capabilities; окремий Ubuntu server є цільовим runtime/deployment environment для Go API після прямої команди користувача на deployment.
- **Supabase project `oavnrlwsfiiehluubwjk` є поточною managed PostgreSQL/PostGIS infrastructure для LinkUp.** Go API залишається єдиним application-facing authority для domain/auth/data flow; Supabase Auth/Data API/Realtime/Storage не є product authority або обов'язковою dependency `v1.0`, якщо користувач прямо не змінить це пізніше. `service_role`, database credentials та інші server secrets ніколи не потрапляють у Android/iOS клієнти або Git history.
- **До окремої прямої команди користувача на deployment робота виконується тільки в GitHub-репозиторії.** Не підключатися до Ubuntu server, не копіювати туди код і не виконувати deploy/runtime changes завчасно.

## RULE 1 — EVERYTHING IN THE REPOSITORY MUST EXIST IN ITS TARGET RELEASE

Усе, що описано у файлах репозиторію як функція, поведінка, продуктова можливість, UX/UI-вимога, дизайн, анімація, архітектурна вимога, safety/privacy правило, тестова вимога або користувацький сценарій, **обов'язково має бути реально реалізовано в LinkUp у тій версії, до якої це віднесено canonical README roadmap**, з урахуванням тимчасового Android-only platform gate цього файла.

- Документація не є списком необов'язкових ідей.
- Не можна мовчки пропускати складні вимоги.
- Не можна замінювати production functionality декоративними макетами, fake/mock-даними або нефункціональними екранами.
- **Наявний дизайн не вважається “mock, який треба переписати”: він є canonical design contract. Заборона fake/mock behavior стосується production data/domain behavior, а не дозволу на redesign існуючого UI.**
- Якщо файли суперечать один одному, застосовуються ці NON-NEGOTIABLE PROJECT RULES; інший конфлікт має трактуватися відповідно до цих правил.
- Якщо README вимагає iOS parity або iOS implementation, ця вимога **не активується і не блокує Android development/readiness**, доки користувач прямо не розблокує iOS.
- Текст документації не обов'язково має відображатися в UI, але вся продуктова поведінка активного Android/Go scope повинна існувати та працювати до завершення відповідного scope.

## RULE 2 — ONLY A REAL WORKING SOCIAL NETWORK

Мета — не prototype, demo, showcase, skeleton або набір заготовок. Результат активної реалізації — **реально працююча end-to-end соціальна мережа LinkUp на Android із Go backend**.

Заборонено вважати production functionality завершеною, якщо в ній є:

- `TODO`, `FIXME` або еквівалент незавершеної критичної роботи;
- fake/mock дані замість реального product flow;
- кнопки або екрани без робочої поведінки там, де вони є частиною production flow;
- hardcoded успішні відповіді замість реальної state/domain логіки;
- декоративні realtime/chat/map/auth/social interactions без функціонального end-to-end flow;
- формулювання на кшталт «зробимо потім», «закомітимо основу», «тимчасова заглушка» як спосіб оголосити production functionality готовою.

Кожна завершена функція повинна мати необхідну Kotlin/Go domain/data/platform реалізацію, коректні стани помилок, реальну взаємодію та перевірки/тести відповідно до її ризику.

## RULE 3 — MANDATORY REPORT AFTER EVERY CHANGE

Після кожної виконаної роботи обов'язковий чіткий звіт у двох частинах.

### Я зробив:

- які саме файли, модулі, функції або конфігурації створено/змінено;
- що конкретно почало працювати;
- що виправлено;
- які перевірки та тести виконані і який їх результат.

### Треба ще:

- що реально залишилося незавершеним у **поточному активному Android/Go scope**;
- який наступний обов'язковий крок;
- які є відомі блокери, ризики або залежності.

Заборонено писати лише «готово», якщо активний scope не завершений повністю.

## RULE 4 — ANDROID IS ACTIVE; IOS IS FROZEN UNTIL DIRECT USER COMMAND

- **Android є єдиною активною mobile platform для поточної розробки.**
- Новий Android/client functionality пишеться на **Kotlin**; Android UI/platform code — **Kotlin + Jetpack Compose**.
- Backend/domain authority та server-side functionality пишуться на **Go** відповідно до canonical backend contract.
- Існуючий дизайн зберігається як є; Android implementation повинна відтворювати/використовувати цей design contract, а не замінювати його новим дизайном.
- **iOS повністю frozen. Без прямої команди користувача заборонено:** створювати або змінювати Swift/SwiftUI; створювати або змінювати Xcode project/workspace; змінювати iOS resources/assets; додавати iOS-specific dependencies; змінювати signing/provisioning; писати або міняти iOS tests; виконувати iOS build/migration/refactor/parity work.
- README або інший документ не може самостійно розблокувати iOS. Розблокування можливе тільки прямою командою користувача.
- До такого розблокування відсутність iOS implementation, iOS parity або iOS tests **не є blocker-ом Android Version 1 development/readiness**.
- Не будувати WebView-first, web-first або desktop-first production product замість Android застосунку. Існуючий React/TypeScript design layer може залишатися design reference, але новий production functionality не реалізується там як окрема authority.
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
- required test matrix для активного Android/Go scope;
- Definition of Done;
- Android вимоги;
- realtime/offline/reconnect вимоги, якщо вони стосуються scope;
- design/product вимоги з інших актуальних файлів репозиторію.

**iOS-вимоги в README перечитуються лише для розуміння майбутнього compatibility contract, але не виконуються, не змінюють поточний work block і не є blocking requirement, доки користувач прямо не активує iOS.**

Не можна покладатися лише на пам'ять про README або на старе прочитання. Якщо `README.md` змінився, для наступної роботи використовується актуальна версія з `main`.

Якщо active Android/Go scope вимагає фундаментальний dependency перед feature scope, не можна перескакувати dependency лише заради швидшого видимого результату.

## RULE 6 — ALWAYS REPORT PRODUCTION READINESS

У кожному підсумковому статусі роботи по LinkUp потрібно вказувати **готовність поточного активного Android + Go release target до production від 0% до 100%**.

- **0%** — фактично немає робочого active release scope.
- **100%** — весь scope поточного активного release (`v1.0`, а після його завершення `v1.1`) реалізований, перевірений і реально готовий до production-релізу на Android із production Go backend.
- Відсоток не можна штучно підвищувати за документацію, scaffolding або декоративний UI; він зростає тільки за реально інтегровані та перевірені production capabilities.
- **Майбутній release scope `v1.1` не знижує readiness активного `v1.0`, доки він не активований відповідно до README release train.**
- Після переходу на наступний release readiness оцінюється для його повного active Android/Go scope разом із regression-вимогами попередніх releases.
- **iOS readiness, parity, build або tests не враховуються в поточний readiness і не знижують його, доки iOS frozen.**

## RULE 7 — ALL DEVELOPMENT GOES DIRECTLY TO MAIN

Уся подальша розробка цього репозиторію виконується **тільки напряму в `main`**.

- Усі зміни коду, тестів, конфігурації, міграцій та документації потрібно комітити й пушити безпосередньо в `main`.
- Не створювати feature branches, agent branches, draft branches або Pull Requests для звичайної роботи.
- Не залишати завершені зміни в іншій гілці замість `main`.
- Окрему гілку або Pull Request дозволено створити **лише якщо користувач прямо попросив про це в поточному завданні**.
- Якщо інструмент або workflow за замовчуванням пропонує працювати через окрему гілку, це правило має пріоритет: використовувати `main`, якщо користувач явно не наказав інакше.

## RULE 8 — RELEASE TRAIN v1.0 → v1.1; DELIVER BY DEPENDENCY-SAFE BLOCKS

Поточний активний product release: **LinkUp v1.0**.

Canonical README визначає два послідовні Version 1 releases: **v1.0 → v1.1**.

- Кожний release має власний scope і Definition of Done у `README.md`.
- **v1.0 не блокується незавершеним scope v1.1.**
- **v1.1 активується тільки після green v1.0 foundation**, якщо користувач прямо не змінить порядок.
- Capability, реалізована раніше свого release, не видаляється; вона просто проходить повний regression/DoD у своєму release gate.
- Вже реалізований social baseline не видаляється й залишається фундаментом: account → profile → PUBLIC + APPROVAL Slot → Pulse → REQUEST → APPROVE/REJECT → accepted-only temporary chat → START/COMPLETE або CANCEL → terminal chat purge → safety/privacy.
- Existing Instant access engine та інші canonical domain foundations не видаляються тільки через те, що їх user-facing surface належить пізнішому release.
- Release train не дозволяє decorative stubs, fake/mock production flows або передчасне оголошення release готовим.
- Реалізація виконується dependency-safe capability-блоками у порядку, визначеному `README.md`, тільки для активного Android/Go scope.
- Кожний завершений блок має бути інтегрований і перевірений на Android та backend у межах заявленої поведінки.
- **iOS implementation/parity не виконується і не є gate для v1.0/v1.1 Android readiness, доки користувач прямо не розблокує iOS.**
- Новий product release number поза `v1.0/v1.1` не додається без прямої команди користувача.

## RULE 9 — PRESERVE EXISTING PRODUCT SCOPE

**Не прибирати наявний код, дизайн, файли, migrations, тести, документацію, roadmap requirements або вже реалізовані/частково реалізовані feature blocks без прямої команди користувача.**

- **Особливо заборонено “очищати”, переписувати, переносити або замінювати поточний дизайн лише тому, що production functionality реалізується на Kotlin/Go.**
- Існуючий React/TypeScript design code може залишатися в репозиторії як canonical design source/reference і не є підставою для redesign.
- Якщо capability тимчасово неактивна через platform gate, її product requirement залишається частиною LinkUp roadmap.
- уже написана реалізація зберігається;
- за потреби неактивна capability просто не входить у current release surface або контролюється feature flag;
- platform deferral не є підставою для destructive cleanup;
- applied migrations залишаються forward-only;
- refactor/rename/move допускаються лише зі збереженням потрібної поведінки та даних і **не повинні змінювати затверджений дизайн без прямої команди користувача**.

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
- Release evidence та production readiness мають спиратися на реальні repository checks/tests, production Supabase, дозволені Firebase integrations, Ubuntu runtime після deployment та реальні **Android** platform/device перевірки.
- Інший CI/CD, hosting або managed database provider не додається без прямої команди користувача.

## RULE 12 — DESIGN IS FROZEN; NEW FUNCTIONALITY IS KOTLIN + GO

Це правило має пріоритет над будь-яким старим формулюванням у README або інших файлах, яке можна трактувати як вимогу переписати чи замінити поточний дизайн.

- **Поточний дизайн LinkUp у репозиторії затверджений і залишається таким, як написаний.**
- Не робити redesign, facelift, visual cleanup, component-system replacement, UX rewrite або “native reinterpretation” без прямої команди користувача.
- Не переносити нову production business/domain functionality у React/TypeScript. React/TypeScript design implementation не є новим domain authority.
- Новий Android/client functionality: **Kotlin**.
- Новий backend/server functionality: **Go**.
- Database changes виконуються через canonical PostgreSQL/PostGIS migration flow відповідно до README та цих правил.
- Якщо для підключення production functionality до затвердженого UI потрібно створити Kotlin/Compose representation, вона повинна зберігати дизайн, структуру, semantics і поведінку UI настільки точно, наскільки дозволяє Android platform, без самовільного redesign.
- **iOS не чіпати ні під приводом parity, ні під приводом shared architecture, ні під приводом майбутньої сумісності.** Чекати прямої команди користувача.

## DEFINITION OF "DONE"

Слово «готово» дозволено тільки коли відповідний **активний Android/Go scope**:

1. Реально реалізований.
2. Інтегрований у LinkUp.
3. Не залежить від fake/mock behavior для production flow.
4. Має коректні loading/content/empty/error/offline/reconnect стани там, де вони потрібні саме цьому scope.
5. Перевірений відповідними unit/integration/Android/end-to-end тестами.
6. Не порушує privacy, safety та security правила репозиторію.
7. Працює на всіх **активних** для цього scope цільових платформах; до окремого розблокування єдиною активною mobile platform є Android.
8. Не змінює затверджений дизайн без прямої команди користувача.
9. Відображений у звіті «Я зробив / Треба ще».
10. Відповідає Definition of Done відповідного active release/capability block у `README.md`, крім iOS-specific gates, які frozen цими правилами до прямої команди користувача.
