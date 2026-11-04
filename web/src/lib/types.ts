export type Status = "checking" | "connected" | "degraded" | "disconnected";
export type Diagnosis = "checking" | "ok" | "down" | "dns" | "http" | "partial" | "lan" | "isp";
export type Layer = "tcp" | "http" | "dns";
export type UpdateStatus = "idle" | "downloading" | "restarting" | "error";

export interface TargetResult {
  layer: Layer;
  target: string;
  ok: boolean;
  latency_ms: number;
}

export interface EventEntry {
  time: string;
  ts: number;
  event_type: Exclude<Status, "checking">;
  duration_seconds: number;
  tcp_failed: boolean;
  http_failed: boolean;
  dns_failed: boolean;
  packet_loss_pct: number;
  latency_ms: number;
}

export interface TickEntry {
  time: string;
  ts: number;
  status: Status;
  latency_ms: number;
  packet_loss_pct: number;
  tcp_ok: boolean;
  http_ok: boolean;
  dns_ok: boolean;
}

export interface Availability {
  today: number | null;
  week: number | null;
  month: number | null;
}

export interface UpdateInfo {
  has_update: boolean;
  latest_version: string;
  current_version: string;
  release_notes: string;
}

export interface Snapshot {
  type: string;
  boot_id: string;
  version: string;
  server_time: number;
  config_revision: number;
  config_error?: string;
  language: string;
  check_interval_sec: number;
  status: Status;
  status_since: number;
  last_check: number;
  latency_ms: number;
  avg_latency_ms: number;
  packet_loss: number;
  tcp_ping_ok: boolean;
  http_ok: boolean;
  dns_ok: boolean;
  diagnosis: Diagnosis;
  gateway?: string;
  total_checks: number;
  up_checks: number;
  disconnections: number;
  uptime_seconds: number;
  uptime_pct: number;
  jitter_ms: number;
  latency_history: number[] | null;
  targets: TargetResult[] | null;
  events: EventEntry[] | null;
  ticks: TickEntry[] | null;
  availability?: Availability | null;
  update_info?: UpdateInfo | null;
  update_status: UpdateStatus;
  update_error?: string;
  system_notifs: boolean;
  speed_test_running: boolean;
}

export interface SpeedTestResult {
  timestamp: string;
  download_mbps: number;
  upload_mbps: number | null;
  latency_ms: number;
  duration_seconds: number;
  parallel_connections: number;
  triggered_by: "user" | "schedule";
}

export interface SpeedProgress {
  type: "speed_test_progress";
  phase?: "ping" | "download" | "upload" | "done" | "error" | "cancelled";
  done: boolean;
  cancelled?: boolean;
  error?: string;
  current_mbps?: number;
  elapsed_seconds?: number;
  total_seconds?: number;
  latency_ms?: number;
  download_mbps?: number;
  result?: SpeedTestResult;
}

export interface SpeedTestConfig {
  download_targets: string[];
  parallel_connections: number;
  timeout_seconds: number;
  upload_target: string;
  alert_threshold_mbps: number;
  schedule_minutes: number;
}

export type NotifyEvent = "down" | "degraded" | "restored" | "speed" | "weekly";
export type NotifyChannel = "desktop" | "webhook" | "telegram" | "ntfy";
export type QuietMode = "sound" | "all";

export type ChannelSwitches = Record<NotifyChannel, boolean>;

export interface NotificationSettings {
  sound: { enabled: boolean; volume: number };
  quiet_hours: { enabled: boolean; start: string; end: string; days: number[]; mode: QuietMode };
  events: Record<NotifyEvent, ChannelSwitches>;
  min_outage_seconds: number;
  reminder_minutes: number;
}

export type AlertKind = "down" | "degraded" | "restored" | "reminder" | "speed" | "test" | "weekly";
export type AlertChannel = "desktop" | "sound" | "webhook" | "telegram" | "ntfy";
export type MuteReason = "disabled" | "quiet_hours" | "sound_off" | "short_outage" | "startup";

export interface NotificationRecord {
  id: string;
  time: string;
  kind: AlertKind;
  reason?: {
    tcp_ping_failed: boolean;
    http_failed: boolean;
    dns_failed: boolean;
    packet_loss_pct: number;
    avg_latency_ms: number;
  };
  duration_seconds?: number;
  weekly?: WeeklySummary;
  speed?: {
    download_mbps: number;
    upload_mbps?: number;
    latency_ms: number;
    threshold_mbps?: number;
    below?: boolean;
  };
  sent: AlertChannel[];
  muted?: Partial<Record<AlertChannel, MuteReason>>;
}

export interface NotificationList {
  items: NotificationRecord[];
  unread: number;
  last_read_at: string | null;
}

export interface Config {
  check_interval_sec: number;
  ping_targets: string[];
  http_targets: string[];
  dns_targets: string[];
  fail_threshold: number;
  packet_loss_threshold: number;
  latency_threshold_ms: number;
  log_dir: string;
  webhook_url: string;
  telegram_bot_token: string;
  telegram_chat_id: string;
  ntfy_url: string;
  ntfy_token: string;
  dashboard_port: number;
  use_icmp: boolean;
  auto_update_check: boolean;
  language: string;
  speed_test: SpeedTestConfig;
  notifications: NotificationSettings;
}

export interface LogEvent {
  timestamp: string;
  event: Exclude<Status, "checking">;
  duration_seconds?: number;
  reason: {
    tcp_ping_failed: boolean;
    http_failed: boolean;
    dns_failed: boolean;
    packet_loss_pct: number;
    avg_latency_ms: number;
  };
}

export interface TestTargetResult {
  target: string;
  ok: boolean;
  latency_ms: number;
  error?: string;
}

export interface TestTargetsResponse {
  ping_targets: TestTargetResult[];
  http_targets: TestTargetResult[];
  dns_targets: TestTargetResult[];
}

export interface CauseStat {
  outages: number;
  downtime_seconds: number;
  pct: number;
}

export interface ReportEventRow {
  time: string;
  type: string;
  duration_seconds: number;
  cause: string;
  after_gap?: boolean;
}

export interface ReportDay {
  date: string;
  downtime_seconds: number;
  outages: number;
  worst_outage_seconds: number;
  avg_latency_ms: number;
  avg_jitter_ms: number;
  avg_loss_pct: number;
  uptime_pct: number;
}

export interface MonthlyReport {
  month: string;
  generated_at: string;
  monitored_seconds: number;
  total_downtime_seconds: number;
  total_degraded_seconds: number;
  uptime_pct: number;
  disconnections: number;
  degraded_episodes: number;
  longest_outage_seconds: number;
  longest_outage_at: string;
  carried_over_seconds: number;
  mttr_seconds: number;
  avg_jitter_ms: number;
  causes: Record<string, CauseStat>;
  outage_types: Record<string, number>;
  events: ReportEventRow[];
  days: ReportDay[];
  coverage_start?: string;
  window_seconds: number;
  coverage_pct: number;
  unmonitored_seconds: number;
  gaps: ReportGap[];
  speed: ReportSpeed;
  profile: CaseProfile;
  complaints: Complaint[];
}

export interface ReportGap {
  start: string;
  end: string;
  duration_seconds: number;
  kind: "off" | "power";
}

export interface ReportSpeed {
  tests: number;
  avg_download_mbps: number;
  median_download_mbps: number;
  min_download_mbps: number;
  max_download_mbps: number;
  avg_upload_mbps: number;
  avg_latency_ms: number;
  plan_download_mbps: number;
  pct_of_plan: number;
  below_half_plan: number;
}

export interface CaseProfile {
  country: "EG" | "SA" | "GB" | "US" | "";
  response_hours: number;
  provider: string;
  plan_name: string;
  plan_download_mbps: number;
  plan_upload_mbps: number;
  line_number: string;
  account_number: string;
}

export type ComplaintStatus = "open" | "escalated" | "resolved" | "rejected";

export interface Complaint {
  id: string;
  provider: string;
  ticket: string;
  opened_at: string;
  channel: string;
  agent: string;
  status: ComplaintStatus;
  escalated_at?: string;
  regulator_ticket?: string;
  closed_at?: string;
  notes: string;
}

export interface CaseFile {
  profile: CaseProfile;
  complaints: Complaint[];
}

export interface ConfigView extends Config {
  secret_set: { webhook_url: boolean; telegram_bot_token: boolean; ntfy_url?: boolean; ntfy_token?: boolean };
}

export interface ConfigSaveResponse {
  ok: true;
  revision: number;
  restart_required: boolean;
  config: ConfigView;
}

export interface StartupState {
  supported: boolean;
  enabled: boolean;
}

export interface SpeedTestEvent extends SpeedTestResult {
  event: "speed_test";
  endpoints: string[];
}

export type RangeSummary = Omit<MonthlyReport, "speed" | "profile" | "complaints">;

export interface RangeEvidence {
  from: string;
  to: string;
  summary: RangeSummary;
  speed: ReportSpeed;
}

export interface WeeklySummary {
  from: string;
  to: string;
  outages: number;
  downtime_seconds: number;
  longest_seconds: number;
  uptime_pct: number;
  coverage_pct: number;
  speed_tests: number;
  median_mbps: number;
  plan_mbps: number;
  pct_of_plan: number;
  prev_outages?: number;
}

export type TimelineStatus = "connected" | "degraded" | "disconnected" | "off";

export interface TimelineSegment {
  start: string;
  end: string;
  status: TimelineStatus;
}

export interface Timeline {
  from: string;
  to: string;
  segments: TimelineSegment[];
}
