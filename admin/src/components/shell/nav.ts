// Sidebar navigation model + the screen-id → route-path map. Route names equal
// screen ids so the active link can be derived from the current route name.
export interface NavItem {
  id: string
  label: string
  icon: string
  group: string
}

export const NAV: NavItem[] = [
  { id: 'overview', label: 'Overview', icon: 'layout-dashboard', group: 'Platform' },
  { id: 'organizations', label: 'Organizations', icon: 'building-2', group: 'Manage' },
  { id: 'users', label: 'Users', icon: 'users', group: 'Manage' },
  { id: 'roles', label: 'Roles & Permissions', icon: 'shield-check', group: 'Manage' },
  { id: 'billing', label: 'Billing', icon: 'credit-card', group: 'Manage' },
  { id: 'audit', label: 'Audit Log', icon: 'scroll-text', group: 'Operate' },
  { id: 'flags', label: 'Feature Flags', icon: 'flag', group: 'Operate' },
]

export const NAV_GROUPS = ['Platform', 'Manage', 'Operate']

export const SCREEN_PATHS: Record<string, string> = {
  overview: '/',
  organizations: '/organizations',
  users: '/users',
  roles: '/roles',
  billing: '/billing',
  audit: '/audit',
  flags: '/flags',
}
