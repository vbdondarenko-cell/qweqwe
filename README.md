# LinkUp — PRODUCT & ENGINEERING CONTRACT

Статус: **canonical product/engineering contract після `PROJECT_RULES.md`**.

Поточний release train: **v1.0 → v1.1**.

Поточна активна версія для розробки: **LinkUp v1.0**.

`PROJECT_RULES.md` має найвищий пріоритет. `README.md` визначає product scope, architecture, version scope, invariants, test/release contract і Definition of Done. `IMPLEMENTATION_STATUS.md` є єдиним фактичним журналом того, **що реально вже зроблено, перевірено або ще не зроблено**.

> README більше не є implementation ledger. У ньому не повинно бути історичних тверджень на кшталт “already implemented”, старих migration IDs як доказу готовності або псевдо-версій `1.0.1 … 2.12.0`. Реальний стан завжди перевіряється в `IMPLEMENTATION_STATUS.md` та в коді `main`.

Основна навігація продукту: **Pulse · Map · LINK · Fly · Me**.

---

# 0. VERSION MODEL

LinkUp Version 1 складається з двох реальних послідовних product releases:

| Release | Назва | Головна мета |
|---|---|---|
| **v1.0** | Core Social Network | реальна Android соціальна мережа: account → Slot → REQUEST → APPROVE → chat → real meeting lifecycle |
| **v1.1** | Realtime Real-World City Network + LinkUp+ | realtime/offline, City Context, Map, Waitlist, BUMP/Reliability, City BPM, swarms, Fly, Me 2.0, AR/ranking, venue/BLE/safety/media/adaptive systems, advanced discovery/hosting/privacy, billing, rewarded access, ecosystem hardening |

## 0.1. Release boundary rule

- **v1.0 не блокується вимогами v1.1.**
- **v1.1 починається тільки поверх стабільного v1.0 foundation.**
- Функція, що належить v1.1, може мати ранній foundation у коді, але це не переносить її release requirement назад у v1.0.
- Не можна оголошувати feature готовою через scaffold, mock, decorative UI або документацію.
- Якщо capability вже реально реалізована раніше свого release, її не видаляють: вона просто проходить свій повний DoD у відповідному release gate.

## 0.2. Active platform gate

Поки користувач прямо не скаже працювати над iOS:

- active mobile platform: **Android**;
- Android/client implementation: **Kotlin + Jetpack Compose**;
- backend/domain authority: **Go**;
- database: **PostgreSQL/PostGIS**;
- iOS: **frozen** і не є release blocker;
- відсутність iOS parity не знижує Android readiness;
- Kotlin Multiplatform дозволений лише там, де він не створює iOS work block.

---

# 1. PRODUCT NORTH STAR

LinkUp — real-world social operating system.

Мета продукту — максимально швидко й безпечно перетворити намір людини на реальну взаємодію з іншими людьми.

Canonical product loop:

```text
intent
  ↓
local/city context
  ↓
Pulse / Map / Fly discovery
  ↓
canonical Slot
  ↓
JOIN / REQUEST / WAITLIST
  ↓
temporary coordination
  ↓
real-world meeting
  ↓
verified trust / reliability signals
```

## 1.1. Product principles

1. **Real World First** — продукт веде до реальної дії, а не нескінченного скролу.
2. **Local Context First** — місто, район, час і physical context використовуються там, де capability це потребує.
3. **Zero-Friction** — ключові дії мають мінімальну кількість кроків.
4. **Privacy by Architecture** — немає public exact stranger GPS або continuous public tracks.
5. **Server Authority** — capacity, lifecycle, access, block, billing, reliability, BUMP і critical social state вирішує backend.
6. **No Fake Production State** — fake users, fake Slots, fake online counts, fake City BPM, fake reliability або decorative “working” interactions заборонені.

---

# 2. DESIGN CONTRACT

Поточний React/TypeScript дизайн у repository є **frozen canonical visual/UI contract**.

- Не редизайнити.
- Не робити facelift/visual cleanup замість функціоналу.
- Не міняти navigation model `Pulse · Map · LINK · Fly · Me` без прямої команди користувача.
- Не переносити нову production business logic у React/TypeScript.
- Android Compose implementation відтворює затверджений visual language, layout hierarchy, interaction language, colors, cards, sheets, chips, buttons, states та screen structure настільки точно, наскільки це дозволяє native Android.
- Якщо design показує capability, якої ще немає server-side, production Android не підставляє fake data. Surface або неактивний, або показує чесний unavailable/not-yet-active state до відповідного release.

Canonical design tokens включають dark/black surfaces, LinkUp red accent, semantic success/warning/info colors, Outfit/Inter/JetBrains Mono typography contract і відповідні spacing/radius/animation patterns із design reference.

---

# 3. CANONICAL ARCHITECTURE

## 3.1. Android

- Kotlin;
- Jetpack Compose;
- Coroutines + Flow;
- Android lifecycle/process-death handling;
- Android Keystore для secure local secrets/session material;
- native Android APIs для location, camera, BLE, notifications, audio та device security, коли відповідний release їх активує;
- no WebView-first / web-first production app.

## 3.2. Backend

- provider-neutral Go API;
- Go API — єдина app-facing authority для auth/domain/data decisions;
- PostgreSQL/PostGIS — source of truth;
- explicit transactions для race-sensitive mutations;
- idempotent critical commands;
- transactional outbox, коли realtime стає active;
- replaceable adapters для push, mail, map/place, media, translation, weather, billing verification та rewarded providers;
- structured logs без bearer/body/location-secret leakage;
- request IDs, bounded timeouts, abuse/rate limits і explicit error contract.

## 3.3. Infrastructure

Canonical stack:

- **GitHub** — source control, direct work in `main`;
- **Supabase project `oavnrlwsfiiehluubwjk`** — managed PostgreSQL/PostGIS infrastructure;
- **Firebase** — дозволений platform service layer для явно інтегрованих Android capabilities;
- **separate Ubuntu server** — майбутній Go API runtime тільки після прямої deployment-команди користувача.

Заборонено як canonical infrastructure без нового прямого рішення користувача:

- Render.com;
- Google Cloud Build;
- Cloud Run;
- Cloud Deploy;
- Artifact Registry;
- GitHub Actions build/test/deploy pipeline.

Supabase Auth/Data API/Realtime/Storage не є product authority автоматично. Android не отримує database/service-role secrets.

---

# 4. CANONICAL DOMAIN — SLOT

Pulse, Map, LINK і Fly працюють навколо **одного canonical Slot aggregate**. Вони не створюють несумісні паралельні event models.

## 4.1. Lifecycle

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

- `CANCELLED`;
- `EXPIRED`;
- `MODERATED`.

## 4.2. Access modes

Canonical modes:

- `INSTANT`;
- `APPROVAL`;
- `WAITLIST`.

v1.0 release path використовує **APPROVAL** як primary social path. `INSTANT` лишається canonical domain mode. `WAITLIST` стає user-facing production capability у v1.1.

## 4.3. Visibility model

Canonical target:

- Public;
- Friends/Links;
- Selected people;
- City-only;
- Lasso/geo-scoped;
- Travel corridor;
- Private / invite-only.

v1.0 обов’язково має Public. Розширена visibility активується за scope v1.1.

## 4.4. Core invariants

Ці правила діють у всіх releases:

- `accepted_count <= capacity`;
- last-seat allocation atomic;
- duplicate JOIN/REQUEST/membership заборонені;
- critical mutation idempotency обов’язкова;
- stale client version не може тихо перетерти новішу server state;
- client ніколи не є authority для capacity/state/version/entitlement/reliability;
- blocked pair не може взаємодіяти через discovery/query/ranking/realtime/chat;
- terminal Slot не приймає нові JOIN/REQUEST;
- `FULL → FILLING` можливий після LEAVE або authorized removal;
- pending requester не має chat access;
- host/current accepted participant мають chat access тільки доки relationship valid;
- LEAVE/block/kick/revocation прибирає chat authorization;
- terminal Slot закриває chat;
- ephemeral chat rows physically purge на terminal transition, де contract цього вимагає;
- exact location/privacy policy ніколи не обходиться visibility/access mode.

---

# 5. VERSION v1.0 — CORE SOCIAL NETWORK

## 5.1. Goal

v1.0 — найменша **реально працююча production-safe Android соціальна мережа LinkUp**, якою можуть користуватися щонайменше дві реальні людини end-to-end без demo/mock social behavior.

Canonical v1.0 flow:

```text
register/login
  ↓
Me/profile/session
  ↓
create PUBLIC + APPROVAL Slot
  ↓
real Pulse discovery
  ↓
REQUEST
  ↓
host APPROVE / REJECT
  ↓
accepted membership
  ↓
temporary coordination chat
  ↓
START → ACTIVE → COMPLETE
  ↓
terminal chat purge
```

## 5.2. Account & session

Required:

- registration;
- login;
- logout;
- secure opaque bearer session;
- session restore після process death;
- password recovery/reset;
- username;
- display name;
- optional avatar identity field;
- PUBLIC/HIDDEN profile visibility;
- uk/en language preference baseline;
- server-side block list/control;
- secure Android local session storage;
- auth abuse/rate-limit foundation;
- sensitive-log prevention.

## 5.3. Basic Me

Required:

- own profile view;
- edit display name/avatar reference/profile visibility/language preference;
- block list + unblock;
- basic account/session state;
- My LINKs dashboard;
- Hosting / Joined / Requests views using real server data;
- logout;
- legal/version surface.

No fake Reliability/BUMP/Passport metrics in v1.0 unless corresponding server capability is actually active.

## 5.4. LINK / Event Core

Required:

- create PUBLIC Slot;
- APPROVAL access path;
- title;
- activity/scenario;
- optional details/description;
- place/zone text;
- optional date/time;
- organizer identity;
- capacity;
- server lifecycle state;
- create/publish may be one simple v1.0 flow;
- host read/edit/cancel;
- edit title/details/place/time/capacity;
- capacity edit cannot drop below accepted count;
- `expectedVersion` optimistic concurrency;
- cancel is canonical state transition, not hard-delete;
- cancelled Slot disappears from normal Pulse;
- critical create/edit/cancel mutation idempotency.

## 5.5. Approval social loop

Required:

- non-member → REQUEST;
- REQUEST creates pending relation only;
- pending user can withdraw via LEAVE semantics;
- duplicate REQUEST protection;
- host sees pending requester `displayName/@username`;
- host APPROVE;
- host REJECT;
- approval creates membership only server-side;
- atomic last seat;
- accepted roster for host;
- accepted participant LEAVE;
- authorized host remove participant;
- block checks on every transition;
- `FULL → FILLING` reopen;
- host START;
- START → `ACTIVE`;
- host COMPLETE;
- COMPLETE → `COMPLETED`;
- version/state/capacity remain server-authoritative.

## 5.6. Pulse

Required:

- real server data only;
- PUBLIC Slot list;
- title/activity/details;
- place/zone;
- date/time when present;
- organizer identity;
- capacity `x/y`;
- server state;
- viewer relationship state;
- Request / Pending / Leave / Host actions відповідно до authority;
- active Slot remains resolvable for host/current members but is not a new-request target for strangers;
- Loading / Content / Empty / Error / Refreshing states;
- bounded manual refresh is sufficient for v1.0;
- search/filter only over real returned data;
- no fake city/BPM/online/reliability numbers.

## 5.7. Basic Zero-Trace Coordination Chat

For host + current accepted participants:

- ephemeral text messages;
- author display name + username;
- server-created timestamp;
- send;
- bounded message length;
- bounded recent thread;
- manual refresh / bounded foreground refresh is sufficient;
- pending/stranger/left user denied even with known `slotId`;
- block/revocation immediately removes authorization;
- terminal state closes chat;
- PostgreSQL physically purges message rows on `COMPLETED`, `CANCELLED`, `EXPIRED`, `MODERATED`;
- chat is not a permanent messenger/archive.

## 5.8. v1.0 safety/privacy/security

Required before release:

- server authorization for every sensitive action;
- bidirectional block enforcement;
- no public exact stranger GPS;
- no public continuous tracks;
- no client role/state/capacity authority;
- secrets/config separation;
- secure session storage;
- bounded auth/social abuse controls;
- no bearer/body/sensitive location logging;
- IDOR protection;
- replay/idempotency protection;
- production data only;
- no fake success path.

## 5.9. v1.0 stability & release engineering

v1.0 includes its own launch stabilization. It is not deferred to a fake `1.0.1` release.

Required:

- crash/ANR fixes;
- network resilience;
- bounded GET retry policy;
- no blind POST/mutation retry;
- recoverable transport error UX;
- cached/read content not destroyed by refresh failure where caching exists;
- performance/query/index review;
- accessibility baseline;
- Ukrainian + English user-facing baseline for active v1.0 screens;
- API latency/error observability;
- bounded rate-limit storage;
- backup/recovery procedure;
- application rollback procedure;
- Android release build/AAB;
- signing configuration;
- forward-only migrations;
- real two-user social-loop regression;
- no GitHub Actions requirement under current project rules.

## 5.10. Definition of Done — v1.0

v1.0 is Done only when all below are real and verified on Android + Go + PostgreSQL:

1. User A registers/logs in and session restore works after process death.
2. User A creates a real PUBLIC Approval Slot with title/details/place/time/capacity.
3. User A can edit allowed fields; stale version conflict is rejected server-side.
4. User B sees the Slot in real Pulse.
5. User B submits REQUEST and remains pending.
6. User A sees B in pending list and can APPROVE or REJECT.
7. APPROVE produces accepted membership without exceeding capacity under race.
8. Host can see accepted roster; authorized removal/LEAVE correctly updates capacity/state.
9. Pending/stranger cannot use chat; host + accepted can exchange real messages.
10. Block/LEAVE/removal revokes social/chat access.
11. Host can START; current host/accepted participants keep coordination access while ACTIVE.
12. Host can COMPLETE; chat closes and canonical message rows are physically purged.
13. Host can CANCEL; Slot leaves normal discovery and chat rows purge.
14. Duplicate/replayed critical commands do not create duplicate state.
15. Me/profile/block/Hosting/Joined/Requests surfaces use real server data.
16. Loading/content/empty/error/recovery states work without fake data.
17. Android build/tests, Go tests, PostgreSQL integration/race tests and real two-user smoke are actually executed successfully.
18. Security/privacy/accessibility/localization/recovery/rollback gates pass for active v1.0 scope.

---

# 6. VERSION v1.1 — REALTIME REAL-WORLD CITY NETWORK + LINKUP+

## 6.1. Goal

v1.1 перетворює core social network на **live real-world city network**: realtime convergence, durable offline behavior, location-aware discovery, richer access/coordination, verified real-world signals, venue/offline ecosystem capabilities, adaptive experiences та server-authoritative LinkUp+ monetization.

v1.0 capabilities залишаються mandatory regression baseline.

## 6.2. Realtime + durable offline

Required:

- PostgreSQL transactional outbox;
- Slot realtime events;
- user channel;
- city activity channel;
- authoritative snapshot;
- ordered delta stream;
- monotonic cursor/version;
- duplicate/out-of-order protection;
- reconnect convergence;
- foreground/background lifecycle;
- process-death recovery;
- Android durable mutation outbox;
- exact idempotent pending-command replay;
- airplane-mode recovery;
- current block/access re-evaluation while stream active;
- REQUEST/APPROVE/REJECT/LEAVE/FULL/REOPEN/START/COMPLETE convergence between clients;
- realtime pending/roster/access revocation;
- controlled fan-out/LISTEN-NOTIFY wake where appropriate.

Canonical client model:

```text
server snapshot
+ ordered deltas
+ local pending/idempotent command state
```

After reconnect:

1. resubscribe;
2. obtain authoritative snapshot/cursor;
3. replay pending exact idempotent commands;
4. reconcile UI;
5. ignore stale/duplicate deltas.

## 6.3. City Context + scheduled foundation

Required:

- PostGIS locality resolution;
- locality polygons;
- freshness/accuracy policy;
- approximate/precise permission classes;
- City-Lock;
- boundary hysteresis/stability;
- stale/fake/low-accuracy rejection;
- privacy-safe city context;
- NOW / Scheduled separation;
- timezone/DST-safe scheduling;
- expiry policy;
- city-only visibility foundation;
- no public exact stranger coordinates.

## 6.4. Map v1

Required:

- native Android map surface preserving frozen design;
- replaceable map/place provider adapter;
- canonical place identities;
- server viewport queries;
- Slot pins/aggregates, never public people pins;
- privacy-safe exposure;
- shared city/time scope with Pulse;
- open Slot from Map;
- JOIN/REQUEST according to access mode;
- clustering/viewport performance;
- no raw full-city user download.

Google Maps/Places may be used as an Android mapping/place provider if configured, but they are not domain authority and do not change the canonical no-GCP-build/deploy rule.

## 6.5. LINK / hosting expansion

Required:

- explicit DRAFT;
- preview;
- separate publish;
- richer edit/history surface;
- explicit Now / Scheduled modes;
- canonical place identity;
- structured vibe tags;
- advanced visibility baseline;
- create command survives network loss/process death without duplicate Slot;
- richer Hosting/Joined dashboard;
- configurable access mode surface preserving Instant + Approval;
- advanced roster/request summaries.

## 6.6. Approval + Waitlist + Host Control V2

Required:

- Waitlist;
- waitlist promotion;
- request expiry/withdrawal hardening;
- FULL/reopen race handling;
- expiry during mutation;
- richer host removal/control;
- capacity/access transitions;
- concurrency-safe approval/waitlist transactions;
- block/authorization on every transition;
- version conflict UX;
- complete roster/request/waitlist states.

## 6.7. Coordination Chat V2

Still zero-trace by product design:

- realtime delivery;
- offline/idempotent send retry;
- duplicate convergence;
- activation/expiry/retention rules;
- system messages for important Slot lifecycle events;
- reconnect thread convergence;
- stronger kick/block/revocation handling;
- physical terminal cleanup verification;
- no permanent social-message archive by default.

## 6.8. Notifications

Notifications are server-authoritative domain projections, not direct side effects of HTTP handlers.

Canonical delivery flow:

```text
domain transaction
→ transactional outbox
→ notification projector/worker
→ dedupe(idempotency key)
→ TTL check
→ quiet-hours check
→ frequency-cap check
→ FCM/APNs adapter
```

Rules:

- push is never called directly from an HTTP handler;
- the domain mutation and outbox record commit atomically or neither is considered complete;
- notification delivery may retry independently without replaying the domain mutation;
- idempotency/dedupe is required per logical notification;
- expired notifications are dropped before provider delivery;
- capability registry controls `notifications` and remains fail-closed by default.

Canonical notification types:

- `MESSAGE`;
- `EVENT`;
- `EVENT_RECOMMENDATION`;
- `EVENT_REMINDER`;
- `FRIEND_REQUEST`;
- `FRIEND_ACCEPTED`;
- `SYSTEM`;
- `SECURITY`;
- `ACCOUNT`;
- `PROMO` / `ADVERTISEMENT` for LinkUp+ marketing only where consent/policy allows.

Grouping/collapse:

- multiple `MESSAGE` notifications from the same sender/thread collapse into one grouped surface;
- repeated event state changes collapse by Slot/event identity when the newest state supersedes the older one;
- grouping must not merge security/account alerts with marketing or social notifications.

Canonical deep-link routing must open the relevant destination, never generic Home:

- `MESSAGE` → `app://chat/{slotId}`;
- `EVENT`, `EVENT_REMINDER`, waitlist/reopen lifecycle → `app://slot/{slotId}`;
- `EVENT_RECOMMENDATION` → `app://pulse?slotId={slotId}`;
- `FRIEND_REQUEST` → `app://me/requests`;
- `FRIEND_ACCEPTED` → `app://me/connections`;
- `SECURITY` → `app://me/security`;
- `ACCOUNT` → `app://me/account`;
- `PROMO` / `ADVERTISEMENT` → `app://linkup-plus` or another explicitly campaign-scoped destination.

User controls:

- category toggles exist for social/event/recommendation/promo classes where policy permits disabling;
- `SECURITY` and critical account-integrity notifications cannot be silently converted into marketing controls;
- default quiet hours for non-critical notifications are **23:00–08:00 local time**;
- quiet hours use the user's current configured timezone and must be DST-safe;
- recommendation and promo classes have explicit frequency caps;
- `MESSAGE` and `SECURITY` are not frequency-capped by marketing/recommendation limits;
- preference changes take effect server-side for future projections/delivery decisions.

Admin campaigns:

- authorized admin roles may create segmented campaigns;
- test push must be available before broad delivery;
- scheduled delivery uses server time plus recipient-local policy where configured;
- analytics include `targeted`, `sent`, `delivered`, `opened` and CTR without sensitive social/location payload;
- campaign create/edit/send/cancel operations require explicit permissions and immutable audit events;
- campaign tooling cannot bypass user notification preferences, age/region policy, quiet hours or frequency caps except explicitly defined critical-system paths.

Required evidence:

- request/approval decisions, cancellation, starting soon, reopened seat/waitlist promotion, safety/moderation and coordination events project through the outbox path;
- dedupe, TTL, grouping, deep links, preferences, quiet hours and frequency caps are verified;
- provider delivery/reliability metrics exist without treating provider acceptance as domain success.

## 6.9. BUMP + Reliability

Required:

- BUMP proof baseline;
- verified attendance;
- server-verified proof;
- Android Keystore-backed device proof where applicable;
- anti-replay;
- anti-farm;
- one-event/one-contribution protections;
- reliability events;
- private/public reliability bands;
- BUMP Vault baseline;
- after-check flow;
- verified real-world completion metrics.

Client tap alone can never increase trust/reliability.

## 6.10. City BPM + map intelligence

Required:

- City BPM;
- activity buckets;
- normalized city baseline;
- time decay;
- minimum cohort;
- dominant categories/tags;
- privacy suppression;
- realtime city telemetry;
- Kinetic Proof presentation;
- spatial/time blur;
- anti-manipulation weighting;
- privacy hex grid;
- Vibe Topology;
- Echo Hotspots;
- Lasso;
- shared GeoTemporalScope Pulse ↔ Map;
- Time-Lapse baseline;
- Dark Zone Ignition;
- accessibility labels/patterns beyond color.

Low cohorts must be suppressed so a small group or person cannot be inferred.

### Recommendation / ranking pipeline

Every recommendation surface (Pulse, Map intelligence, Auto-Swarms, Fly discovery, adaptive discovery and AR/ranking) uses the same ordered eligibility boundary.

**Layer 1 — mandatory deterministic eligibility filters**

Always evaluated first and never bypassed by ranking, ML, monetization, venue status or admin campaign logic:

- block relationships;
- privacy/visibility rules;
- safety/moderation eligibility;
- Slot lifecycle and access mode;
- age/region/content policy where applicable;
- capacity/waitlist state;
- City Context / locality eligibility;
- physical-presence requirements where the feature requires them.

An item rejected by Layer 1 is not eligible for scoring in later layers.

**Layer 2 — deterministic ranking from explicit user signals**

Eligible candidates may be ordered using only explicit/product-authoritative signals such as:

- interests selected by the user;
- activity/category match;
- current City Context/locality;
- time window / NOW vs Scheduled relevance;
- explicit language/accessibility/intent filters;
- deterministic freshness, distance bucket or lifecycle relevance where privacy rules allow.

Layer 2 must remain inspectable and reproducible from canonical inputs.

**Layer 3 — opt-in personalization from participation history**

This layer is disabled unless the user gives separate explicit consent in **Settings → Privacy**.

- history-based personalization may use aggregated/coarse patterns from verified participations or places;
- raw GPS tracks and continuous routes are forbidden;
- retained location-derived patterns must stay locality/coarse-bucket level and follow the existing City Context privacy contract;
- disabling consent immediately stops future use and deletes collected personalization-history data without waiting for the normal retention window;
- disabling Layer 3 must not degrade access to deterministic Layer 1/Layer 2 discovery;
- consent state is server-authoritative and auditable without storing sensitive raw location history.

ML/learned ranking, when enabled, operates only inside the candidate set that survived Layer 1 and after deterministic product constraints. It may refine ordering but never restore an ineligible candidate or override privacy, block, safety, capacity or access decisions.

Anti-manipulation invariant: venue identity, Venue Perks, Host status, LinkUp+ payment or any other commercial consideration cannot purchase organic ranking, Hotspot, City BPM or recommendation priority. Sponsored/promotional content, if ever displayed, must be explicitly labeled and handled as a separate policy-controlled surface rather than disguised organic rank.


## 6.11. Auto-Swarms

Required:

- normalized intent taxonomy;
- clustering;
- confidence score;
- proposal lifecycle;
- explicit consent;
- neutral meeting zone;
- realtime proposal UI;
- false-positive analytics;
- privacy/safety/block filters before grouping.

Auto-Swarm never auto-joins users.

Auto-Swarm candidate generation must consume the same Layer 1 deterministic eligibility boundary from §6.10 before clustering. Confidence/clustering can prioritize only already-eligible candidates and cannot bypass block, privacy, safety, capacity, locality or consent rules.

## 6.12. Fly

### Fly Now

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

### Fly Travel

- Astral remote-city context;
- remote scheduled joins;
- Road-Trip Slots;
- Just Landed;
- transit contexts;
- Event Squads;
- Local Ambassador;
- Highway Flare / non-emergency road help;
- Nomad/Home Cities;
- macro city activity map.

ASTRAL context must never pretend to be physical presence.

### Fly Motion

- activity recognition;
- motion confidence;
- driver/passenger distinction;
- low-interaction HUD;
- trajectory Slot discovery;
- safe-state interaction;
- anonymous encounter tokens;
- Slipstream aggregates;
- battery profiling.

Probable driver state must not encourage tapping/swiping/typing while driving.

## 6.13. Me 2.0 + Squad Radar

Required:

- Social Passport;
- Links Graph;
- Reliability Card;
- Now Card;
- Activity Cockpit;
- Real-World Footprint;
- expanded BUMP Vault;
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
- sharing modes `OFF / ETA_ONLY / APPROXIMATE / LIVE_PRECISE`;
- exact sharing opt-in + TTL + instant revoke;
- block/kick membership revocation.

## 6.14. AR + Ranking V2

Required:

- AR Slot/Place beacons;
- capability detection;
- camera/IMU;
- coarse anchors;
- non-AR fallback;
- accessibility fallback;
- learned ranking;
- preference modeling;
- notification intelligence V2;
- swarm tuning;
- explainability/debug tooling;
- opt-in personalization.

Ranking V2 is downstream of the §6.10 recommendation pipeline:

1. Layer 1 deterministic eligibility/privacy/block/safety/capacity/access filters produce the only legal candidate set;
2. Layer 2 deterministic explicit-signal ranking establishes an inspectable baseline;
3. Layer 3 history-based personalization is optional and requires explicit Settings → Privacy consent;
4. learned/ML scoring may refine order only inside the surviving candidate set.

ML never overrides deterministic eligibility/privacy/block/safety/capacity/access filters, never converts a filtered candidate back into an eligible one, and must expose enough debug/explainability information to diagnose ranking without exposing another user's sensitive data. A non-personalized deterministic fallback is mandatory when consent is absent, models/providers fail, or feature capability is disabled.

## 6.15. Cold start + venue ecosystem

Required:

- Ignite City;
- launch pool anti-sybil;
- activation fan-out;
- verified venue identity;
- Venue Perks;
- qualification/redemption;
- Venue Vibe feedback/snapshots;
- accessibility venue attributes;
- venue fraud protection;
- canonical external place identity + internal venue identity.

Guardrails:

- no fake users/Slots/online counts;
- venue payment/perk cannot buy organic Hotspot/City BPM/ranking;
- venue feedback never leaks individual participant identity.

## 6.16. Offline real-world operations

Required:

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
- real Android BLE/device tests.

Offline proof is pending until server verification; offline device data alone cannot create trust.

## 6.17. Safety + accessibility expansion

Required:

- Ghost Guardian;
- temporary opaque guardian links;
- `STATUS_ONLY`;
- `ETA_APPROXIMATE`;
- `SAFETY_RADAR`;
- explicit `LIVE_PRECISE`;
- indistinguishable/burned access after expiry/revoke/completion;
- Barrier-Free Mode;
- strict/highlight accessibility filtering;
- privacy-safe accessibility preferences;
- stronger home/private scenario policy;
- guardian abuse/rate limits.

Exact location sharing is always explicit, temporary, revocable and minimized.

## 6.18. Ephemeral media & communication

Required:

- Slot Cam;
- transient encrypted media relay;
- Memory Drop;
- local final gallery/collage;
- server cleanup verification;
- Auto-Translate Hub with original text canonical;
- Sonic Vibes / short audio intent;
- Bill Splitter calculator with deterministic minor-unit rounding;
- chat/media offline/reconnect integration.

Guardrails:

- no permanent cloud album by default;
- no permanent translation-only copy;
- no background microphone;
- Bill Splitter is not a payment processor.

## 6.19. Adaptive experience

Required:

- Host Perks;
- Zen Mode;
- Surprise Me / Magic Button;
- Weather-Triggered Swarms;
- Asset Match;
- adaptive ranking integrations;
- anti-spam/anti-farm;
- expanded preference/privacy controls.

Guardrails:

- Host Perks are not purchasable ranking power;
- Zen does not penalize or collect invasive raw device history;
- Surprise Me cannot bypass Approval/eligibility/physical-presence rules;
- Weather Swarms never auto-join;
- Asset Match must reject unsafe/regulated asset categories according to product safety policy.

## 6.20. LinkUp+ billing foundation

`docs/LINKUP_PLUS_MONETIZATION.md` is the detailed authority for LinkUp+ pricing, plan durations, entitlement composition, rewarded access, referral rewards, billing state, purchase verification and monetization anti-fraud. README intentionally keeps only the release-level engineering contract; if the two documents differ on those details, the monetization contract controls unless `PROJECT_RULES.md` or a newer explicit roadmap decision says otherwise.

Required at release level:

- Google Play purchase/restore on active Android scope;
- server-side purchase validation and provider-event verification;
- server-authoritative entitlement state machine including active, grace/retry, expiry and revoke semantics defined by the monetization contract;
- account/device switching and restore behavior;
- subscription management in Me;
- refund/revoke/grace handling;
- billing observability;
- bounded entitlement cache/offline behavior;
- replay-safe/idempotent provider processing;
- capability-gated rollout and rollback.

Client `isPlus=true`, local receipt state or UI purchase success is never authority.

Free core safety/privacy/create/join functionality must not be paywalled, and LinkUp+ entitlement cannot bypass block, moderation, lifecycle, capacity, access or privacy rules.

## 6.21. LinkUp+ Travel Pro

Required:

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

## 6.22. LinkUp+ Host Power Tools

Required:

- Mega-Slots up to product ceiling;
- scalable roster/pagination/realtime;
- Stealth Slots / invite-link-only visibility;
- opaque revocable invite tokens;
- advanced Slot Blueprints;
- no cloning of old participants/private locations/tokens;
- Co-Hosting;
- granular organizer permissions;
- owner protection;
- audited grants/revokes.

## 6.23. LinkUp+ Advanced Discovery

Required:

- Radius Overdrive with bounded policy-defined radius;
- viewport/cursor/aggregate queries;
- Vibe-Match Filters;
- language/interests/assets/accessibility-compatible filters;
- Auto-Pilot Join rules;
- explicit automation scope;
- Approval → REQUEST only;
- Instant auto-JOIN only with explicit opt-in;
- daily caps/cooldowns/dedupe;
- Pulse Time-Machine;
- historical privacy-safe aggregate city/map data;
- no individual route reconstruction.

## 6.24. LinkUp+ Privacy & QoL

Required:

- Ghost Mode;
- no passive public presence/BPM contribution while Ghost active;
- Priority Boarding visual/request aid without seat/waitlist advantage;
- Oops-Rewind through compensating commands;
- Multi-Threading / intent groups;
- atomic winner lock;
- no silent cancellation after external commitment;
- Guardian Auto-Ping;
- trusted guardian contacts;
- no automatic LIVE_PRECISE without explicit consent.

## 6.25. Identity, analytics & themes

Required:

- Hex-Aura cosmetic treatment only;
- no ranking/BPM/Hotspot benefit from cosmetics;
- BUMP Vault Pro self-only analytics;
- unique people met;
- repeat BUMPs;
- verified meetup count;
- activity/city trends;
- no exact route reconstruction;
- Custom App Icons;
- OLED Black / Neon Cyberpunk class themes where included by frozen product design;
- contrast/font scaling/Reduce Motion QA.

## 6.26. Forgiveness + billing hardening

Required:

- No-Strike Forgiveness;
- max one non-stacking eligible credit per policy period;
- original cancellation event remains auditable;
- no no-show/safety/moderation forgiveness;
- bounded reliability modifier;
- webhook replay/idempotency;
- renewal/expiry/refund/chargeback/revoke;
- grace/billing retry;
- restore after reinstall/device change;
- account switching;
- Mega-Slot load tests;
- Stealth penetration tests;
- Auto-Pilot abuse/rate tests;
- Ghost privacy audit;
- Guardian delivery/privacy audit;
- historical Map inference/privacy tests;
- subscription localization/cancellation UX;
- feature flag rollback;
- production smoke.

## 6.27. LinkUp+ Free Day / rewarded access

The detailed rewarded-access state machine, timing, cooldown, provider-verification and precedence rules are canonical in `docs/LINKUP_PLUS_MONETIZATION.md` §6 and §8. This README section is the release-level summary and must not diverge from that authority.

Required:

- rewarded quest available only inside LinkUp+ surface;
- five verified rewarded views;
- minimum 4 hours between verified steps;
- 24-hour quest window;
- fifth verified view → atomic 24-hour Plus grant;
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
- paid subscription precedence without reward stacking.

Guardrails:

- no ads in Pulse/Map/LINK/Fly/chat/safety flows;
- no forced interstitials;
- no click/install/purchase requirement;
- no precise social/location data for ad targeting;
- probable driver state never launches rewarded video;
- an already valid earned grant is not silently removed by feature kill-switch.

## 6.28. Final ecosystem hardening

Required:

- Android real-device BLE matrix;
- Keystore key-loss/reinstall flows;
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
- billing/store validation;
- Android production smoke.

## 6.29. Definition of Done — v1.1

v1.1 is Done only when:

- v1.0 regression remains green;
- two Android clients converge after stream loss/reconnect/process death/airplane mode;
- durable commands do not duplicate Slot/membership/chat state;
- City Context obeys accuracy/privacy boundaries;
- Pulse/Map share canonical server scope;
- Waitlist/host-control races are transaction-safe;
- Chat V2 converges without becoming permanent storage;
- push deep links/dedupe/TTL/preferences work;
- BUMP/Reliability cannot be forged by client-only actions;
- City BPM/Hotspots suppress low cohorts;
- Swarms never auto-join;
- Fly respects driver/location/privacy guardrails;
- Squad/Guardian-style sharing is explicit, revocable and TTL-bound where active;
- AR/ranking has privacy-safe fallback and deterministic eligibility boundary;
- Android/Go/PostgreSQL integration, offline/realtime/device tests and security/privacy tests are executed successfully.
- venue/cold-start systems contain no fake social state;
- BLE/offline proof cannot mint trust before server verification;
- Guardian/accessibility/location policies pass abuse/privacy tests;
- ephemeral media and translation cleanup is verifiably bounded;
- adaptive features cannot bypass deterministic safety/access rules;
- Play Billing entitlement is server-authoritative across renew/refund/revoke/restore/account switch;
- Plus never paywalls free core safety/social access;
- Stealth/co-host/automation/ghost/time-machine flows pass security/privacy/race testing;
- rewarded access is server-verified, replay-safe and isolated from core social surfaces;
- full Android/Go/PostgreSQL/Firebase/provider integration and restore/rollback/security/load smoke gates are executed successfully.

---

# 8. CROSS-VERSION PRIVACY / SAFETY / SECURITY

These are non-negotiable in every active release.

## 8.1. Privacy

- no public exact stranger GPS;
- no public continuous tracks;
- exact sharing only opt-in, temporary and revocable;
- raw location retention minimized;
- low-cohort aggregates suppressed;
- private/home Slot inference through heatmaps/history forbidden;
- blocked users filtered server-side from every active social/query/realtime/chat layer;
- sensitive location never sent to unrelated analytics/ad targeting.

## 8.2. Security

- server authorization for every sensitive operation;
- no trusted client role/entitlement/capacity flags;
- secrets never embedded in Android/Git history;
- strong opaque tokens/invite/challenge identifiers;
- bounded session/device lifecycle;
- rate limits and abuse protection;
- mutation idempotency;
- replay protection where applicable;
- critical mutation audit/domain events;
- IDOR tests;
- sensitive logging prevention;
- fail-closed provider/security boundaries.

## 8.3. Anti-stalking

Location, Fly, Squad, Guardian, Map, history and ranking features must be designed so a stranger cannot derive a person’s continuous route, home pattern, exact current position or private gathering from public/aggregate surfaces.

---

# 9. UX STATE CONTRACT

Every production screen/feature must expose relevant explicit states instead of silent failure:

- Loading;
- Content;
- Empty;
- Refreshing where relevant;
- OfflineCached where relevant;
- OfflinePending where relevant;
- ErrorRecoverable;
- ErrorBlocking.

Rules:

- mutation success comes only from server-authoritative response;
- stale responses cannot restore disposed/account-old UI state;
- double taps cannot create duplicate critical mutations;
- cancelled coroutines/requests must not keep locks forever;
- process death/recreation must not lose security-critical session state or duplicate durable commands;
- accessibility cannot rely only on color.

---

# 10. API CONTRACT

## 10.1. General

- HTTPS outside explicit local emulator development;
- bearer auth for protected routes;
- bounded request bodies;
- unknown-field handling is explicit;
- generic auth errors avoid account enumeration where required;
- error responses expose stable machine code + safe user message + request ID;
- server timestamps are canonical;
- client-supplied actor/user IDs never replace authenticated actor identity.

## 10.2. Mutations

Critical mutations require:

- idempotency key;
- actor authorization;
- block/privacy/eligibility check;
- state/version precondition where relevant;
- transaction boundary for multi-row/race-sensitive state;
- deterministic conflict semantics;
- no blind transport retry.

## 10.3. Reads

- bounded pagination/limits;
- privacy-aware filtering server-side;
- no full-city raw people dump;
- no reliance on client filtering to enforce sensitive visibility;
- GET retry only within bounded policy.

---

# 11. DATABASE / MIGRATION CONTRACT

- PostgreSQL/PostGIS is canonical storage;
- schema changes only through **forward-only migrations**;
- already-applied migrations are never silently edited;
- migration ledger/checksum drift must fail closed;
- DB constraints backstop critical invariants where possible;
- row locks/transactions are explicit for capacity/waitlist/winner races;
- ephemeral retention/purge rules must exist at canonical storage boundary when required;
- indexes are reviewed for active release query patterns;
- production database changes are not applied until the user explicitly authorizes the relevant environment/deployment work.

---

# 12. ANDROID CONTRACT

## 12.1. Lifecycle

Required as relevant to release:

- process death;
- configuration recreation;
- foreground/background;
- network loss/reconnect;
- expired/revoked session;
- Keystore key loss;
- account switching;
- permission denied/approximate permission;
- battery-safe polling/realtime behavior.

## 12.2. Build/security

- release API endpoint is build-time/config supplied;
- release must fail closed if production API URL is absent;
- cleartext traffic disabled in release;
- debug localhost/emulator exception is explicit only;
- release signing material never enters Git;
- APK/AAB is not committed to Git or stored in Supabase;
- release artifact handoff follows `PROJECT_RULES.md`.

## 12.3. Accessibility/localization

Every active release must maintain:

- touch target correctness;
- screen-reader labels;
- non-color-only state cues;
- font scaling;
- contrast;
- Reduce Motion handling where animations exist;
- Ukrainian + English baseline for active product surfaces;
- equivalent safety/error meaning across supported language strings.

---

# 13. TEST CONTRACT

Documentation or source test files do not count as verification until actually executed.

## 13.1. v1.0 mandatory matrix

At minimum:

- registration/login/logout/session restore;
- recovery/reset;
- block/unblock;
- create idempotency;
- Slot serialization/readback;
- host edit authorization;
- stale version conflict;
- capacity edit floor;
- CANCEL state/discovery removal;
- request/withdraw;
- pending cannot self-accept;
- host-only APPROVE/REJECT;
- last-seat concurrency;
- duplicate request/membership prevention;
- accepted LEAVE;
- host participant removal;
- START/COMPLETE transitions;
- pending/stranger/left chat rejection;
- host/member chat send/read;
- block/LEAVE revocation;
- terminal physical chat purge;
- IDOR/replay/rate-limit/log secrecy;
- Pulse/Me/My LINKs loading/empty/error states;
- Android unit/UI/instrumentation where risk requires it;
- Go unit/integration/race tests;
- PostgreSQL migration/integration tests;
- real two-account Android ↔ Go ↔ PostgreSQL smoke.

## 13.2. v1.1 mandatory matrix

Includes v1.0 regression plus:

- stream drop/duplicate/out-of-order;
- background/foreground/process kill/airplane mode;
- pending command replay;
- two-client convergence;
- city boundary jitter/accuracy/stale location;
- permission denied/approximate mode;
- Map privacy/viewport/low cohort;
- Waitlist promotion/concurrency/expiry;
- Chat V2 reconnect/duplicate/system-message/retention;
- notification dedupe/TTL/deep-link/preferences;
- BUMP replay/anti-farm/device proof;
- City BPM manipulation/low cohort;
- Swarm consent/false-positive/block filters;
- Fly driver/passenger safety;
- Squad precise-share expiry/revoke;
- AR fallback/ranking explainability/privacy filters.

### Unified real-world ecosystem + LinkUp+ coverage

Additionally required within the same v1.1 release gate:

- BLE offline proof/reconciliation/replay;
- Swarm BUMP quorum/concurrency;
- Guardian token/expiry/revoke/cache/abuse;
- Barrier-Free strict/highlight/privacy consistency;
- media permission/encryption/cleanup/process death;
- translation provider/reconnect/retention/original-text;
- audio duration/permission/driver/offline;
- Bill Splitter rounding/auth/version races;
- Host Perks farm/moderation/ranking cap;
- Zen privacy/no penalty;
- Surprise Me eligibility/idempotency;
- weather stale/duplicate/outage/quiet hours/consent;
- Asset Match TTL/privacy/unsafe-asset rejection;
- Play Billing replay/renewal/expiry/grace/refund/restore/account switch;
- Mega-Slot load/capacity ceiling;
- Stealth token guessing/revoke/rotation/discovery leakage;
- Co-Host permission escalation/revoke/audit;
- Auto-Pilot duplicate/cap/quiet-hours/block/disable race;
- Ghost presence/privacy audit;
- Oops-Rewind expiry/irreversible/version conflicts;
- Multi-Threading simultaneous winner/process-death;
- rewarded Free Day timing/provider verification/multi-device race/atomic grant/cooldown.

---

# 14. RELEASE ENGINEERING

For every version:

- release decision is based on executed evidence, not source presence;
- migrations are forward-only;
- privacy/security review required;
- test matrix for active release executed;
- Android build/AAB produced and signing verified;
- artifact integrity/SHA-256 recorded when release artifact is produced;
- no fake/mock production path;
- accessibility/localization checked;
- observability sufficient for active scope;
- backup/recovery/rollback path exists;
- no GitHub Actions or Google Cloud build/deploy unless `PROJECT_RULES.md` is explicitly changed;
- Ubuntu runtime/deployment is not touched until direct user command.

---

# 15. DEPENDENCY ORDER

Canonical work order:

```text
v1.0
  account/session/security
  → Slot core
  → Approval/membership/roster
  → Pulse/LINK/Me native binding
  → basic zero-trace chat
  → lifecycle + block/revocation
  → build/DB/device verification
  → v1.0 release gates

v1.1
  transactional outbox + durable Android outbox
  → realtime convergence
  → City Context + scheduled/expiry
  → Map + place identities
  → hosting/access/waitlist expansion
  → Chat V2 + notifications
  → BUMP/Reliability
  → City BPM + Map intelligence
  → Auto-Swarms
  → Fly Now/Travel/Motion
  → Me 2.0/Squad
  → AR/ranking
  → venue/cold start
  → offline BLE proof
  → safety/accessibility expansion
  → ephemeral media/translation/audio
  → adaptive experience
  → Play Billing foundation
  → Plus travel/host/discovery/privacy/identity
  → billing hardening
  → rewarded Free Day
  → final ecosystem hardening
  → unified v1.1 release gates
```

Не перескакувати fundamental dependency заради видимого feature, якщо це створює другу authority або майбутню переробку.

---

# 16. CURRENT IMPLEMENTATION STATUS

README навмисно **не дублює** implementation status.

Перед будь-яким work block обов’язково читати:

1. `PROJECT_RULES.md` — non-negotiable rules;
2. `README.md` — release/product contract;
3. `IMPLEMENTATION_STATUS.md` — що реально вже зроблено, які commit SHA, які verification gates відкриті;
4. `REPOSITORY_AUDIT.md`, якщо він існує — відомі дефекти/ризики, які ще не виправлені.

Якщо README каже, що capability required, а `IMPLEMENTATION_STATUS.md` каже “не verified” — capability **не вважається готовою**.

---

# 17. CURRENT TARGET

**Active release target: LinkUp v1.0 — Android + Go.**

Поточний priority до переходу на v1.1:

1. не змінювати frozen design;
2. завершити й перевірити v1.0 account/social/chat/lifecycle flow;
3. довести Android/Go build/test infrastructure до reproducible green state;
4. виконати PostgreSQL migrations/integration/race tests у дозволеному environment;
5. виконати реальний two-user Android ↔ Go ↔ PostgreSQL smoke;
6. закрити v1.0 security/privacy/accessibility/localization/recovery/rollback gates;
7. лише після green v1.0 перейти до v1.1 transactional outbox/realtime/City Context dependency chain.

**iOS не чіпати до прямої команди користувача.**