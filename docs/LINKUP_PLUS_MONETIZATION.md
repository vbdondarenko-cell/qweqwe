# LinkUp+ — Monetization and Entitlement Contract

Статус: **canonical product contract для монетизації та складу LinkUp+**.

Документ визначає:

- стартову цінову сітку;
- єдиний набір переваг для всіх тривалостей підписки;
- межу між безкоштовним LinkUp і LinkUp+;
- rewarded/referral правила;
- вимоги до billing, entitlement і UX;
- фактичну різницю між target scope та поточною реалізацією.

LinkUp+ належить до **v1.2**. Активний release не можна блокувати незавершеним LinkUp+, а LinkUp+ не можна продавати як готовий продукт, поки оплачувані переваги та store/server verification не працюють end-to-end. `PROJECT_RULES.md` і `README.md` мають вищий пріоритет щодо release ordering, security, server authority та platform boundaries.

## 1. Продуктове позиціонування

**LinkUp+ — це розширений набір інструментів для активніших учасників, організаторів і мандрівників.**

Користувач платить за:

- ширше та точніше керування пошуком реальних активностей;
- потужніші інструменти організації LINK;
- планування активностей в інших містах і часових поясах;
- автоматизацію повторюваних дій із жорсткими safety/rate-limit межами;
- приватну self-only аналітику;
- розширену персоналізацію;
- додаткові convenience/privacy режими, які не підміняють базову безпеку.

Усі плани відкривають **однаковий LinkUp+ entitlement**. Відрізняється лише тривалість і ціна. Заборонено створювати приховані feature-відмінності між 7 днями, місяцем, 3 місяцями та роком.

## 2. Стартові ціни

Це рекомендована **launch pricing сітка для раннього етапу**, а не доведений оптимум. Її потрібно перевіряти на реальних даних paywall funnel, purchase conversion, renewals, refunds, churn і використанні кожної premium capability.

| План | Ціна користувача | Ефективно за місяць | Економія відносно місячного плану |
|---|---:|---:|---:|
| 7 днів | **39,99 грн** | не застосовується | короткий гнучкий доступ |
| 1 місяць | **99,99 грн** | **99,99 грн** | базова ціна |
| 3 місяці | **249,99 грн** | **83,33 грн/міс.** | **49,98 грн / 16,7%** |
| 12 місяців | **799,99 грн** | **66,67 грн/міс.** | **399,89 грн / 33,3%** |

Рекомендований порядок у paywall:

1. **3 місяці — рекомендований стартовий вибір**;
2. 1 місяць — базовий вибір;
3. 12 місяців — максимальна економія;
4. 7 днів — короткий доступ без нав'язування довгого зобов'язання.

Правила показу ціни:

- показувати повну суму списання, період і auto-renewal/prepaid type до підтвердження;
- не називати оплачувані 7 днів “free trial”;
- не використовувати фальшивий countdown, fake discount або приховане автопродовження;
- реальна локальна store price є authority для checkout UI; серверний catalog не може підміняти ціну, повернену Google Play/App Store;
- зовнішні product/base-plan IDs не вигадуються в документації;
- перед зміною ціни аналізувати реальні cohort retention, conversion, churn, refund rate, net revenue після store fee/tax та feature usage;
- не підвищувати ціну тільки через календарну дату або бажаний revenue без продуктового сигналу.

## 3. Що входить у LinkUp+

Нижче описаний **повний target entitlement v1.2**. У production paywall дозволено обіцяти лише capabilities, які реально ввімкнені сервером і пройшли відповідний release gate.

### 3.1. Travel Pro

- Global Astral Jump — перегляд підтримуваних міст без підміни фізичної присутності;
- remote exploration підтримуваних locality;
- чітке розділення physical і astral state;
- Pre-Flight Slots;
- розширений горизонт планування подорожей і scheduled LINK;
- timezone/DST-safe planning;
- Voice Babel: тимчасове speech-to-text і переклад для координації;
- відсутність постійного voice archive.

### 3.2. Host Power Tools

- Mega-Slots до серверного product ceiling;
- масштабований roster із pagination/realtime;
- Stealth Slots з invite-link-only visibility;
- opaque revocable invite tokens;
- розширені Slot Blueprints;
- Co-Hosting;
- granular organizer permissions;
- owner protection;
- audited grants/revokes;
- шаблон ніколи не клонує старих учасників, приватні адреси або токени.

### 3.3. Advanced Discovery

- Radius Overdrive у межах server-defined radius;
- Vibe-Match Filters;
- фільтри за мовою, інтересами, assets і accessibility compatibility;
- Auto-Pilot Join із явною згодою, caps, cooldowns і dedupe;
- для Approval Slot автоматизація створює тільки REQUEST;
- для Instant Slot auto-JOIN можливий лише після explicit opt-in;
- Pulse Time-Machine;
- historical privacy-safe city/map aggregates;
- заборона реконструкції індивідуальних маршрутів.

### 3.4. Privacy and Quality of Life

- Ghost Mode як розширений режим керування видимістю;
- відсутність passive public presence/BPM contribution під час Ghost Mode;
- Priority Boarding як візуальний/request aid без переваги за місце або waitlist;
- Oops-Rewind через compensating commands, а не видалення audit history;
- Multi-Threading / intent groups;
- atomic winner lock;
- відсутність silent cancellation після зовнішнього commitment;
- Guardian Auto-Ping для заздалегідь визначених trusted contacts;
- ніколи не вмикати `LIVE_PRECISE` без окремої явної згоди.

### 3.5. Identity, Analytics and Personalization

- Hex-Aura як cosmetic-only оформлення;
- BUMP Vault Pro: self-only аналітика;
- unique people met;
- repeat BUMPs;
- verified meetup count;
- activity/city trends без exact route reconstruction;
- Custom App Icons;
- OLED Black / Neon Cyberpunk class themes;
- contrast, font scaling і Reduce Motion сумісність.

Косметика та оплата не дають ranking, BPM, Hotspot, trust, moderation або capacity переваги.

### 3.6. Forgiveness

- один non-stacking eligible No-Strike Forgiveness credit на policy period;
- вихідна cancellation event лишається auditable;
- forgiveness не діє на no-show, safety або moderation порушення;
- reliability modifier обмежений server policy.

## 4. Що завжди залишається безкоштовним

LinkUp+ не може paywallити core social network або базову безпеку.

Безкоштовними лишаються:

- реєстрація, login, profile і session restore;
- створення базового LINK;
- перегляд базового Pulse/Map scope;
- REQUEST/JOIN/LEAVE, host approval/rejection і basic roster;
- accepted-only Zero-Trace coordination chat;
- block, report, moderation appeals та emergency/safety entry points;
- Privacy Center і базові visibility controls;
- приховування exact location від незнайомців;
- базові accessibility controls;
- manual guardian/safety action, якщо вона потрібна для особистої безпеки;
- cancellation та керування підпискою.

Важлива межа: **базова приватність безкоштовна**. LinkUp+ може давати automation, scheduling або extended convenience навколо Ghost/Guardian, але не може змушувати платити за припинення небажаного стеження, блокування людини, приховування точної геолокації чи виклик допомоги.

## 5. Мінімальний склад першого платного релізу

Не можна запускати billing лише з красивим paywall. Перед першою реальною оплатою мають працювати щонайменше:

- store purchase + restore;
- server-side receipt/purchase verification;
- entitlement states `ACTIVE / GRACE / BILLING_RETRY / EXPIRED / REVOKED`;
- refund/revoke/renewal/account-switch handling;
- subscription management/cancellation entry;
- щонайменше один реально корисний shipped блок із Host Power Tools;
- щонайменше один реально корисний shipped блок із Advanced Discovery;
- щонайменше один shipped блок із Analytics або Personalization;
- чесний feature availability list із server capability flags;
- Android/iOS UI не показує unavailable target features як already included and active.

Travel Pro, Voice Babel, Auto-Pilot, Pulse Time-Machine, Ghost automation або Guardian Auto-Ping можуть з'являтися пізніше, але до активації їх не можна використовувати як неправдиву причину купівлі.

## 6. Rewarded LinkUp+ Free Day

Product policy:

- rewarded quest доступний тільки всередині LinkUp+ surface;
- **5** server-verified rewarded views;
- мінімум **4 години** між зарахованими кроками;
- quest window: **24 години**;
- п'ятий verified view атомарно надає **24 години LinkUp+**;
- cooldown: **7 днів після завершення earned grant**;
- multi-device state синхронізується через сервер;
- paid subscription має пріоритет; reward не stack-иться поверх оплачуваного періоду;
- provider outage/no-fill не створює fake completion.

Заборонено:

- ads у Pulse, Map, LINK, Fly, chat або safety flows;
- forced interstitials;
- вимогу click/install/purchase для зарахування;
- precise social/location targeting;
- rewarded flow у probable driver state;
- client-authoritative `watched=true`.

Canonical tracking state:

- `last_video_watched_at`;
- `videos_watched_count` (`0..5`);
- `last_free_premium_claimed_at`.

Raw provider receipts/tokens не зберігаються й не логуються. Зберігаються hash та нормалізовані verified facts.

## 7. Referral program

Referral кваліфікується лише після server-verified paid subscription запрошеного користувача протягом **14 днів** після реєстрації його account.

| Qualified referrals | Нагорода inviter | Нагорода triggering invitee |
|---:|---:|---:|
| 1 | 1 день LinkUp+ | 1 день LinkUp+ |
| 3 | 7 днів LinkUp+ | 3 дні LinkUp+ |
| 5 | 30 днів LinkUp+ | 7 днів LinkUp+ |
| 10 | 90 днів LinkUp+ + badge/status | 7 днів LinkUp+ |

Правила:

- кожний milestone видається inviter лише один раз;
- один invitee може прив'язати лише одного inviter;
- повторне прив'язування того самого code idempotent;
- self-referral заборонений;
- deadline обчислюється від server `app_users.created_at`;
- binding сам по собі не дає reward;
- refund/revocation до qualification не може створити paid referral;
- leaderboard reward не видається, доки окремо не визначені amount, ranking window, ties і anti-fraud policy.

## 8. Server authority and anti-fraud

- mobile client ніколи не надає LinkUp+ локально;
- billing/rewarded/referral qualification вирішує Go;
- store/provider price є checkout authority, а Go перевіряє product, purchase state і entitlement;
- purchase/ad tokens є secrets і не логуються;
- duplicate provider events idempotent;
- refund/revoke/chargeback/grace представлені server-side;
- новий grant ніколи не скорочує вже довший valid grant;
- direct Supabase `anon`/`authenticated` access до monetization tables заборонений;
- entitlement не дає обхід block, moderation, safety, capacity, lifecycle або privacy rules;
- capability flags fail closed.

## 9. Android and iOS UX contract

LinkUp+ відкривається з **Me** і не змінює `Pulse · Map · LINK · Fly · Me`.

Surface показує:

- чіткий список доступних зараз переваг;
- 7-day, monthly, 3-month і annual plans;
- одну й ту саму entitlement для всіх durations;
- current status, expiry і billing state;
- повну суму списання та renewal/prepaid behavior;
- effective monthly price і реальну економію;
- purchase, restore і manage/cancel actions;
- rewarded progress, next step і cooldown;
- own referral code, binding deadline, milestones і qualified count;
- unavailable/fail-closed state, якщо store/provider/server verification не готові.

Заборонені fake activation, fake ad completion, fake referral count, hardcoded checkout price як заміна store price та client-authoritative entitlement.

## 10. Current repository implementation status — 2026-09-08

У `main` уже є early v1.2 foundation:

### Database

`db/migrations/000009_linkup_plus_monetization.sql` містить таблиці для:

- premium grants;
- rewarded progress і hashed verified rewarded receipts;
- verified subscription receipts;
- referral codes і invitee-to-inviter binding;
- one-time referral milestone awards.

Direct access для `PUBLIC`, `anon` і `authenticated` revoked.

### Go

Наявні:

- `GET /v1/me/monetization`;
- referral binding endpoint;
- server-backed status;
- referral codes/deadlines/milestones;
- fail-closed provider capability flags.

Поточний Go catalog **ще застарілий відносно цього контракту**:

- повертає тільки `monthly` і `annual`;
- використовує 149,99 грн/місяць та 1 199,88 грн/рік;
- не має 7-day і 3-month plans;
- не повертає feature availability/entitlement-benefit catalog.

### Android

Наявні model/client/native LinkUp+ surface, status, old plan catalog, rewarded progress і referrals.

Потрібно оновити:

- generic rendering усіх чотирьох plan durations;
- launch prices;
- store-returned localized checkout prices;
- benefits/availability section;
- purchase/restore/manage actions;
- server verification flow.

### iOS

Наявні Swift models, API/session binding, LinkUp+ view, status, plan list, rewarded progress і referrals.

Потрібно оновити:

- коректні labels для `WEEK`, `MONTH`, `THREE_MONTH`, `YEAR`;
- усі чотири plan durations;
- localized strings;
- StoreKit purchase/restore/manage;
- App Store server verification;
- benefits/availability section.

## 11. Required synchronization after approval

Перед production activation цього pricing contract потрібно синхронно оновити:

- `README.md` §7.7: `WEEKLY / MONTHLY / THREE_MONTH / ANNUAL`;
- Go catalog constants, plan model і policy tests;
- Android parser, UI labels, tests і localization;
- iOS parser, UI labels, tests і localization;
- Google Play base plans/offers;
- App Store subscription products/group;
- server product allowlist і provider verification adapters;
- paywall analytics events без sensitive/social/location payload;
- implementation ledger після фактично виконаних змін.

Зміна цього документа **не означає**, що billing уже активований або що target LinkUp+ features реалізовані.
