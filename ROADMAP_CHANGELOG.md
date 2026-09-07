# LinkUp — ROADMAP CHANGELOG

Цей файл документує зміни структури product roadmap. Він **не є implementation status** і не підвищує production readiness. Фактичний стан коду, перевірок і commit history дивитися в `IMPLEMENTATION_STATUS.md`.

## 2026-09-07 — Full README roadmap rebase

### Причина

Старий `README.md` одночасно:

- називав весь продукт одним релізом `1.0.0`;
- містив історичні capability headings від `1.0.0` до `2.12.0`;
- казав, що всі ці блоки блокують один `1.0.0` Done;
- містив historical “already implemented” claims, які не були надійним джерелом фактичного стану current repository;
- містив iOS release gates, хоча `PROJECT_RULES.md` прямо freeze-ить iOS.

Це створювало неоднозначність між release scope, implementation order і factual implementation status.

### Нове canonical versioning

README повністю переписаний під три послідовні реальні Version 1 releases:

- **v1.0 — Core Social Network**;
- **v1.1 — Realtime City Network**;
- **v1.2 — Real-World Ecosystem + LinkUp+**.

Поточний active release: **v1.0**.

Release boundary:

- v1.0 не блокується scope v1.1/v1.2;
- v1.1 активується після green v1.0;
- v1.2 активується після green v1.1;
- capability, реалізована раніше свого release, не видаляється і проходить regression/DoD у своєму release gate.

### Mapping старого roadmap у новий

| Старий historical block | Новий release |
|---|---|
| `1.0.0` core social foundation | **v1.0** |
| `1.0.1` launch stabilization | **v1.0 release gate** |
| `1.0.2` realtime/offline → `1.12.0` AR/ranking | **v1.1** |
| `2.0.0` venue/cold-start → `2.12.0` rewarded/final hardening | **v1.2** |

Всі product capabilities старого roadmap збережені у новому README, але перегруповані в зрозумілі release boundaries.

### Що принципово змінилося

- Видалені псевдо-release headings `1.0.1 … 2.12.0`.
- Видалене правило “весь README блокує один v1.0”.
- README більше не містить historical “already implemented” assertions як evidence.
- `IMPLEMENTATION_STATUS.md` лишається єдиним factual ledger для “є / немає / verified / not verified”.
- iOS не є gate для v1.0/v1.1/v1.2, доки користувач прямо не розблокує iOS.
- Frozen React/TypeScript design лишається canonical visual contract; redesign не дозволений.
- Android implementation — Kotlin/Jetpack Compose; backend/domain authority — Go.
- Canonical data layer — PostgreSQL/PostGIS; Go API — app-facing authority.
- Current infrastructure restrictions GitHub + Supabase + Firebase + Ubuntu залишені без змін.

### Documentation commits

- Full `README.md` rewrite: `1adf9e6de6dbd18575b95d771714b57039ebb14a`
- `PROJECT_RULES.md` versioning/readiness synchronization: `574bc7b95718c580eaa7c4024340dc0795262116`

### Rule for old worklog entries

Старі записи в `IMPLEMENTATION_STATUS.md` та `REPOSITORY_AUDIT.md`, створені до цього rebase, можуть містити historical labels `1.0.1`, `1.0.2`, `1.1.0`, `2.x` або формулювання “full Version 1 scope”.

Їх **не треба переписувати заднім числом**. Вони є історичним audit trail. Для нового planning їх потрібно трактувати через mapping вище та актуальний `README.md`.

### Current planning rule after rebase

Перед новим work block читати в такому порядку:

1. `PROJECT_RULES.md`;
2. `README.md`;
3. `IMPLEMENTATION_STATUS.md`;
4. `REPOSITORY_AUDIT.md` — якщо relevant;
5. цей `ROADMAP_CHANGELOG.md` — тільки для version-history context.

Не відновлювати старі `1.0.1 … 2.12.0` roadmap headings без прямої нової команди користувача.
