// Mock data for the admin console (the "Helm" design). Mirrors the original
// design's window.DATA. Swap these arrays for real API calls (service-core /
// Logto) when wiring the screens to live data — the screen components only
// depend on the shapes exported here.
import type { BadgeVariant } from './types'

interface Meta {
  label: string
  variant: BadgeVariant
}

export const PLANS: Record<string, Meta> = {
  free: { label: 'Free', variant: 'muted' },
  starter: { label: 'Starter', variant: 'info' },
  growth: { label: 'Growth', variant: 'violet' },
  enterprise: { label: 'Enterprise', variant: 'solid' },
}

export const STATUS: Record<string, Meta> = {
  active: { label: 'Active', variant: 'success' },
  trial: { label: 'Trial', variant: 'warning' },
  past_due: { label: 'Past due', variant: 'danger' },
  suspended: { label: 'Suspended', variant: 'danger' },
  invited: { label: 'Invited', variant: 'info' },
  archived: { label: 'Archived', variant: 'muted' },
}

export const REGIONS: Record<string, string> = {
  'us-east': 'US East',
  'us-west': 'US West',
  'eu-central': 'EU Central',
  'eu-west': 'EU West',
  'ap-south': 'AP South',
}

export interface Org {
  id: string
  name: string
  slug: string
  plan: string
  status: string
  seats: number
  seatsUsed: number
  mrr: number
  region: string
  owner: string
  created: string
  industry: string
}

export const ORGS: Org[] = [
  { id: 'org_8f21', name: 'Northwind Trading', slug: 'northwind', plan: 'enterprise', status: 'active', seats: 248, seatsUsed: 211, mrr: 4900, region: 'us-east', owner: 'Dana Whitfield', created: '2023-02-14', industry: 'Logistics' },
  { id: 'org_7c0a', name: 'Lumen Health', slug: 'lumen-health', plan: 'enterprise', status: 'active', seats: 180, seatsUsed: 164, mrr: 3600, region: 'us-west', owner: 'Marcus Reed', created: '2023-05-09', industry: 'Healthcare' },
  { id: 'org_5b93', name: 'Atlas Robotics', slug: 'atlas-robotics', plan: 'growth', status: 'active', seats: 90, seatsUsed: 77, mrr: 1450, region: 'eu-central', owner: 'Sofia Marchetti', created: '2023-08-21', industry: 'Manufacturing' },
  { id: 'org_3d41', name: 'Cobalt Studios', slug: 'cobalt', plan: 'growth', status: 'trial', seats: 25, seatsUsed: 19, mrr: 0, region: 'us-east', owner: 'Theo Nguyen', created: '2025-05-28', industry: 'Media' },
  { id: 'org_9a17', name: 'Ridgeline Capital', slug: 'ridgeline', plan: 'enterprise', status: 'past_due', seats: 120, seatsUsed: 118, mrr: 3200, region: 'us-east', owner: 'Priya Anand', created: '2022-11-03', industry: 'Finance' },
  { id: 'org_2e88', name: 'Verde Energy', slug: 'verde', plan: 'starter', status: 'active', seats: 40, seatsUsed: 31, mrr: 480, region: 'eu-west', owner: 'Liam O’Connor', created: '2024-01-19', industry: 'Energy' },
  { id: 'org_6f52', name: 'Pinnacle Labs', slug: 'pinnacle', plan: 'growth', status: 'active', seats: 65, seatsUsed: 52, mrr: 1100, region: 'us-west', owner: 'Hana Kim', created: '2024-03-30', industry: 'Biotech' },
  { id: 'org_1b09', name: 'Driftwood Apps', slug: 'driftwood', plan: 'starter', status: 'trial', seats: 15, seatsUsed: 8, mrr: 0, region: 'ap-south', owner: 'Raj Malhotra', created: '2025-06-01', industry: 'Software' },
  { id: 'org_4c70', name: 'Mercer & Vale', slug: 'mercer-vale', plan: 'enterprise', status: 'active', seats: 310, seatsUsed: 295, mrr: 6200, region: 'eu-central', owner: 'Eleanor Vale', created: '2021-09-12', industry: 'Legal' },
  { id: 'org_0d23', name: 'Tidewater Co', slug: 'tidewater', plan: 'free', status: 'suspended', seats: 5, seatsUsed: 5, mrr: 0, region: 'us-east', owner: 'Owen Brooks', created: '2024-10-08', industry: 'Retail' },
  { id: 'org_8e64', name: 'Solis Aerospace', slug: 'solis', plan: 'enterprise', status: 'active', seats: 145, seatsUsed: 122, mrr: 2950, region: 'us-west', owner: 'Camila Torres', created: '2023-12-15', industry: 'Aerospace' },
  { id: 'org_3f19', name: 'Bramble & Co', slug: 'bramble', plan: 'starter', status: 'active', seats: 22, seatsUsed: 17, mrr: 264, region: 'eu-west', owner: 'Felix Hartman', created: '2024-07-22', industry: 'Hospitality' },
]

export interface User {
  id: string
  name: string
  email: string
  org: string
  role: string
  status: string
  lastActive: string
  mfa: boolean
  created: string
}

export const USERS: User[] = [
  { id: 'usr_01', name: 'Dana Whitfield', email: 'dana@northwind.com', org: 'Northwind Trading', role: 'Owner', status: 'active', lastActive: '4m ago', mfa: true, created: '2023-02-14' },
  { id: 'usr_02', name: 'Marcus Reed', email: 'marcus@lumenhealth.io', org: 'Lumen Health', role: 'Owner', status: 'active', lastActive: '22m ago', mfa: true, created: '2023-05-09' },
  { id: 'usr_03', name: 'Sofia Marchetti', email: 's.marchetti@atlasrobotics.eu', org: 'Atlas Robotics', role: 'Admin', status: 'active', lastActive: '1h ago', mfa: true, created: '2023-08-21' },
  { id: 'usr_04', name: 'Theo Nguyen', email: 'theo@cobalt.studio', org: 'Cobalt Studios', role: 'Owner', status: 'active', lastActive: '3h ago', mfa: false, created: '2025-05-28' },
  { id: 'usr_05', name: 'Priya Anand', email: 'priya@ridgelinecap.com', org: 'Ridgeline Capital', role: 'Owner', status: 'active', lastActive: '2d ago', mfa: true, created: '2022-11-03' },
  { id: 'usr_06', name: 'Jonah Klein', email: 'jonah@northwind.com', org: 'Northwind Trading', role: 'Billing Manager', status: 'active', lastActive: '36m ago', mfa: true, created: '2023-03-02' },
  { id: 'usr_07', name: 'Aisha Bello', email: 'aisha@lumenhealth.io', org: 'Lumen Health', role: 'Member', status: 'active', lastActive: '5h ago', mfa: false, created: '2024-01-11' },
  { id: 'usr_08', name: 'Greg Sato', email: 'greg@pinnaclelabs.com', org: 'Pinnacle Labs', role: 'Admin', status: 'invited', lastActive: '—', mfa: false, created: '2025-06-05' },
  { id: 'usr_09', name: 'Camila Torres', email: 'camila@solis.aero', org: 'Solis Aerospace', role: 'Owner', status: 'active', lastActive: '11m ago', mfa: true, created: '2023-12-15' },
  { id: 'usr_10', name: 'Owen Brooks', email: 'owen@tidewater.co', org: 'Tidewater Co', role: 'Owner', status: 'suspended', lastActive: '14d ago', mfa: false, created: '2024-10-08' },
  { id: 'usr_11', name: 'Eleanor Vale', email: 'eleanor@mercervale.com', org: 'Mercer & Vale', role: 'Owner', status: 'active', lastActive: '1h ago', mfa: true, created: '2021-09-12' },
  { id: 'usr_12', name: 'Raj Malhotra', email: 'raj@driftwood.app', org: 'Driftwood Apps', role: 'Owner', status: 'active', lastActive: '6h ago', mfa: false, created: '2025-06-01' },
  { id: 'usr_13', name: 'Nadia Cruz', email: 'nadia@verde.energy', org: 'Verde Energy', role: 'Admin', status: 'active', lastActive: '2h ago', mfa: true, created: '2024-02-04' },
  { id: 'usr_14', name: 'Felix Hartman', email: 'felix@bramble.co', org: 'Bramble & Co', role: 'Owner', status: 'active', lastActive: '40m ago', mfa: false, created: '2024-07-22' },
  { id: 'usr_15', name: 'Lena Brandt', email: 'lena@atlasrobotics.eu', org: 'Atlas Robotics', role: 'Member', status: 'active', lastActive: '8m ago', mfa: true, created: '2024-05-19' },
]

export interface PermissionGroup {
  id: string
  label: string
  perms: { id: string; label: string; desc: string }[]
}

export const PERMISSION_GROUPS: PermissionGroup[] = [
  { id: 'orgs', label: 'Organizations', perms: [
    { id: 'orgs.read', label: 'View organizations', desc: 'See organization profiles and metadata' },
    { id: 'orgs.create', label: 'Create organizations', desc: 'Provision new tenant organizations' },
    { id: 'orgs.update', label: 'Edit organizations', desc: 'Modify name, region, limits and settings' },
    { id: 'orgs.suspend', label: 'Suspend / archive', desc: 'Disable access to an organization' },
  ] },
  { id: 'users', label: 'Users & Teams', perms: [
    { id: 'users.read', label: 'View users', desc: 'Browse member directories' },
    { id: 'users.invite', label: 'Invite users', desc: 'Send seat invitations' },
    { id: 'users.update', label: 'Edit users', desc: 'Change profile and role assignment' },
    { id: 'users.impersonate', label: 'Impersonate', desc: 'Log in as a member for support' },
  ] },
  { id: 'billing', label: 'Billing', perms: [
    { id: 'billing.read', label: 'View billing', desc: 'See plans, invoices and usage' },
    { id: 'billing.manage', label: 'Manage subscriptions', desc: 'Change plans, seats and discounts' },
    { id: 'billing.refund', label: 'Issue refunds', desc: 'Refund invoices and credits' },
  ] },
  { id: 'roles', label: 'Roles & Access', perms: [
    { id: 'roles.read', label: 'View roles', desc: 'Inspect roles and permission sets' },
    { id: 'roles.manage', label: 'Manage roles', desc: 'Create, edit and delete roles' },
  ] },
  { id: 'platform', label: 'Platform', perms: [
    { id: 'flags.manage', label: 'Manage feature flags', desc: 'Toggle and roll out features' },
    { id: 'audit.read', label: 'View audit log', desc: 'Read the immutable activity log' },
    { id: 'settings.manage', label: 'Manage settings', desc: 'Configure global platform settings' },
  ] },
]

export const ALL_PERMS: string[] = PERMISSION_GROUPS.flatMap((g) => g.perms.map((p) => p.id))

export interface Role {
  id: string
  name: string
  scope: 'system' | 'org'
  members: number
  system: boolean
  color: string
  desc: string
  perms: string[]
}

export const SYSTEM_ROLES: Role[] = [
  { id: 'role_super', name: 'Superadmin', scope: 'system', members: 2, system: true, color: '#18181b', desc: 'Unrestricted access to every organization and platform control.', perms: ALL_PERMS },
  { id: 'role_support', name: 'Support Lead', scope: 'system', members: 4, system: true, color: '#2563eb', desc: 'Cross-org read access plus impersonation for customer support.', perms: ['orgs.read', 'users.read', 'users.impersonate', 'billing.read', 'roles.read', 'audit.read'] },
  { id: 'role_billingops', name: 'Billing Ops', scope: 'system', members: 3, system: false, color: '#16a34a', desc: 'Manage subscriptions, invoices and refunds across all tenants.', perms: ['orgs.read', 'users.read', 'billing.read', 'billing.manage', 'billing.refund', 'audit.read'] },
  { id: 'role_readonly', name: 'Read-only Auditor', scope: 'system', members: 1, system: false, color: '#71717a', desc: 'View-only access to all data for compliance review.', perms: ['orgs.read', 'users.read', 'billing.read', 'roles.read', 'audit.read'] },
]

export const ORG_ROLES: Role[] = [
  { id: 'orole_owner', name: 'Owner', scope: 'org', members: 12, system: true, color: '#18181b', desc: 'Full control of a single organization, including billing and deletion.', perms: ['orgs.read', 'orgs.update', 'users.read', 'users.invite', 'users.update', 'billing.read', 'billing.manage', 'roles.read', 'roles.manage'] },
  { id: 'orole_admin', name: 'Admin', scope: 'org', members: 34, system: true, color: '#2563eb', desc: 'Manage members, roles and settings within the organization.', perms: ['orgs.read', 'orgs.update', 'users.read', 'users.invite', 'users.update', 'roles.read', 'roles.manage'] },
  { id: 'orole_billing', name: 'Billing Manager', scope: 'org', members: 19, system: false, color: '#16a34a', desc: 'Access invoices, payment methods and plan changes only.', perms: ['orgs.read', 'billing.read', 'billing.manage'] },
  { id: 'orole_member', name: 'Member', scope: 'org', members: 1840, system: true, color: '#71717a', desc: 'Standard product access with no administrative permissions.', perms: ['orgs.read', 'users.read'] },
  { id: 'orole_guest', name: 'Guest', scope: 'org', members: 96, system: false, color: '#d97706', desc: 'Limited, time-boxed access to shared resources.', perms: ['orgs.read'] },
]

export interface Invoice {
  id: string
  org: string
  amount: number
  status: string
  date: string
  method: string
  plan: string
}

export const INVOICES: Invoice[] = [
  { id: 'INV-20461', org: 'Mercer & Vale', amount: 6200, status: 'active', date: '2026-06-01', method: 'ACH', plan: 'enterprise' },
  { id: 'INV-20460', org: 'Northwind Trading', amount: 4900, status: 'active', date: '2026-06-01', method: 'Visa •4421', plan: 'enterprise' },
  { id: 'INV-20459', org: 'Lumen Health', amount: 3600, status: 'active', date: '2026-06-01', method: 'ACH', plan: 'enterprise' },
  { id: 'INV-20458', org: 'Ridgeline Capital', amount: 3200, status: 'past_due', date: '2026-05-01', method: 'Visa •8890', plan: 'enterprise' },
  { id: 'INV-20457', org: 'Solis Aerospace', amount: 2950, status: 'active', date: '2026-06-01', method: 'Amex •1007', plan: 'enterprise' },
  { id: 'INV-20456', org: 'Atlas Robotics', amount: 1450, status: 'active', date: '2026-06-01', method: 'SEPA', plan: 'growth' },
  { id: 'INV-20455', org: 'Pinnacle Labs', amount: 1100, status: 'active', date: '2026-06-01', method: 'Visa •2231', plan: 'growth' },
  { id: 'INV-20454', org: 'Verde Energy', amount: 480, status: 'active', date: '2026-06-01', method: 'SEPA', plan: 'starter' },
]

export interface PlanPricing {
  plan: string
  price: number | null
  seats: string
  orgs: number
  features: string[]
}

export const PLAN_PRICING: PlanPricing[] = [
  { plan: 'free', price: 0, seats: '5 seats', orgs: 1, features: ['Core product', 'Community support'] },
  { plan: 'starter', price: 12, seats: 'per seat / mo', orgs: 4, features: ['Everything in Free', 'Email support', 'Basic analytics'] },
  { plan: 'growth', price: 22, seats: 'per seat / mo', orgs: 4, features: ['Everything in Starter', 'SSO', 'Advanced roles', 'API access'] },
  { plan: 'enterprise', price: null, seats: 'Custom', orgs: 4, features: ['Everything in Growth', 'SCIM', 'Audit log export', 'Dedicated CSM', '99.99% SLA'] },
]

export interface Flag {
  id: string
  name: string
  desc: string
  enabled: boolean
  rollout: number
  env: string
  updated: string
  owner: string
  kind: string
}

export const FLAGS: Flag[] = [
  { id: 'new_billing_ui', name: 'New billing UI', desc: 'Redesigned invoices & subscription management.', enabled: true, rollout: 100, env: 'all', updated: '2026-06-05', owner: 'Robin Diaz', kind: 'release' },
  { id: 'scim_provisioning', name: 'SCIM provisioning', desc: 'Automated user provisioning for enterprise tenants.', enabled: true, rollout: 60, env: 'prod', updated: '2026-06-03', owner: 'Avery Cole', kind: 'release' },
  { id: 'ai_insights', name: 'AI usage insights', desc: 'Anomaly detection on org usage patterns.', enabled: true, rollout: 25, env: 'prod', updated: '2026-06-07', owner: 'Sam Park', kind: 'experiment' },
  { id: 'org_data_residency', name: 'Data residency controls', desc: 'Per-org region pinning for stored data.', enabled: false, rollout: 0, env: 'staging', updated: '2026-05-29', owner: 'Avery Cole', kind: 'release' },
  { id: 'granular_roles', name: 'Granular custom roles', desc: 'Allow tenants to define custom permission sets.', enabled: true, rollout: 40, env: 'prod', updated: '2026-06-06', owner: 'Avery Cole', kind: 'release' },
  { id: 'usage_alerts', name: 'Seat overage alerts', desc: 'Notify owners when nearing seat limits.', enabled: true, rollout: 100, env: 'all', updated: '2026-05-20', owner: 'Robin Diaz', kind: 'release' },
  { id: 'legacy_export', name: 'Legacy CSV export', desc: 'Deprecated export path kept for two tenants.', enabled: false, rollout: 0, env: 'prod', updated: '2026-04-11', owner: 'Sam Park', kind: 'ops' },
]

export interface AuditEntry {
  id: string
  actor: string
  action: string
  target: string
  cat: string
  org: string
  ip: string
  time: string
  ago: string
}

export const AUDIT: AuditEntry[] = [
  { id: 'a1', actor: 'Avery Cole', action: 'flag.enabled', target: 'ai_insights', cat: 'platform', org: '—', ip: '10.2.4.19', time: 'Today, 09:41', ago: '12m ago' },
  { id: 'a2', actor: 'Robin Diaz', action: 'billing.refund', target: 'INV-20431 · $1,100', cat: 'billing', org: 'Pinnacle Labs', ip: '10.2.4.51', time: 'Today, 09:18', ago: '35m ago' },
  { id: 'a3', actor: 'Sam Park', action: 'user.impersonate.start', target: 'theo@cobalt.studio', cat: 'security', org: 'Cobalt Studios', ip: '10.2.4.33', time: 'Today, 08:55', ago: '58m ago' },
  { id: 'a4', actor: 'Avery Cole', action: 'org.created', target: 'Driftwood Apps', cat: 'orgs', org: 'Driftwood Apps', ip: '10.2.4.19', time: 'Today, 08:30', ago: '1h ago' },
  { id: 'a5', actor: 'System', action: 'org.suspended', target: 'Tidewater Co · payment failed', cat: 'orgs', org: 'Tidewater Co', ip: 'system', time: 'Today, 07:02', ago: '3h ago' },
  { id: 'a6', actor: 'Robin Diaz', action: 'subscription.upgraded', target: 'growth → enterprise', cat: 'billing', org: 'Solis Aerospace', ip: '10.2.4.51', time: 'Yesterday, 17:44', ago: '16h ago' },
  { id: 'a7', actor: 'Avery Cole', action: 'role.updated', target: 'Billing Ops · +billing.refund', cat: 'security', org: '—', ip: '10.2.4.19', time: 'Yesterday, 15:20', ago: '18h ago' },
  { id: 'a8', actor: 'Sam Park', action: 'user.suspended', target: 'owen@tidewater.co', cat: 'security', org: 'Tidewater Co', ip: '10.2.4.33', time: 'Yesterday, 11:09', ago: '22h ago' },
  { id: 'a9', actor: 'Avery Cole', action: 'flag.rollout.changed', target: 'scim_provisioning · 40% → 60%', cat: 'platform', org: '—', ip: '10.2.4.19', time: 'Jun 6, 14:02', ago: '2d ago' },
  { id: 'a10', actor: 'Robin Diaz', action: 'invoice.sent', target: 'INV-20458 · $3,200', cat: 'billing', org: 'Ridgeline Capital', ip: '10.2.4.51', time: 'Jun 5, 10:30', ago: '3d ago' },
  { id: 'a11', actor: 'System', action: 'login.failed', target: 'owen@tidewater.co · 5 attempts', cat: 'security', org: 'Tidewater Co', ip: '203.0.113.7', time: 'Jun 5, 09:12', ago: '3d ago' },
  { id: 'a12', actor: 'Avery Cole', action: 'org.updated', target: 'Lumen Health · region us-west', cat: 'orgs', org: 'Lumen Health', ip: '10.2.4.19', time: 'Jun 4, 16:48', ago: '4d ago' },
]

export const AUDIT_CAT: Record<string, BadgeVariant> = {
  orgs: 'info',
  billing: 'success',
  security: 'danger',
  platform: 'violet',
}

export interface Kpi {
  id: string
  label: string
  value: string
  delta: string
  dir: 'up' | 'down'
  icon: string
  tone: string
  sub: string
}

export const KPIS: Kpi[] = [
  { id: 'orgs', label: 'Organizations', value: '1,284', delta: '+24', dir: 'up', icon: 'building-2', tone: 'primary', sub: 'this month' },
  { id: 'mrr', label: 'MRR', value: '$486.2K', delta: '+8.4%', dir: 'up', icon: 'credit-card', tone: 'success', sub: 'vs last month' },
  { id: 'users', label: 'Active users', value: '38,902', delta: '+5.1%', dir: 'up', icon: 'users', tone: 'info', sub: 'last 30 days' },
  { id: 'seats', label: 'Seat utilization', value: '86.4%', delta: '-1.2%', dir: 'down', icon: 'gauge', tone: 'warning', sub: 'across all plans' },
]

export const TREND = [
  { w: 'W1', signups: 42, churn: 8 }, { w: 'W2', signups: 51, churn: 11 }, { w: 'W3', signups: 38, churn: 7 },
  { w: 'W4', signups: 64, churn: 9 }, { w: 'W5', signups: 58, churn: 13 }, { w: 'W6', signups: 72, churn: 10 },
  { w: 'W7', signups: 69, churn: 12 }, { w: 'W8', signups: 81, churn: 9 }, { w: 'W9', signups: 77, churn: 14 },
  { w: 'W10', signups: 92, churn: 11 }, { w: 'W11', signups: 88, churn: 8 }, { w: 'W12', signups: 104, churn: 12 },
]

export const PLAN_MIX = [
  { plan: 'enterprise', count: 312, color: '#18181b' },
  { plan: 'growth', count: 428, color: '#7c3aed' },
  { plan: 'starter', count: 396, color: '#2563eb' },
  { plan: 'free', count: 148, color: '#a1a1aa' },
]

export function fmtMoney(n: number): string {
  return n === 0 ? '$0' : '$' + n.toLocaleString('en-US')
}
