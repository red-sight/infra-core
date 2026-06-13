<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Icon from '@/components/ui/Icon.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Avatar from '@/components/ui/Avatar.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import Field from '@/components/ui/Field.vue'
import SearchBox from '@/components/ui/SearchBox.vue'
import Dropdown from '@/components/ui/Dropdown.vue'
import MenuItem from '@/components/ui/MenuItem.vue'
import MenuSep from '@/components/ui/MenuSep.vue'
import Dialog from '@/components/ui/Dialog.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import IconChip from '@/components/ui/IconChip.vue'
import { useToast } from '@/composables/useToast'
import { ORGS, STATUS, USERS, type User } from '@/lib/data'
import type { BadgeVariant } from '@/lib/types'

const toast = useToast()

const ROLE_VARIANT: Record<string, BadgeVariant> = {
  Owner: 'solid',
  Admin: 'info',
  'Billing Manager': 'success',
  Member: 'muted',
  Guest: 'warning',
}
const ORG_OPTS = [{ value: '', label: 'All organizations' }, ...[...new Set(USERS.map((u) => u.org))].map((o) => ({ value: o, label: o }))]
const ROLE_OPTS = [{ value: '', label: 'All roles' }, ...[...new Set(USERS.map((u) => u.role))].map((r) => ({ value: r, label: r }))]

const users = ref<User[]>([...USERS])
const q = ref('')
const org = ref('')
const role = ref('')
const inviting = ref(false)

const filtered = computed(() =>
  users.value.filter(
    (u) =>
      (!q.value || u.name.toLowerCase().includes(q.value.toLowerCase()) || u.email.toLowerCase().includes(q.value.toLowerCase())) &&
      (!org.value || u.org === org.value) &&
      (!role.value || u.role === role.value),
  ),
)

const stats = computed(() => [
  { label: 'Total users', value: String(users.value.length), icon: 'users', tone: 'primary' },
  { label: 'Active', value: String(users.value.filter((u) => u.status === 'active').length), icon: 'user-check', tone: 'success' },
  { label: 'Pending invites', value: String(users.value.filter((u) => u.status === 'invited').length), icon: 'mail', tone: 'info' },
  { label: 'MFA enabled', value: Math.round((users.value.filter((u) => u.mfa).length / users.value.length) * 100) + '%', icon: 'shield-check', tone: 'violet' },
])

/* invite dialog */
const email = ref('')
const inviteOrg = ref(ORGS[0].name)
const inviteRole = ref('Member')
const touched = ref(false)
const emailErr = computed(() => (touched.value && !/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(email.value) ? 'Enter a valid email' : null))

watch(inviting, (open) => {
  if (open) {
    email.value = ''
    inviteRole.value = 'Member'
    touched.value = false
  }
})

function submitInvite() {
  touched.value = true
  if (emailErr.value || !email.value) return
  const name = email.value
    .split('@')[0]
    .split('.')
    .map((s) => s[0].toUpperCase() + s.slice(1))
    .join(' ')
  users.value = [
    { id: 'usr_' + Math.random().toString(16).slice(2, 5), name, email: email.value, org: inviteOrg.value, role: inviteRole.value, status: 'invited', lastActive: '—', mfa: false, created: '2026-06-08' },
    ...users.value,
  ]
  inviting.value = false
  toast({ title: 'Invitation sent', desc: email.value })
}

function suspend(u: User) {
  const next = u.status === 'suspended' ? 'active' : 'suspended'
  users.value = users.value.map((x) => (x.id === u.id ? { ...x, status: next } : x))
  toast({
    title: u.status === 'suspended' ? 'User reactivated' : 'User suspended',
    desc: u.email,
    variant: u.status === 'suspended' ? 'success' : 'error',
  })
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1 class="page-title">Users &amp; Teams</h1>
        <p class="page-sub">Every member across all tenant organizations.</p>
      </div>
      <div class="page-actions">
        <Button variant="primary" icon="user-plus" @click="inviting = true">Invite user</Button>
      </div>
    </div>

    <div class="section-stack">
      <div class="grid-kpi">
        <Card v-for="s in stats" :key="s.label" pad style="display: flex; align-items: center; gap: 13px">
          <IconChip :name="s.icon" :tone="s.tone" :size="40" :icon-size="18" />
          <div>
            <div class="kpi__label">{{ s.label }}</div>
            <div style="font-size: 22px; font-weight: 700" class="tnum">{{ s.value }}</div>
          </div>
        </Card>
      </div>

      <Card>
        <div class="toolbar" style="padding: 14px; display: flex; gap: 9px; flex-wrap: wrap; border-bottom: 1px solid var(--border)">
          <div style="flex: 1 1 240px; min-width: 200px; display: flex"><SearchBox v-model="q" placeholder="Search by name or email…" /></div>
          <Select v-model="org" :options="ORG_OPTS" filter />
          <Select v-model="role" :options="ROLE_OPTS" filter />
        </div>

        <EmptyState v-if="filtered.length === 0" icon="users" title="No users found" desc="Try a different search or filter." />

        <div v-else class="table-wrap">
          <table class="table">
            <thead>
              <tr><th>User</th><th>Organization</th><th>Role</th><th>Status</th><th>MFA</th><th>Last active</th><th style="width: 44px"></th></tr>
            </thead>
            <tbody>
              <tr v-for="u in filtered" :key="u.id">
                <td>
                  <div class="idcell">
                    <Avatar :name="u.name" size="sm" />
                    <div class="idcell__main"><div class="idcell__name">{{ u.name }}</div><div class="idcell__sub">{{ u.email }}</div></div>
                  </div>
                </td>
                <td><span class="cell-muted">{{ u.org }}</span></td>
                <td><Badge :variant="ROLE_VARIANT[u.role] || 'muted'">{{ u.role }}</Badge></td>
                <td><Badge :variant="STATUS[u.status].variant" dot>{{ STATUS[u.status].label }}</Badge></td>
                <td>
                  <span v-if="u.mfa" class="dotline" style="color: var(--success-fg); font-size: 12.5px; font-weight: 500"><Icon name="shield-check" :size="14" /> On</span>
                  <span v-else class="cell-muted dotline" style="font-size: 12.5px"><Icon name="shield" :size="14" /> Off</span>
                </td>
                <td><span class="cell-muted tnum">{{ u.lastActive }}</span></td>
                <td>
                  <Dropdown align="end">
                    <template #trigger>
                      <button class="btn btn--ghost btn--icon btn--sm row-actions"><Icon name="more-horizontal" :size="16" /></button>
                    </template>
                    <MenuItem icon="user">View profile</MenuItem>
                    <MenuItem icon="pencil">Edit role</MenuItem>
                    <MenuItem icon="log-in" @click="toast({ title: 'Impersonation started', desc: u.email, variant: 'info' })">Impersonate</MenuItem>
                    <MenuSep />
                    <MenuItem :icon="u.status === 'suspended' ? 'user-check' : 'user-x'" :danger="u.status !== 'suspended'" @click="suspend(u)">
                      {{ u.status === 'suspended' ? 'Reactivate' : 'Suspend' }}
                    </MenuItem>
                  </Dropdown>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div style="padding: 12px 16px; border-top: 1px solid var(--border); font-size: 12.5px; color: var(--muted-foreground)">
          Showing {{ filtered.length }} of {{ users.length }} users
        </div>
      </Card>
    </div>

    <Dialog :open="inviting" :width="500" title="Invite user" desc="Send a seat invitation to join an organization." @close="inviting = false">
      <div style="display: flex; flex-direction: column; gap: 16px">
        <Field label="Email address" required :error="emailErr">
          <Input icon="mail" type="email" placeholder="person@company.com" v-model="email" :error="emailErr" autofocus />
        </Field>
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 14px">
          <Field label="Organization"><Select v-model="inviteOrg" :options="ORGS.map((o) => o.name)" /></Field>
          <Field label="Role"><Select v-model="inviteRole" :options="['Owner', 'Admin', 'Billing Manager', 'Member', 'Guest']" /></Field>
        </div>
      </div>
      <template #footer>
        <Button variant="ghost" @click="inviting = false">Cancel</Button>
        <Button variant="primary" icon="send" @click="submitInvite">Send invite</Button>
      </template>
    </Dialog>
  </div>
</template>
