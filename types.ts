export type Screen = 'pulse' | 'map' | 'create' | 'fly' | 'me';

export type LoadState = 'loading' | 'content' | 'empty' | 'error';

export type ActivityType =
  | 'coffee'
  | 'running'
  | 'walk'
  | 'food'
  | 'drinks'
  | 'sport'
  | 'yoga'
  | 'cycling'
  | 'photography'
  | 'music'
  | 'art'
  | 'games'
  | 'chess'
  | 'cowork'
  | 'networking'
  | 'study';

export type SlotStatus = 'OPEN' | 'LIVE' | 'FULL' | 'APPROVAL';

export type AccessLevel = 'instant' | 'approval';

export type Visibility = 'public' | 'friends' | 'private';

export type TimeFilter = 'now' | 'tonight' | 'tomorrow' | 'all';

export type CategoryFilter = 'all' | 'social' | 'active' | 'food';

export type FlyTab = 'now' | 'travel' | 'motion';

export type MeTab = 'profile' | 'passport' | 'settings';

export interface ActivityMeta {
  type: ActivityType;
  emoji: string;
  label: string;
  category: CategoryFilter;
}

export interface Organizer {
  id: string;
  name: string;
  username: string;
  avatarColor: string;
  initials: string;
  reliability: number;
}

export interface Slot {
  id: string;
  activity: ActivityType;
  status: SlotStatus;
  title: string;
  description: string;
  location: string;
  area: string;
  timeLabel: string;
  distanceKm: number;
  organizer: Organizer;
  capacity: number;
  joined: number;
  tags: string[];
  accessLevel: AccessLevel;
  happeningNow: boolean;
  startsInMinutes: number;
}

export interface MapPin {
  id: string;
  slotId: string;
  x: number;
  y: number;
  activity: ActivityType;
  joined: number;
  capacity: number;
  status: SlotStatus;
}

export interface FlashDrop {
  id: string;
  title: string;
  emoji: string;
  location: string;
  distanceKm: number;
  tags: string[];
  secondsLeft: number;
  joined: number;
  capacity: number;
}

export interface NotificationItem {
  id: string;
  type: 'join' | 'approval' | 'reminder' | 'bump' | 'system' | 'host';
  title: string;
  description: string;
  time: string;
  read: boolean;
}

export interface CityBadge {
  city: string;
  country: string;
  flag: string;
  visits: number;
}

export interface BumpRecord {
  id: string;
  name: string;
  event: string;
  date: string;
  verified: boolean;
}

export interface MonthActivity {
  month: string;
  value: number;
}
