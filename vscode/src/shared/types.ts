// Shared shapes mirroring agilemarkdown MCP responses. Keep these in
// sync with the Go server (mcpserver/server.go and friends).

export type StoryType = 'feature' | 'bug' | 'chore' | 'release' | string;
export type StoryStatus =
  | 'unstarted'
  | 'started'
  | 'finished'
  | 'delivered'
  | 'accepted'
  | 'rejected';

export interface ItemSummary {
  path: string;
  title: string;
  status: StoryStatus;
  type?: StoryType;
  assigned?: string;
  assignees?: string[];
  estimate?: string;
  tags?: string[];
  blocked?: boolean;
  comment_count?: number;
  epic?: string;
}

export interface OrderRow {
  index: number;
  title: string;
  path: string;
  status?: StoryStatus;
  estimate?: string;
  type?: StoryType;
  assignees?: string[];
  tags?: string[];
  blocked?: boolean;
  comment_count?: number;
  epic?: string;
}

export interface PriorityListResult {
  backlog: string;
  items: OrderRow[];
  count: number;
  velocity: number;
}

export interface IceboxListResult {
  backlog: string;
  items: OrderRow[];
  count: number;
}

export interface ListItemsResult {
  items: ItemSummary[];
  count: number;
}

export interface DashboardResult {
  velocity: number;
  velocity_bootstrap: boolean;
  volatility_percent: number;
  cycle_time_median_hours: number;
  rejection_rate_latest_percent: number;
  stories_accepted_total: number;
}

export interface VelocityHistoryRow {
  iteration: number;
  start: string;
  planned: number;
  accepted: number;
  length_weeks: number;
  team_strength: number;
}

export interface VelocityHistoryResult {
  rows: VelocityHistoryRow[];
}

export interface BurnupRow {
  day: string;
  scope: number;
  done: number;
}

export interface BurnupResult {
  iteration: number;
  start: string;
  end: string;
  rows: BurnupRow[];
}

export interface CFDRow {
  day: string;
  accepted: number;
  in_flight: number;
  backlog: number;
}

export interface CFDResult {
  rows: CFDRow[];
}

export interface HistoryEntry {
  hash: string;
  author: string;
  email?: string;
  when: string;
  subject: string;
}

export interface HistoryResult {
  entries: HistoryEntry[];
  count: number;
}

export interface SearchHit {
  path: string;
  title: string;
  status?: string;
  type?: string;
  snippet?: string;
  score: number;
}

export interface SearchResult {
  query: string;
  hits: SearchHit[];
  count: number;
}

export interface TypeMixRow {
  type: string;
  count: number;
  percent: number;
}

export interface TypeMixResult {
  rows: TypeMixRow[];
  total: number;
}

export interface CommentRow {
  author?: string;
  users?: string[];
  when?: string;
  text: string;
}

export interface GetCommentsResult {
  comments: CommentRow[];
  count: number;
}

export interface TaskRow {
  index: number;
  done: boolean;
  text: string;
}

export interface ListTasksResult {
  tasks: TaskRow[];
  count: number;
}

export interface NextItemResult {
  found: boolean;
  backlog?: string;
  path?: string;
  title?: string;
  status?: StoryStatus;
  type?: StoryType;
}

export interface EpicProgressResult {
  slug: string;
  total_stories: number;
  accepted_stories: number;
  total_points: number;
  accepted_points: number;
  percent_done: number;
  ascii: string;
}

export interface GetItemResult {
  path: string;
  title: string;
  status: StoryStatus;
  type?: StoryType;
  assigned?: string;
  assignees?: string[];
  estimate?: string;
  tags?: string[];
  blocked?: boolean;
  blocked_reason?: string;
  epic?: string;
  author?: string;
  started?: string;
  finished?: string;
  delivered?: string;
  accepted?: string;
  iteration?: number;
  iteration_label?: string;
  body: string;
}

// Webview <-> extension protocol envelopes.
export interface InboundMessage {
  type: string;
  payload?: any;
}

export interface OutboundMessage {
  type: string;
  payload?: any;
}
