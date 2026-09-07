# LinkUp — FROZEN DESIGN PARITY CHECKLIST

Status: **active visual implementation contract**.

This checklist exists to implement the already approved React/TypeScript LinkUp design in production Android without redesigning, simplifying, expanding, or reinterpreting it. The frozen React/TS files remain the visual source of truth. Kotlin/Compose must match them before new product functionality or animation work is layered on top.

Baseline reviewed on Ubuntu: `16b6ed30fc54ec89eddc625e54734ff83984848f`.

## Mandatory order

1. Freeze and inventory the complete React/TS visual contract.
2. Build shared Compose design primitives/tokens to match it.
3. Match every frozen screen and state visually.
4. Complete visual QA against the frozen source and the 35 device screenshots.
5. Only after `DESIGN PARITY = 100%`, bind production functionality into the completed UI.
6. Only after functional binding, implement animations/micro-interactions.

Until parity is complete, do not add new product surfaces merely because backend capability exists.

---

## 1. Frozen visual source files

### App shell / screens

- `App.tsx`
- `PulseScreen.tsx`
- `MapScreen.tsx`
- `CreateLinkScreen.tsx`
- `FlyScreen.tsx`
- `MeScreen.tsx`
- `NotificationsPanel.tsx`
- `BottomNav.tsx`
- `SlotCard.tsx`

### Shared primitives

- `Avatar.tsx`
- `Button.tsx`
- `Card.tsx`
- `Chip.tsx`
- `ProgressBar.tsx`
- `Sheet.tsx`
- `Skeleton.tsx`
- `StatusBadge.tsx`

### Tokens / state / reference data

- `index.css`
- `tailwind.config.js`
- `types.ts`
- `data.ts`
- `StateToggle.tsx` — state reference only; do **not** ship the visible developer toggle in production Android.

---

## 2. Global visual tokens

### Colors — exact contract

- background `#050506`
- surface `#0D0E10`
- elevated `#141518`
- zone `#1A1C20`
- red `#FF2D35`
- red signal `#FF3B42`
- red deep `#9F171E`
- critical `#EF4444`
- success `#22C55E`
- warning `#F59E0B`
- info `#3B82F6`
- text primary `#F7F8FA`
- text dimmed `#A5A9B0`
- text muted `#747982`
- border `#24262B`

Current Android status: **colors already match** in `ui/theme/LinkUpTheme.kt`.

### Typography

Frozen families:

- display: `Outfit`
- body: `Inter`
- mono: `JetBrains Mono`

Android gap: current theme defines colors but does **not** yet establish the frozen three-family typography system globally.

### Global frame / surfaces

- mobile design frame max width: `420px` reference width
- black background
- hidden visual scrollbars in prototype reference
- safe-bottom handling
- `glass`: `rgba(13,14,16,.72)` + 20px blur
- `glass-elevated`: `rgba(20,21,24,.78)` + 24px blur

Android gap: existing surfaces approximate opacity/backgrounds but do not yet provide one canonical glass/elevated implementation.

### Shape language

Reference radii repeatedly used:

- 6px status badge
- 8px compact control/tag
- 12px standard control/input
- 16px card/control
- 24px sheet top corners / large empty-state tile
- full circle for avatar, LINK CTA and map pins

Compose must use shared shapes instead of per-screen ad-hoc values.

---

## 3. Shared primitive parity

### Avatar

Reference sizes:

- sm: 24px
- md: 32px
- lg: 48px
- xl: 80px

Contract:

- circular
- background = avatar color with `0x22` alpha
- 1.5px solid avatar-color border
- bold primary text

Current Android gap: several screens use square/rounded initial tiles instead of the canonical round bordered avatar.

### Button

Variants required:

- primary
- secondary
- ghost
- danger
- success

Sizes required:

- sm
- md
- lg

Contract includes active scale feedback, but motion implementation is deferred until the animation stage. Static colors/shape/spacing are part of design parity now.

### Card

- elevated background
- 1px border
- 16px radius
- optional clickable treatment

### Chip

- 14px horizontal / 8px vertical reference padding
- 12px radius
- active = solid red + primary text
- inactive = elevated + border + dimmed text

Current Android gap: existing Pulse chips use pill/999dp geometry and therefore do not match frozen design.

### ProgressBar

- track height 6px
- zone track
- red/green/gradient variants
- optional tabular mono label

### Sheet

- bottom sheet
- black 60% scrim
- max content height 85vh reference
- glass-elevated surface
- top border
- 24px top radius
- centered 40x4 drag handle
- optional title/action + close control

### Skeleton

- shimmer block
- 12px radius default
- canonical Slot card skeleton structure

### StatusBadge

Required visual states:

- LIVE = green
- OPEN = blue
- FULL = muted
- APPROVAL = warning

Badge typography: compact mono semibold.

### ErrorState

- centered
- 64x64 critical-tint tile
- 16px radius
- icon 28px
- display-bold 18px title
- dimmed body
- optional secondary-style retry button

### EmptyState

- centered
- 80x80 zone tile
- 24px radius
- display-bold 18px title
- max 240px dimmed subtitle

---

## 4. App shell / BottomNav

Frozen navigation order:

`Pulse · Map · LINK · Fly · Me`

Required:

- glass bottom bar
- top border
- horizontal 16px reference padding
- safe-bottom spacing
- 22px tab icons
- 10px labels
- active red icon/label
- active red-tinted 12px icon background
- active 4px dot
- central LINK button raised above nav
- LINK circle 56x56, red, bold `LINK`
- label `Create`

Current Android gap:

- current navigation is functional but does not yet match exact frozen geometry/states;
- LINK is handled as a normal enum tab internally rather than as the exact frozen central overlay interaction;
- visual parity must be measured before functional restructuring.

---

## 5. Pulse screen

### Header

Required:

- top safe spacing equivalent to reference `pt-14`
- sticky glass header
- bottom border
- location row: red map pin + `Kyiv · Podil`
- City BPM row with red live dot/ring, BPM number and activity label
- notification square 40x40 / radius 12
- unread red count badge
- search input under header

### Filters

Time row:

- Now
- Tonight
- Tomorrow
- All

Category row:

- All
- Social
- Active
- Food

### Content

- Happening Now green-tint banner when applicable
- canonical SlotCard list
- loading = four SlotCard skeletons
- empty = `Quiet around here`
- error = canonical ErrorState
- filtered empty = `No matches`

Current Android gaps:

- no frozen location/BPM header;
- no notification button/badge;
- only one category-filter row;
- no Now/Tonight/Tomorrow/All row;
- no Happening Now banner;
- loading uses spinner rather than SlotCard skeletons;
- error/empty geometry differs;
- SlotCard organizer/avatar/footer structure differs;
- distance/tags/reliability treatment is missing or replaced by server-specific values.

---

## 6. SlotCard / Slot detail family

### SlotCard exact content order

1. 48x48 activity tile
2. status badge + time
3. title
4. location + distance
5. two-line description
6. organizer round avatar + organizer name + reliability
7. progress bar + count
8. tags
9. primary action

Action visuals:

- approval = warning tint/border
- instant = solid red

Current Android gap:

- current card is structurally close but not exact;
- avatar is not canonical;
- distance/tags/reliability are not represented as in frozen design;
- action/state logic currently changes visible structure beyond the frozen reference.

### Global slot detail Sheet from `App.tsx`

Required visual content:

- 56x56 activity tile
- status
- title
- location + distance
- description
- avatar + organizer + reliability
- progress
- tags
- one full-width action

Production-specific host/member controls must later be fitted into a design-compatible detail hierarchy without changing this visual contract during the design-only stage.

---

## 7. Create LINK — 3-step overlay

### Shell

- full-screen background overlay
- header with 36x36 close control
- centered `Create LINK`
- mono `Step X of 3`
- three-segment progress
- scrollable body
- fixed footer

### Step 1

Heading: `What's happening?`

Subtitle: `Pick an activity to get started.`

Activity grid: **16 canonical activities**, four columns:

- Coffee
- Running
- Walk
- Food
- Drinks
- Sport
- Yoga
- Cycling
- Photography
- Music
- Art
- Games
- Chess
- Co-work
- Networking
- Study

Then:

- Title field with counter
- Description field

Current Android gap: only 8 activities and different activity vocabulary.

### Step 2

Heading: `Where & when?`

Required visual controls:

- Location
- Capacity minus/value/plus
- Access level:
  - Instant Join
  - Approval Required
- Visibility:
  - Public
  - Friends Only
  - Private

Current Android gaps:

- only Approval Required is rendered as fixed option;
- only Public is rendered as fixed visibility;
- native date/time field exists in Android but is not part of the original frozen React screen at this exact position; it must not distort the frozen visual layout during design parity.

### Step 3

- Preview heading/subtitle
- elevated preview card
- 56x56 activity tile
- OPEN status
- title/location
- optional description
- capacity/access/visibility summary
- informational zone card

Footer:

- Back when step > 1
- Continue until step 3
- Publish LinkUp on step 3
- disabled Continue visual state

---

## 8. Map screen

Design parity is required **before** Maps SDK/functionality.

Required visual composition:

- full-screen dark map canvas
- grid/road/river/park styling or static design-equivalent placeholder until real SDK binding stage
- top centered `Podil · Kyiv` glass pill
- right control column:
  - Filter
  - Hot
  - Settings
- activity pins with colored 40x40 circles
- joined/capacity mini badge under each pin
- LIVE pulse-ring treatment static frame/visual reserved; actual animation later
- bottom glass summary panel
- green ACTIVE indicator
- active LinkUps count
- `List view >`
- selected-pin visual state
- selected-pin detail bottom sheet
- loading state
- empty state
- error state

Current Android status: **missing**; current Map tab is a capability placeholder.

---

## 9. Fly screen

Required:

- sticky glass header
- red rocket icon
- `Fly Now`
- LIVE count badge
- `Flash drops · Kyiv · right now`
- Joined / Passed / Available counters
- tabs:
  - Now
  - Travel
  - Motion
- FlashCard list
- countdown block
- tags
- progress
- Pass + Join controls
- joined confirmation cards
- all-caught-up state
- loading skeleton state
- empty state
- error state

Current Android status: **missing**; current Fly tab is a capability placeholder.

---

## 10. Me screen

### Shell

- sticky glass header
- title `Me`
- tabs:
  - Profile
  - Passport
  - Settings

Current Android gap: current Me is one functional account screen without canonical three-tab structure.

### Profile tab

Required:

- 80px canonical Avatar
- name / username / city-area
- city badges
- Reliability card
- four metric tiles:
  - Showed up
  - Hosted
  - BUMP verified
  - No-show
- BUMP Vault
- My LinkUps summary card

### Passport tab

Required:

- 3 metric cards: Meetups / Cities / Hosted
- Cities Explored
- Interests chips
- Activity · 6 months chart

### Settings tab

Sections and row order:

Privacy & Safety:

- Privacy Center
- Safety Center
- Guardian + NEW
- Ghost Mode + toggle

Account:

- Notifications
- Language + value
- Accessibility
- Data & Privacy

LinkUp+:

- Upgrade to LinkUp+
- Rewarded Free Day + FREE

App:

- Themes + OLED Dark
- Legal
- Version
- Log out

Current Android gap: current Me exposes edit/profile/block/legal/runtime controls directly instead of the frozen tabbed composition.

---

## 11. Notifications panel

Required Sheet:

- title `Notifications`
- conditional `Mark all read`
- notification types:
  - join green
  - approval warning
  - reminder info
  - bump red
  - host red
  - system dimmed
- 40x40 round icon tile
- unread row red-tint + border
- unread red dot
- title/body/time hierarchy
- empty state `All caught up`
- footer: `No engagement bait. Only events that matter.`

Current Android status: no frozen Notifications panel parity surface yet.

---

## 12. Required design states

Every primary screen must be visually verified in all frozen states where applicable:

- loading
- content
- empty
- error
- selected
- disabled
- unread/read
- active/inactive navigation
- approval/instant action visual
- LIVE/OPEN/FULL/APPROVAL status

`StateToggle.tsx` is the reference for state coverage only. Production Android must use previews/test fixtures rather than shipping that developer control.

---

## 13. Explicit non-goals until design parity is complete

Do not spend the design-only milestone on:

- Google Maps SDK wiring
- Places API
- real map queries
- new database schema for map coordinates
- new FCM behavior
- billing/provider adapters
- realtime/background expansion
- motion/sensor features
- new backend features
- new product screens absent from frozen React/TS
- animation implementation

Existing production code may remain in the repository, but it must not drive redesign decisions.

---

## 14. Current Android parity matrix

| Surface | Current parity estimate | Main design gap |
|---|---:|---|
| Global colors | 100% | none |
| Typography system | 35% | fonts/families not canonicalized |
| Shared primitives | 45% | chips/avatar/sheet/error/empty/skeleton inconsistent |
| Bottom navigation | 65% | exact LINK CTA/nav geometry missing |
| Pulse | 55% | header/time filters/banner/states/card details |
| SlotCard | 70% | avatar/reliability/distance/tags/footer |
| Create LINK | 60% | 16 activities + access/visibility parity |
| Map | 5% | placeholder only |
| Fly | 5% | placeholder only |
| Me Profile | 35% | frozen profile composition absent |
| Me Passport | 0% | absent |
| Me Settings | 25% | frozen grouped rows absent |
| Notifications | 0% | frozen sheet absent |
| Global load/empty/error states | 35% | inconsistent implementations |

These percentages are planning estimates only. `DESIGN PARITY = 100%` is declared only after screen-by-screen visual QA.

---

## 15. Implementation sequence after this checklist

### Phase A — shared Compose design system

1. Typography + font roles.
2. Frozen shapes/spacing constants.
3. Glass/elevated surfaces.
4. Avatar.
5. Button variants/sizes.
6. Chip.
7. ProgressBar.
8. StatusBadge.
9. Card.
10. Sheet.
11. Skeleton / SlotCardSkeleton.
12. EmptyState / ErrorState.

### Phase B — screen parity

1. App shell + BottomNav.
2. Pulse + SlotCard.
3. Create LINK.
4. Map visual-only screen.
5. Fly visual-only screen.
6. Me Profile.
7. Me Passport.
8. Me Settings.
9. Notifications.
10. Global Slot detail visual shell.

### Phase C — visual QA

- compare against frozen source;
- compare against the 35 supplied device screenshots where they represent the current visual family;
- verify no developer/debug copy is visible;
- verify portrait safe areas and keyboard overlap;
- verify no screen is hidden behind BottomNav;
- verify all frozen load/content/empty/error states;
- record each surface as PASS/FAIL.

Only after every frozen surface is PASS may the milestone be marked:

`DESIGN PARITY = 100%`.
