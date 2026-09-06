# LinkUp — UNIFIED VERSION 1 PRODUCT & ENGINEERING CONTRACT

Статус: **canonical product/release contract після `PROJECT_RULES.md`**.

Поточна й єдина цільова версія: **LinkUp Version 1 (`1.0.0`) — Unified Product Scope**.

Увесь scope цього README входить до **єдиного релізу LinkUp Version 1.0.0**. Історичні заголовки `VERSION 1.0.0` → `VERSION 2.12.0` нижче збережено лише як traceability IDs і dependency-ordered capability blocks; вони **не є окремими product releases і не дозволяють відкладати функціональність за межі `1.0.0`**.

> **CANONICAL VERSION RULE:** кожна вимога, feature, invariant, UI surface, backend contract, migration/test requirement і Definition of Done, описані будь-де в цьому `README.md`, є обов'язковою частиною **LinkUp Version 1.0.0**. Формулювання нижче на кшталт «later version», «target version», «пізніша version» або історичний номер capability-block не змінюють release scope: вони означають лише внутрішній dependency order усередині того самого `1.0.0`. Повний `1.0.0` не вважається Done, доки не закритий увесь README scope.

Основна навігація продукту: **Pulse · Map · LINK · Fly · Me**.

---

# 0. ГОЛОВНІ ПРАВИЛА VERSION 1 DELIVERY

LinkUp Version 1 містить увесь описаний нижче product scope. Щоб не ламати server authority, privacy та Android/iOS parity, одна product version реалізується послідовними dependency-safe capability-блоками.

## 0.1. Нічого не викидаємо

Цей roadmap є **dependency-ordered розподілом вимог усередині одного релізу `1.0.0`**, а не поділом на окремі product releases і не скороченням продукту.

- Жодна описана можливість LinkUp не може бути перенесена за межі `1.0.0` лише через історичний номер або пізніший capability-block.
- Жодну вже реалізовану або частково реалізовану фічу не потрібно видаляти тільки тому, що вона не входить у поточний release surface.
- Capability дозволено реалізовувати dependency-safe блоками і тимчасово тримати за feature flag під час розробки, але перед фінальним `1.0.0` Done весь обов'язковий README scope має бути інтегрований у release surface там, де README цього вимагає.
- Applied migrations залишаються forward-only.
- Історичне переміщення фічі між capability-блоками змінює лише **внутрішній порядок реалізації**, а не release target: release target завжди `1.0.0`.

## 0.2. Межі Version 1

- Єдина активна product version: **LinkUp Version 1 (`1.0.0`)**.
- Усі capability-блоки від історичного `1.0.0` до `2.12.0` входять до обов'язкового scope саме **LinkUp `1.0.0`** і мають бути закриті до фінального Version 1 Definition of Done.
- Нові product version numbers не створюються для проміжних capability-блоків.
- Історичні номери залишаються в заголовках і тестових назвах лише для traceability та безпечної міграції наявного коду/даних.

## 0.3. Active capability-block rule

У кожен момент є одна active product version і один основний dependency-safe capability block усередині неї.

Поки Version 1 не виконала свій Definition of Done:

- не перескакувати на далекі красиві фічі;
- не роздувати work block усім roadmap;
- не позначати capability реалізованою через scaffolding, mock або декоративний UI;
- після завершення блока наступний dependency block стає основним, але product version лишається Version 1.

## 0.4. Незмінний foundation Version 1

Історичний scope `1.0.0` — **мінімальний реальний foundation Version 1, який працює end-to-end на Android та iOS**. Він не є повним Done об'єднаної Version 1, але не може бути послаблений або видалений.

На foundation-рівні користувач повинен мати можливість:

```text
зареєструватися / увійти
  ↓
мати базовий профіль
  ↓
створити повноцінний PUBLIC + APPROVAL Slot
  ↓
бачити назву / опис / місце / час / організатора / capacity / state
  ↓
як host — редагувати дозволені поля або скасувати свій Slot
  ↓
інший користувач бачить Slot у Pulse
  ↓
подає REQUEST
  ↓
host бачить заявника та APPROVE / REJECT
  ↓
після APPROVE користувач стає accepted participant
  ↓
accepted participants + host отримують temporary text chat
  ↓
host START → ACTIVE → COMPLETE або CANCEL
  ↓
terminal Slot закриває доступ і physically purges ephemeral chat rows
  ↓
користуватися базовими block/privacy/security controls
```

Basic Approval та basic ephemeral chat є частиною **foundation milestone** `1.0.0`. Map, push, Waitlist, realtime chat delivery, durable offline chat, BUMP, City BPM, Fly, AR, Plus та всі інші capability з цього README можуть не блокувати завершення саме foundation milestone, але **обов'язково блокують фінальний LinkUp Version `1.0.0` Done**, доки не реалізовані відповідно до свого контракту.

## 0.5. Roadmap rebase після розширення `1.0.0`

Щоб нічого не видалити після перенесення частини social loop у `1.0.0`:

- basic Approval/REQUEST/APPROVE/REJECT тепер стартує в `1.0.0`;
- `1.1.0` **не видаляється** і стає Approval/Waitlist/Host Control V2: Waitlist, promotion, richer roster/host control, configurable access expansion, race hardening;
- basic accepted-only text chat тепер стартує в `1.0.0`;
- `1.2.0` **не видаляється** і стає Zero-Trace Coordination V2: realtime messages, offline/idempotent retry, system messages, stronger revocation/retention tooling;
- existing Instant access engine **не видаляється**; він лишається canonical domain mode і повертається у richer access-mode surfaces пізніших capability-блоків **усередині того самого `1.0.0`**;
- усі вимоги `1.3.0` → `2.12.0` зберігаються;
- уже написані realtime, city/user channels, outbox, locality, Waitlist та інші пізніші foundations не видаляються через те, що UI активної версії їх ще не повністю exposes.

---

# 1. PRODUCT NORTH STAR

LinkUp — real-world social operating system.

Його задача — максимально швидко перетворювати намір людини на безпечну реальну взаємодію з іншими людьми.

Повна продуктова петля, обов'язкова для фінального `1.0.0` scope:

```text
намір
  ↓
City Context
  ↓
Pulse / Map / Fly discovery
  ↓
Slot
  ↓
JOIN / REQUEST / WAITLIST
  ↓
тимчасова координація
  ↓
реальна зустріч
  ↓
Trust / Reliability / verified real-world signals
```

## 1.1. Незмінні принципи

1. **Real World First** — продукт веде до реальної дії, а не нескінченного скролу.
2. **Local Context First** — місто, район, час і рух є основним контекстом там, де версія це використовує.
3. **Zero-Friction** — ключові дії мають займати мінімум кроків.
4. **Privacy by Architecture** — немає public exact stranger GPS або continuous public tracks.
5. **Server Authority** — capacity, lifecycle, block, reliability, BUMP, access і критичний social state вирішує backend.

---

# 2. CANONICAL ARCHITECTURE

## Mobile

- Kotlin Multiplatform для shared domain/data logic;
- Android: Kotlin + Jetpack Compose;
- iOS: Swift + SwiftUI;
- Coroutines/Flow;
- native Android/iOS lifecycle, location, notifications, BLE, camera/audio та security APIs відповідно до target version;
- Android Keystore;
- iOS Keychain/Secure Enclave-compatible storage where appropriate.

## Backend

- provider-neutral Go API;
- PostgreSQL + PostGIS як source of truth;
- transactional outbox;
- idempotent mutations;
- realtime snapshot + ordered delta model там, де realtime вже активований;
- object/media provider behind replaceable abstraction when media arrives;
- push providers behind platform adapters when notifications arrive.

**Render.com не використовується. Для `1.0.0` Supabase використовується як managed PostgreSQL/PostGIS infrastructure; Go API залишається єдиним app-facing auth/domain/data authority. Supabase Auth/Data API/Realtime/Storage не є authority або mobile dependency, якщо це окремо не активовано в майбутній версії.**

---

# 3. CORE DOMAIN CONTRACT

Canonical social aggregate — **Slot**.

Pulse, Map, LINK і Fly не створюють несумісні дублікати event model.

## Full Slot lifecycle target

```text
DRAFT
  ↓
PUBLISHED
  ↓
FILLING
  ↓
FULL
  ↓
ACTIVE
  ↓
COMPLETED
```

Terminal alternatives:

- CANCELLED;
- EXPIRED;
- MODERATED.

## Core invariants

- `accepted_count <= capacity`;
- last seat allocation atomic;
- duplicate JOIN/REQUEST заборонений;
- mutation idempotency обов'язкова;
- blocked pair не може взаємодіяти через query/ranking/realtime/chat;
- cancelled/expired/moderated/completed Slot не приймає нові JOIN/REQUEST;
- FULL може повернутися у FILLING після LEAVE;
- client ніколи не є authority для Slot version/capacity;
- pending requester не має chat access;
- basic `1.0.0` chat доступний тільки host або accepted participant;
- LEAVE/revocation прибирає participant chat authorization;
- terminal Slot (`COMPLETED`, `CANCELLED`, `EXPIRED`, `MODERATED`) закриває chat і physically purges його ephemeral message rows у canonical PostgreSQL storage.

## Full access target

- Instant;
- Approval;
- Waitlist.

`1.0.0` release surface використовує **Approval** як базовий social path. Instant engine та Waitlist requirements не видаляються.

## Full visibility target

- Public;
- Friends/Links;
- Selected people;
- City-only;
- Lasso/geo scoped;
- Travel corridor;
- Private/Invite-only там, де це передбачено відповідним capability-блоком `1.0.0`.

Visibility ніколи не обходить block/privacy/safety.

Не всі ці режими мають бути ввімкнені вже на foundation milestone, але всі режими, які README вимагає для release surface, мають бути активовані за dependency-order нижче **до фінального `1.0.0` Done**.

---

# 4. VERSION 1.0.0 — SOCIALLY COMPLETE WORKING LINKUP

**Мета:** найменший production-safe LinkUp, який уже можна реально встановити й використовувати двом людям не як demo, а як базову соціальну мережу для реальної зустрічі.

## 4.1. Account

Обов'язково:

- registration;
- login/logout;
- secure bearer session;
- session restore після process death;
- базовий recovery/reset flow;
- username/display name;
- базовий avatar/profile identity;
- Android + iOS parity.

Розширене керування sessions/export/account lifecycle може розвиватися далі, але security основи не відкладаються.

## 4.2. Basic Me

- profile view/edit;
- basic account info;
- базова privacy visibility PUBLIC/HIDDEN;
- базовий block list/control;
- language baseline;
- logout;
- legal/version surface;
- зрозумілий session/account state.

## 4.3. Basic Event Core + Slot Engine

У `1.0.0` активний простий, але **не урізаний** social path:

- create PUBLIC Slot;
- release access mode: `APPROVAL`;
- existing `INSTANT` access engine не видаляється з domain/data/tests;
- create/publish як один простий user flow, навіть якщо backend internally зберігає DRAFT/PUBLISHED;
- Event Core: title/activity, optional description/details, place/zone, optional start date/time, organizer identity, capacity, state;
- get/read;
- host edit дозволених полів: title/details/place/start time/capacity;
- optimistic concurrency через server-authoritative Slot version;
- capacity при edit не може бути меншою за вже accepted count;
- host cancel через canonical `CANCEL`; UI може називати це «Видалити», але сам Slot не hard-delete-иться: переходить у `CANCELLED` та зникає з normal Pulse discovery;
- JOIN для `APPROVAL` створює pending REQUEST, а не instant membership;
- pending user може скасувати свою заявку через canonical LEAVE semantics;
- host бачить pending requester identity (`displayName`, `@username`);
- host APPROVE / REJECT;
- APPROVE переводить requester у accepted membership тільки server-side;
- REJECT прибирає pending request без membership;
- accepted participant може LEAVE;
- host START переводить eligible Slot у `ACTIVE`;
- host COMPLETE переводить `ACTIVE` Slot у `COMPLETED`;
- capacity;
- atomic seat allocation при approval;
- duplicate JOIN/REQUEST protection;
- idempotent create/request/approve/reject/leave/start/complete/cancel;
- host/member authorization;
- basic block enforcement;
- server-authoritative version/state;
- PostgreSQL transaction safety.

Waitlist, advanced visibility, explicit DRAFT/preview/publish tooling, co-hosting, expanded roster management та інше **advanced host tooling** переходять у наступні versions. Базове edit/cancel, Approval та start/complete lifecycle власного Slot є частиною `1.0.0`.

## 4.4. Basic Pulse

- real server data only;
- простий список доступних PUBLIC Slots;
- Slot card;
- title/activity;
- description/details where available;
- basic place/zone text where available;
- date/time where available;
- organizer identity;
- capacity `x/y`;
- server-authoritative state;
- host Edit / Delete(cancel) actions для власного Slot;
- host бачить pending request list із requester identity;
- host Accept / Decline actions;
- non-member бачить `Подати заявку` / `Request to join`;
- pending viewer бачить `Заявку надіслано` і може cancel request;
- accepted participant бачить Chat + LEAVE;
- host бачить Chat після появи accepted participant;
- host може START, а після `ACTIVE` — COMPLETE;
- active Slot лишається доступним host/accepted participants, але не відкривається стороннім як новий request target;
- loading/content/empty/error;
- manual/normal refresh достатній для `1.0.0`;
- ніяких fake users/Slots/online numbers.

Realtime без manual refresh буде окремим update.

## 4.5. Basic Zero-Trace Coordination Chat

Для host та **accepted** Slot participants:

- ephemeral text chat;
- pending requester не має доступу;
- stranger/non-member не має доступу;
- author display name;
- author username;
- server-created message timestamp;
- send text;
- normal/manual refresh достатній для `1.0.0`;
- bounded message length;
- bounded recent thread response;
- server authorization на read/send;
- LEAVE прибирає member access;
- terminal Slot закриває chat;
- PostgreSQL trigger physically deletes chat rows when Slot becomes `COMPLETED`, `CANCELLED`, `EXPIRED` або `MODERATED`;
- chat не є permanent social-message archive.

Realtime delivery, durable offline retry, system messages, richer coordination і stronger revocation tooling розвиваються у `1.2.0`, але basic working chat уже обов'язковий у `1.0.0`.

## 4.6. Basic safety/privacy/security

До release обов'язково:

- server authorization;
- basic block enforcement;
- no public exact stranger GPS;
- no public continuous tracks;
- secrets/config separation;
- secure local session storage;
- rate-limit foundation;
- sensitive logs prevention;
- production data only;
- chat access не можна отримати лише через знання `slotId`;
- pending/declined/left user не є accepted chat participant.

## 4.7. Release engineering

`1.0.0` не випускається без:

- Android release build/AAB;
- iOS release/archive readiness;
- signing configuration;
- forward-only migrations;
- basic Go/KMP/Android/iOS tests для active scope;
- basic accessibility;
- Ukrainian + English baseline;
- release smoke на Android та iOS;
- real two-user Approval + chat smoke;
- rollback/recovery path;
- CI, коли GitHub runner реально доступний.

## 4.8. Definition of Done — 1.0.0

1. User A може зареєструватися/увійти.
2. User A створює PUBLIC Approval Slot з реальною інформацією: title/details/place/time/capacity.
3. User A може відредагувати дозволені поля власного Slot, а server version не дозволяє тихо перетерти новішу зміну.
4. User B бачить реальний Slot у Pulse з достатньою інформацією для рішення про REQUEST.
5. User B натискає `Подати заявку`; server створює pending request, а не accepted membership.
6. User A бачить User B у pending list із display name/@username та може APPROVE або REJECT.
7. Після APPROVE User B стає accepted participant, capacity/state залишаються server-authoritative.
8. До APPROVE User B не може читати/писати chat; після APPROVE A і B можуть обмінятися text messages.
9. User B може LEAVE; після LEAVE chat access для B зникає.
10. User A може START подію; під час ACTIVE host/accepted participants зберігають coordination access.
11. User A може COMPLETE подію; після terminal transition chat закривається, а його message rows physically purged from PostgreSQL.
12. User A може «видалити» власний Slot через server `CANCEL`; Slot зникає з normal Pulse, а chat також purged.
13. Concurrent approval/last-seat allocation не перевищує capacity; edit capacity не може впасти нижче accepted count.
14. Duplicate request/command replay не створює duplicate membership або duplicate critical mutation.
15. Block/security/privacy foundation працює server-side, включно з social/chat authorization boundary.
16. Немає fake/demo production data.
17. Android та iOS проходять однаковий release smoke для цього flow.

## 4.9. Реально доданий implementation baseline для `1.0.0`

Цей roadmap відображає вже додану implementation direction, яку заборонено мовчки відкотити:

- full Event Core fields у mobile/shared/server flow;
- host edit endpoint + optimistic version check;
- cancel/delete semantics;
- Approval request/approve/reject domain path;
- pending requester summaries для host;
- temporary chat API read/send;
- accepted-only chat authorization;
- automatic first-accepted-participant chat activation plus explicit activation/expiry metadata;
- idempotent offline-safe send identity, reconnect polling and duplicate convergence;
- block/kick relationship revocation and blocked-identity thread filtering/purge;
- finite mode-aware Slot/chat TTL, automatic `EXPIRED` transition and physical retention cleanup;
- server-authored Slot lifecycle system messages;
- START/COMPLETE lifecycle;
- forward-only migration `000012` для Slot details;
- forward-only migration `000013` для ephemeral Slot chat + terminal purge trigger;
- forward-only migrations `000016`, `000018` та `000020` для safe resend, system messages і finite expiry/retention;
- Android `1.0.0` surface для Event Core / Approval / host controls / temporary chat;
- Android BUMP surface з P-256 Android Keystore proof, fresh precise foreground GPS, server-only verification, anti-replay/anti-farm та self-only Reliability Vault;
- forward-only migration `000022` для BUMP/reliability; exact challenge geography очищається після використання, а guarded PUBLIC Pulse index перенесено без зміни SQL у reserved tail `999999`;
- iOS має отримати еквівалентний native SwiftUI surface до Done `1.0.0`.

---

# 5. VERSION 1.0.1 — LAUNCH STABILIZATION

- crash fixes;
- ANR/hang fixes;
- network resilience;
- basic offline/retry UX;
- query/index tuning;
- performance profiling;
- accessibility fixes;
- localization fixes;
- abuse/rate-limit tuning;
- UX friction cleanup;
- basic observability dashboards;
- API latency/error metrics;
- backup/recovery procedure;
- release rollback procedure hardening;
- Approval request/approve/reject error-state polish;
- temporary chat transport/error-state polish;
- lifecycle START/COMPLETE/CANCEL recovery UX;
- two-user social-loop regression stabilization.

Done: basic create → discover → REQUEST → APPROVE/REJECT → accepted chat → START/COMPLETE/CANCEL flow стабільний у production conditions.

---

# 6. VERSION 1.0.2 — REALTIME + DURABLE OFFLINE

- Slot realtime events;
- city activity channel;
- user channel;
- authoritative snapshot;
- ordered delta;
- monotonic cursor/version handling;
- duplicate/out-of-order protection;
- reconnect;
- foreground/background lifecycle;
- process-death recovery;
- durable mutation outbox;
- exact idempotent replay;
- airplane-mode recovery;
- current block/access re-evaluation while stream is open;
- two-client REQUEST/APPROVE/REJECT/JOIN/LEAVE/FULL/REOPEN/START/COMPLETE convergence;
- realtime pending-request visibility for host;
- realtime membership/access revocation;
- basic chat realtime delivery baseline on top of the `1.0.0` persistent ephemeral thread;
- PostgreSQL transactional outbox;
- controlled realtime fan-out / LISTEN-NOTIFY wake where appropriate.

Done: два клієнти бачать один server-authoritative Slot/social state без divergence після reconnect.

---

# 7. VERSION 1.0.3 — CITY CONTEXT + SCHEDULED FOUNDATION

- physical locality resolution;
- locality polygons;
- PostGIS City Context;
- freshness/accuracy policy;
- approximate/precise classes;
- City-Lock;
- boundary stability/hysteresis;
- stale/fake/low-accuracy rejection rules;
- privacy-safe city context;
- Android + iOS synchronization;
- NOW / Scheduled separation;
- timezone-aware scheduled time;
- basic expiry rules;
- city-only visibility foundation.

Немає public exact stranger coordinates.

---

# 8. VERSION 1.0.4 — MAP V1

- Google Maps native SDK;
- Google Places canonical place identities;
- real server viewport queries;
- Slot pins/aggregates, а не public people pins;
- privacy-safe map exposure;
- shared city/time scope з Pulse;
- open Slot from map;
- JOIN/REQUEST from map відповідно до access mode;
- clustering/viewport performance;
- no raw full-city user download.

Advanced Vibe Topology, Lasso, Hotspots і AR лишаються далі.

---

# 9. VERSION 1.0.5 — LINK CREATION + HOSTING V1 EXPANSION

- explicit DRAFT;
- preview;
- separate publish;
- expanded edit surface/history beyond the `1.0.0` basic fields;
- Now / Scheduled explicit mode separation;
- activity/scenario selection;
- visibility baseline expansion;
- canonical location/place identity;
- structured vibe tags baseline;
- create request survives network loss/process death without duplicate Slot;
- basic Hosting/Joined dashboard;
- configurable basic access choice foundation while preserving canonical Instant + Approval domain modes;
- clearer roster/request/accepted summaries without yet taking over the advanced `1.1.0` host-control scope.

---

# 10. VERSION 1.1.0 — APPROVAL + WAITLIST + HOST CONTROL V2

Basic Approval already exists in `1.0.0`; this version **expands it without deleting the original requirements**:

- Approval access mode expansion/configuration;
- REQUEST;
- APPROVE;
- REJECT;
- Waitlist;
- waitlist promotion;
- FULL → FILLING reopen;
- expiry during mutation;
- host remove where authorized;
- richer hosting/request/waitlist states;
- roster management foundation;
- concurrency-safe Approval/Waitlist transactions;
- block/authorization checks on every transition;
- version conflict handling;
- request expiry/withdrawal UX hardening;
- advanced host control around capacity, roster and access-mode transitions.

---

# 11. VERSION 1.2.0 — ZERO-TRACE COORDINATION V2

Basic accepted-only text chat already exists in `1.0.0`; this version **extends it without converting LinkUp into a permanent messenger**.

Для accepted Slot participants:

- ephemeral text chat;
- membership authorization;
- realtime messages;
- offline retry/idempotency;
- block/kick revokes access;
- activation rules;
- expiry/retention;
- system messages for important Slot state;
- no permanent social-message archive as default product behavior;
- reconnect/thread convergence;
- message duplicate protection;
- stronger terminal cleanup verification;
- chat state transitions tied to host/member lifecycle.

---

# 12. VERSION 1.3.0 — NOTIFICATIONS

- platform push integration;
- JOIN/REQUEST approval;
- Slot cancellation;
- starting soon;
- reopened seat / waitlist promotion;
- safety/moderation messages;
- chat/coordination notification baseline where policy allows;
- dedupe key;
- TTL;
- deep links;
- user preferences;
- quiet-hours foundation;
- notification reliability metrics.

---

# 13. VERSION 1.4.0 — VERIFIED REAL-WORLD + RELIABILITY

- BUMP baseline;
- verified attendance;
- server-verified proof;
- anti-replay/anti-farm;
- reliability events;
- private/public reliability bands;
- BUMP Vault baseline;
- after-check flow;
- verified real-world completion metrics;
- one event/one contribution protections.

Guardrail: local client tap не може сам нарахувати trust/reliability.

---

# 14. VERSION 1.5.0 — CITY ENERGY

- City BPM;
- activity buckets;
- normalized city baseline;
- time decay;
- minimum cohort;
- dominant categories/tags;
- privacy suppression;
- realtime city telemetry;
- Kinetic Proof generation/presentation;
- spatial/time blur;
- anti-manipulation weighting.

Done: city activity explainable, stable і не дозволяє інферити малу групу/окрему людину.

---

# 15. VERSION 1.6.0 — MAP INTELLIGENCE

- privacy hex grid;
- Vibe Topology;
- Echo Hotspots;
- Lasso;
- shared GeoTemporalScope Pulse ↔ Map;
- Time-Lapse baseline;
- Dark Zone Ignition;
- advanced map privacy thresholds;
- accessibility labels/patterns, не тільки колір.

---

# 16. VERSION 1.7.0 — AUTO-SWARMS

- normalized intent taxonomy;
- clustering;
- confidence score;
- proposal lifecycle;
- consent;
- neutral meeting zone;
- realtime proposal UI;
- false-positive analytics;
- privacy/safety/block filters before candidate grouping.

Guardrail: Auto-Swarm ніколи не auto-joins users.

---

# 17. VERSION 1.8.0 — FLY NOW

- MICRO Slots;
- Flash Drops;
- short TTL;
- proximity feed;
- Speed-Swipe semantics;
- Instant Duo;
- Micro-Sparks;
- Action Stream;
- Hot-Swap Interceptor;
- privacy-safe motion buckets;
- server-time countdown;
- no public live GPS track.

---

# 18. VERSION 1.9.0 — FLY TRAVEL

- Astral remote-city context;
- remote scheduled joins;
- Road-Trip Slots;
- Just Landed;
- transit contexts;
- Event Squads;
- Local Ambassador;
- Highway Flare / non-emergency road help;
- Nomad/Home Cities;
- Macro city activity map.

Guardrail: ASTRAL context ніколи не маскується під physical presence.

---

# 19. VERSION 1.10.0 — FLY MOTION

- activity recognition;
- motion confidence;
- driver/passenger distinction;
- low-interaction HUD;
- trajectory Slot discovery;
- safe-state interaction;
- anonymous Fly encounter tokens;
- Slipstream aggregates;
- battery profiling.

Guardrail: probable driver не отримує UX, який заохочує tap/swipe/typing while driving.

---

# 20. VERSION 1.11.0 — ME 2.0 + SQUAD RADAR

- Social Passport;
- Links Graph;
- expanded Reliability Card;
- Now Card;
- Activity Cockpit;
- Real-World Footprint;
- BUMP Vault expansion;
- achievements;
- personal templates;
- home cities;
- interests/current intentions;
- Privacy Center;
- Safety Center;
- Security Center;
- notification intelligence;
- Data Transparency;
- private Squad Radar;
- OFF / ETA_ONLY / APPROXIMATE / LIVE_PRECISE;
- exact sharing opt-in + TTL + instant revoke;
- blocked/kicked membership revocation.

---

# 21. VERSION 1.12.0 — AR + RANKING V2

**Остання версія лінійки `1.x`.**

- AR Slot/Place beacons;
- capability detection;
- camera/IMU;
- coarse slot anchors;
- accessibility/fallback;
- learned ranking;
- preference modeling;
- notification intelligence V2;
- swarm tuning;
- explainability/debug tooling;
- opt-in personalization.

ML ніколи не замінює deterministic eligibility/privacy filters.

---

# 22. VERSION 2.0.0 — COLD START + VENUE ECOSYSTEM

- Ignite City;
- launch pool anti-sybil;
- activation fan-out;
- verified venue identity;
- Venue Perks;
- perk qualification/redemption;
- Venue Vibe feedback;
- Venue Vibe snapshots;
- accessibility venue attributes;
- venue fraud protection;
- canonical Google Place ID + internal venue identity.

Guardrails:

- немає fake users/Slots/online count;
- Venue Perks не купують organic Hotspot/City BPM/ranking;
- venue feedback не розкриває individual participant identity.

---

# 23. VERSION 2.1.0 — OFFLINE REAL-WORLD OPS

- device public keys;
- BLE Outbox Mesh;
- encrypted offline BUMP envelopes;
- server reconciliation;
- Swarm BUMP;
- group proximity challenge;
- Second Wind recruitment;
- ACTIVE Slot hot seats;
- process-death/background sync;
- anti-replay/anti-farm;
- cross-platform real-device BLE tests.

Offline proof створює лише pending proof; trust виникає після server verification.

---

# 24. VERSION 2.2.0 — SAFETY + ACCESSIBILITY EXPANSION

- Ghost Guardian;
- temporary opaque guardian links;
- STATUS_ONLY;
- ETA_APPROXIMATE;
- SAFETY_RADAR;
- explicit LIVE_PRECISE;
- burned/indistinguishable 404 after expiry/revoke/completion;
- Barrier-Free Mode;
- strict/highlight accessibility filtering;
- privacy-safe accessibility preferences;
- stronger home/private scenario policies;
- Guardian abuse/rate-limit protections.

---

# 25. VERSION 2.3.0 — EPHEMERAL MEDIA & COMMUNICATION

- Slot Cam;
- transient encrypted media relay;
- Memory Drop;
- local final gallery/collage;
- server cleanup verification;
- Auto-Translate Hub;
- original text remains canonical;
- Sonic Vibes;
- max short audio intent;
- Bill Splitter calculator;
- deterministic minor-unit rounding;
- chat/media offline/reconnect integration.

Guardrails:

- no permanent cloud album;
- no permanent translation-only message copy;
- no background microphone;
- Bill Splitter не є payment processor.

---

# 26. VERSION 2.4.0 — ADAPTIVE EXPERIENCE

- Host Perks;
- Zen Mode;
- Surprise Me / Magic Button;
- Weather-Triggered Swarms;
- Asset Match;
- adaptive ranking integrations;
- anti-spam/anti-farm;
- expanded preference/privacy controls.

Guardrails:

- Host Perks не купуються;
- Zen не карає user і не збирає invasive raw device history;
- Surprise Me не обходить Approval/physical presence;
- Weather Swarms не auto-join;
- Asset Match не стає marketplace небезпечних/регульованих предметів.

---

# 27. VERSION 2.5.0 — LINKUP+ BILLING FOUNDATION

- MONTHLY / ANNUAL products;
- canonical base pricing policy;
- Google Play purchase/restore;
- App Store purchase/restore;
- server purchase validation adapters;
- store notifications/webhooks;
- entitlement state machine;
- ACTIVE / GRACE / BILLING_RETRY / EXPIRED / REVOKED;
- account/device switching;
- subscription management in Me;
- refund/revoke/grace handling;
- billing observability.

Guardrail: client `isPlus=true` ніколи не є authority.

Free core safety/privacy/create/join ніколи не закриваються за Plus.

---

# 28. VERSION 2.6.0 — LINKUP+ TRAVEL PRO

- Global Astral Jump;
- supported-locality remote exploration;
- physical vs astral separation;
- Pre-Flight Slots;
- extended scheduled/travel horizon;
- timezone/DST-safe planning;
- Voice Babel;
- transient speech-to-text;
- translated transcript presentation;
- no permanent voice archive.

---

# 29. VERSION 2.7.0 — LINKUP+ HOST POWER TOOLS

- Mega-Slots up to product ceiling;
- scalable membership/pagination/realtime;
- Stealth Slots / invite-link-only visibility;
- opaque revocable invite tokens;
- advanced Slot Blueprints;
- no old participant/token/private-location cloning;
- Co-Hosting;
- granular organizer permissions;
- owner protection;
- audited grant/revoke.

---

# 30. VERSION 2.8.0 — LINKUP+ ADVANCED DISCOVERY

- Radius Overdrive;
- bounded 20–30 km class discovery where policy allows;
- viewport/cursor/aggregate queries;
- Vibe-Match Filters;
- languages/interests/assets/accessibility-compatible filters;
- Auto-Pilot Join rules;
- explicit automation scope;
- Approval → REQUEST;
- Instant auto-JOIN only with explicit opt-in;
- daily caps/cooldown/dedupe;
- Pulse Time-Machine;
- historical privacy-safe aggregate city/map data;
- no individual route reconstruction.

---

# 31. VERSION 2.9.0 — LINKUP+ PRIVACY & QUALITY OF LIFE

- Ghost Mode;
- no passive public presence/BPM contribution;
- Priority Boarding visual/request aid with no seat/waitlist advantage;
- Oops-Rewind via compensating commands;
- Multi-Threading / intent groups;
- atomic winner lock;
- no silent cancellation after external commitment;
- Guardian Auto-Ping;
- trusted guardian contacts;
- no automatic LIVE_PRECISE without explicit consent.

---

# 32. VERSION 2.10.0 — LINKUP+ IDENTITY, ANALYTICS & THEMES

- Hex-Aura cosmetic treatment;
- no ranking/BPM/Hotspot benefit;
- BUMP Vault Pro self-only analytics;
- unique people met;
- repeat BUMPs;
- verified meetup count;
- activity/city trends;
- no exact route reconstruction;
- Custom App Icons;
- themes including OLED Black / Neon Cyberpunk class packs;
- contrast/font scaling/Reduce Motion accessibility QA.

---

# 33. VERSION 2.11.0 — LINKUP+ FORGIVENESS + BILLING HARDENING

- No-Strike Forgiveness;
- maximum one non-stacking eligible credit per policy period;
- real cancellation event remains in audit;
- no no-show/safety/moderation forgiveness;
- bounded reliability modifier;
- billing sandbox matrix Android+iOS;
- webhook replay/idempotency;
- refund/chargeback/revoke;
- grace/billing retry;
- restore after reinstall/device change;
- account switch;
- entitlement cache expiry/offline behavior;
- Mega-Slot load tests;
- Stealth penetration tests;
- Auto-Pilot abuse/rate limits;
- Ghost privacy audit;
- Guardian delivery/privacy audit;
- historical Map inference/privacy tests;
- subscription localization/cancellation UX;
- feature flag rollback;
- production smoke.

---

# 34. VERSION 2.12.0 — LINKUP+ FREE DAY + ECOSYSTEM FINAL HARDENING

**Це максимальна версія всього поточного roadmap.**

## LinkUp+ Free Day / rewarded access

- opt-in rewarded-ad quest only inside LinkUp+ surface;
- five verified rewarded views;
- minimum 4h between verified steps;
- 24h quest window;
- fifth view → atomic 24h Plus access grant;
- 7-day cooldown after reward expiry;
- provider server-side verification;
- one-time challenge model;
- duplicate callback/replay protection;
- multi-device synchronization;
- local reminders;
- privacy/consent/age/region gating;
- provider outage/no-fill fallback;
- kill switches;
- server-authoritative timestamps;
- no client `watched=true` authority;
- paid subscription precedence without grant stacking.

## Final ecosystem hardening

- cross-platform BLE real-device matrix;
- Keychain/Keystore key-loss/reinstall flows;
- Guardian penetration/abuse tests;
- media deletion audit;
- translation privacy audit;
- venue fraud/redemption abuse tests;
- Ignite sybil/load tests;
- weather provider outage fallback;
- accessibility QA;
- localization;
- push storm protection;
- observability dashboards;
- feature flag rollback drills;
- Android/iOS production smoke;
- store/release validation.

Guardrails for rewarded access:

- no ads in Pulse/Map/LINK/Fly/chat/safety flows;
- no forced interstitials;
- no ad-click/install/purchase requirement;
- no precise social/location data sent for ad targeting;
- probable driver state never launches rewarded video;
- already validly earned grant is not silently removed by feature kill-switch.

---

# 35. CROSS-VERSION NON-NEGOTIABLES

Ці правила діють у **кожній** версії.

## Privacy

- no public exact stranger GPS;
- no public continuous tracks;
- block applied server-side at every layer that exists in the active version;
- exact location sharing тільки opt-in і temporary;
- raw location retention minimized;
- low-cohort aggregates suppressed;
- private/home Slots не повинні інферитися через heatmaps/history.

## Security

- server authorization on every sensitive operation;
- no trusted client role/entitlement/capacity flags;
- secrets never shipped in mobile app;
- session/device lifecycle controlled;
- rate limits;
- abuse protection;
- idempotency;
- audit/domain events for critical mutations.

## Realtime

Коли realtime feature вже входить у active version, canonical rule:

```text
server snapshot
+ ordered realtime deltas
+ local optimistic/pending command state
```

Після reconnect:

1. resubscribe;
2. authoritative snapshot/cursor;
3. replay exact pending idempotent commands;
4. reconcile UI;
5. ignore stale/duplicate deltas.

`1.0.0` може працювати через normal refresh і не зобов'язаний чекати повного realtime stack; це стосується і basic temporary chat.

## UX state

Кожна production screen/feature має релевантні explicit states:

- Loading;
- Content;
- Empty;
- OfflineCached / OfflinePending where relevant;
- Refreshing where relevant;
- ErrorRecoverable;
- ErrorBlocking.

## Android + iOS

Фіча не вважається завершеною у своїй target version, якщо заявлена для обох платформ, але реально працює тільки на одній.

---

# 36. TEST CONTRACT

Тести додаються до release gate тоді, коли відповідна feature стає active production scope. Уже написані майбутні тести не видаляються.

## Basic `1.0.0`

- registration/login/session restore;
- create idempotency;
- Slot details serialization/readback;
- host edit authorization;
- edit version conflict;
- edit capacity cannot fall below accepted count;
- host CANCEL removes Slot from normal discovery without hard-delete;
- Approval request creation;
- pending requester cannot become accepted without host approval;
- host-only APPROVE/REJECT authorization;
- approval last-seat/capacity race;
- duplicate request protection;
- request cancellation through LEAVE semantics;
- accepted member LEAVE;
- host START/COMPLETE authorization and state transitions;
- pending/stranger chat rejection;
- accepted/member chat read/send;
- chat access revoked after LEAVE;
- terminal chat purge on COMPLETE/CANCEL/EXPIRED/MODERATED;
- chat table/function authorization boundary;
- blocked actor;
- Pulse real-data loading/empty/error/request/host-state surfaces;
- Android release smoke;
- iOS release smoke;
- real two-account Approval + chat end-to-end smoke.

## Launch stabilization `1.0.1`

- GET/read network failures retry only within bounded timeout/retry policy;
- POST/mutation requests are never blindly auto-retried by transport policy;
- timeout/network/5xx failures map to recoverable user-facing errors rather than raw transport codes;
- successful cached/read content is not discarded just because a refresh fails;
- request ID, HTTP status and latency are emitted by API observability without logging query/body/bearer secrets;
- rate-limit bucket storage is bounded/cleaned so long-running API memory usage cannot grow forever from stale entries;
- performance advisor has no actionable missing-FK index for active `1.0.1` tables;
- two-user PUBLIC + APPROVAL social loop regression covers REQUEST → APPROVE/REJECT → chat → START/COMPLETE/CANCEL and terminal chat purge;
- Android touch targets/backup rules and active accessibility fixes pass build/lint checks;
- Ukrainian + English active surfaces keep equivalent error/recovery meaning;
- Android debug/release compile and iOS simulator/device-archive preflight use `1.0.1` version metadata;
- backup/recovery and application rollback procedures are documented and exercised non-destructively before production release;
- real Android/iOS production-condition smoke includes temporary network loss and recovery without duplicate critical mutation.

## Slot expansion

- JOIN + cancel race;
- approval race;
- waitlist promotion race;
- expiry during mutation;
- duplicate idempotency key;
- lost response replay.

## Realtime/offline

- stream drop;
- duplicate event;
- out-of-order event;
- background/foreground;
- process kill;
- airplane mode;
- reconnect to FULL;
- reopen while offline;
- two-client convergence;
- request/approval convergence;
- chat realtime reconnect/duplicate convergence once `1.0.2`/`1.2.0` scope is active.

## Geo/privacy

- city boundary jitter;
- low accuracy;
- stale sample;
- approximate permission;
- permission denied;
- rural/transit;
- blocked user same geo scope;
- low cohort;
- precise share expiry.

## Safety/security

- unauthorized actor;
- block mid-session;
- deleted account/session;
- rate-limit abuse;
- token guessing for opaque links;
- replay attacks;
- sensitive data leakage tests.

## Version 1 advanced-capability matrices

Усі матриці нижче обов'язкові для LinkUp Version 1; їх виконують у відповідному dependency-safe capability block:

- Chat V2 activation/expiry/offline/duplicate/kick/block/realtime/system-message scenarios;
- BUMP replay/anti-farm/offline/device-key scenarios;
- Swarm BUMP quorum/concurrency/replay;
- Second Wind capacity/realtime/expiry;
- Guardian token/expiry/revoke/cache/permission tests;
- Barrier-Free strict/highlight/privacy consistency;
- Slot Cam/Memory Drop permission/encryption/cleanup/process-death;
- Auto-Translate provider/reconnect/retention/original-text;
- Sonic Vibe duration/media/permission/driver/offline;
- Bill Splitter rounding/authorization/version races;
- Host Perks farm/moderation/ranking cap;
- Zen no-penalty/privacy/offline;
- Surprise Me eligibility/idempotency/explainability;
- Weather stale/duplicate/outage/quiet-hours/consent;
- Asset Match TTL/privacy/unsafe-asset rejection;
- Plus billing replay/renewal/expiry/grace/refund/restore/account switch;
- Mega-Slot 50-person class load and 51st rejection;
- Stealth token guessing/revoke/rotation/no-discovery-leakage;
- Co-Host permission escalation/revoke/audit;
- Auto-Pilot duplicate/cap/quiet-hours/block/disable race;
- Ghost presence/privacy audit;
- Oops-Rewind expiry/irreversible/version conflict;
- Multi-Threading simultaneous winners/external commitment/process death;
- Free Day state machine, 4-hour boundary, provider verification, multi-device race, fifth-view atomic grant, reward expiry and cooldown.

---

# 37. RELEASE DEFINITION OF DONE

Для єдиної LinkUp Version 1:

- увесь scope цього README реалізований end-to-end;
- migrations forward-only;
- Android/iOS parity для заявлених flows;
- privacy/security reviewed;
- offline/reconnect behavior визначений для кожного Version 1 capability block;
- відповідні tests реально виконані;
- CI green, коли runner доступний;
- no fake/mock production flow;
- accessibility/localization оновлені для всіх Version 1 surfaces;
- observability достатня для всього Version 1 release scope;
- release smoke completed;
- rollback path exists.

Capability block можна перевірити й зафіксувати окремо, але незавершений блок із цього README завжди блокує загальний статус `VERSION 1 PRODUCTION DONE`.

---

# 38. VERSION 1 CAPABILITY ORDER

```text
LINKUP VERSION 1 (single product release)
  ↓
foundation 1.0.0 + stabilization 1.0.1
  ↓
realtime/offline + city + map + hosting (legacy 1.0.2–1.0.5)
  ↓
approval/waitlist + coordination + notifications (legacy 1.1.0–1.3.0)
  ↓
verified real world + city intelligence + swarms (legacy 1.4.0–1.7.0)
  ↓
Fly + Me/Squad + AR/ranking (legacy 1.8.0–1.12.0)
  ↓
venue/offline/safety/media/adaptive systems (legacy 2.0.0–2.4.0)
  ↓
Plus billing/travel/host/discovery/privacy/identity (legacy 2.5.0–2.10.0)
  ↓
billing hardening + rewarded access + ecosystem hardening (legacy 2.11.0–2.12.0)
  ↓
VERSION 1 PRODUCTION DONE
```

Якщо з'являється нова ідея:

1. вона не додається мовчки до active version;
2. визначається dependency;
3. призначається конкретний dependency/capability block усередині Version 1;
4. README оновлюється без видалення існуючих roadmap requirements;
5. active version не роздувається без необхідності.

Поточне розширення Approval + basic ephemeral chat у `1.0.0` є **явною зміною product requirement**, а не мовчазним scope creep, тому downstream roadmap вище rebased без видалення старих вимог.

---

# 39. CURRENT TARGET

**Active release: LinkUp Version 1 (`1.0.0`) — Unified Product Scope.**

Увесь README є release scope цієї однієї версії. Наявний social baseline не видаляється й не переписується; він є перевіреним foundation, поверх якого послідовно інтегруються всі capability-блоки.

Поточний dependency order:

1. зберегти green account/Approval/chat/lifecycle foundation;
2. завершити stabilization, backup/restore та реальні Android/iOS build/smoke gates;
3. завершити realtime/offline + City Context foundations;
4. активувати Map, hosting/access/waitlist, coordination і notifications;
5. реалізувати verified real-world, city intelligence, swarms, Fly, Me/Squad та AR;
6. реалізувати venue, offline mesh, safety/accessibility, ephemeral media й adaptive systems;
7. реалізувати Plus billing/travel/host/discovery/privacy/identity та rewarded access;
8. виконати повний cross-platform, privacy, abuse, billing, restore й ecosystem hardening;
9. release decision приймати тільки після фактичних test/build/device/production-smoke results для всього Version 1 scope.

Уже написані realtime, city/user channels, outbox, locality, Instant engine, Waitlist та інші foundations **не видаляються**. Жоден майбутній surface не вважається готовим, доки його server/data/native flow не працює end-to-end.
