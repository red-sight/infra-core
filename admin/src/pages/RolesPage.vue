<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import Icon from '@/components/ui/Icon.vue'
import Card from '@/components/ui/Card.vue'
import CardHead from '@/components/ui/CardHead.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Input from '@/components/ui/Input.vue'
import Textarea from '@/components/ui/Textarea.vue'
import Field from '@/components/ui/Field.vue'
import Segmented from '@/components/ui/Segmented.vue'
import Dialog from '@/components/ui/Dialog.vue'
import Switch from '@/components/ui/Switch.vue'
import { useToast } from '@/composables/useToast'
import { ALL_PERMS, ORG_ROLES, PERMISSION_GROUPS, SYSTEM_ROLES, type Role } from '@/lib/data'

const toast = useToast()
const SWATCHES = ['#18181b', '#2563eb', '#16a34a', '#d97706', '#7c3aed', '#dc2626', '#0891b2', '#db2777']

function groupIcon(id: string) {
  return id === 'orgs' ? 'building-2' : id === 'users' ? 'users' : id === 'billing' ? 'credit-card' : id === 'roles' ? 'shield-check' : 'settings'
}

const scope = ref<'system' | 'org'>('system')
const sysRoles = ref<Role[]>(SYSTEM_ROLES.map((r) => ({ ...r })))
const orgRoles = ref<Role[]>(ORG_ROLES.map((r) => ({ ...r })))
const selId = ref(SYSTEM_ROLES[0].id)
const editing = ref(false)
const creating = ref(false)

const rolesRef = computed(() => (scope.value === 'system' ? sysRoles : orgRoles))
const roles = computed(() => rolesRef.value.value)
const selected = computed(() => roles.value.find((r) => r.id === selId.value) ?? roles.value[0])

watch(scope, () => {
  selId.value = roles.value[0].id
  editing.value = false
})

function togglePerm(pid: string) {
  if (selected.value.system) return
  rolesRef.value.value = roles.value.map((r) =>
    r.id === selected.value.id
      ? { ...r, perms: r.perms.includes(pid) ? r.perms.filter((x) => x !== pid) : [...r.perms, pid] }
      : r,
  )
}

function grantedCount(groupPerms: { id: string }[]) {
  return groupPerms.filter((p) => selected.value.perms.includes(p.id)).length
}

/* create-role dialog */
const name = ref('')
const desc = ref('')
const color = ref(SWATCHES[1])
const perms = reactive<string[]>([])
const touched = ref(false)
const nameErr = computed(() => (touched.value && !name.value.trim() ? 'Role name is required' : null))

watch(creating, (open) => {
  if (open) {
    name.value = ''
    desc.value = ''
    color.value = SWATCHES[1]
    perms.splice(0, perms.length)
    touched.value = false
  }
})

function toggleCreatePerm(id: string) {
  const i = perms.indexOf(id)
  if (i === -1) perms.push(id)
  else perms.splice(i, 1)
}
function groupState(g: (typeof PERMISSION_GROUPS)[number]) {
  const ids = g.perms.map((p) => p.id)
  const all = ids.every((i) => perms.includes(i))
  const some = ids.some((i) => perms.includes(i))
  return { ids, all, some, count: ids.filter((i) => perms.includes(i)).length }
}
function toggleGroup(g: (typeof PERMISSION_GROUPS)[number]) {
  const { ids, all } = groupState(g)
  if (all) {
    for (const id of ids) {
      const i = perms.indexOf(id)
      if (i !== -1) perms.splice(i, 1)
    }
  } else {
    for (const id of ids) if (!perms.includes(id)) perms.push(id)
  }
}

function submitCreate() {
  touched.value = true
  if (!name.value.trim()) return
  const id = 'role_' + Math.random().toString(16).slice(2, 6)
  const role: Role = { id, name: name.value, scope: scope.value, members: 0, system: false, color: color.value, desc: desc.value || 'Custom role.', perms: [...perms] }
  rolesRef.value.value = [...roles.value, role]
  creating.value = false
  selId.value = id
  editing.value = true
  toast({ title: 'Role created', desc: `${name.value} · ${perms.length} permissions` })
}

function toggleEdit() {
  if (editing.value) toast({ title: 'Permissions saved', desc: selected.value.name })
  editing.value = !editing.value
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1 class="page-title">Roles &amp; Permissions</h1>
        <p class="page-sub">
          Define what console operators and organization members can do. System roles apply across all tenants;
          organization roles are templates available inside every tenant.
        </p>
      </div>
      <div class="page-actions">
        <Button variant="primary" icon="plus" @click="creating = true">New role</Button>
      </div>
    </div>

    <div style="margin-bottom: var(--d-gap)">
      <Segmented
        v-model="scope"
        :options="[
          { value: 'system', label: `System roles · ${sysRoles.length}` },
          { value: 'org', label: `Organization roles · ${orgRoles.length}` },
        ]"
      />
    </div>

    <div class="grid-2" style="grid-template-columns: 340px 1fr; align-items: start">
      <div style="display: flex; flex-direction: column; gap: 10px">
        <Card
          v-for="r in roles"
          :key="r.id"
          hover
          class="role-card"
          :style="{
            borderColor: selId === r.id ? 'var(--primary)' : undefined,
            boxShadow: selId === r.id ? '0 0 0 1px var(--primary)' : undefined,
          }"
          @click="selId = r.id; editing = false"
        >
          <div class="role-card__top">
            <span class="role-swatch" :style="{ background: r.color }"><Icon :name="r.system ? 'shield-check' : 'shield'" :size="17" /></span>
            <div style="flex: 1; min-width: 0">
              <div style="display: flex; align-items: center; gap: 7px">
                <span style="font-weight: 600">{{ r.name }}</span>
                <Badge v-if="r.system" variant="muted">Built-in</Badge>
              </div>
              <div style="font-size: 12.5px; color: var(--muted-foreground); margin-top: 2px; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden">
                {{ r.desc }}
              </div>
              <div style="display: flex; gap: 14px; margin-top: 9px; font-size: 12px; color: var(--muted-foreground)">
                <span class="dotline"><Icon name="users" :size="13" /> {{ r.members.toLocaleString() }}</span>
                <span class="dotline"><Icon name="key-round" :size="13" /> {{ r.perms.length }}</span>
              </div>
            </div>
          </div>
        </Card>
      </div>

      <Card>
        <CardHead bordered :desc="selected.desc">
          <template #title>
            <span style="display: flex; align-items: center; gap: 10px">
              <span class="role-swatch" :style="{ background: selected.color, width: '28px', height: '28px' }"><Icon name="shield-check" :size="15" /></span>
              {{ selected.name }}
              <Badge v-if="selected.system" variant="muted">Built-in</Badge>
            </span>
          </template>
          <template #actions>
            <Badge v-if="selected.system" variant="outline" icon="lock">Locked</Badge>
            <Button v-else :variant="editing ? 'primary' : 'outline'" size="sm" :icon="editing ? 'check' : 'pencil'" :icon-size="14" @click="toggleEdit">
              {{ editing ? 'Save changes' : 'Edit permissions' }}
            </Button>
          </template>
        </CardHead>

        <div style="padding: 14px 24px; display: flex; align-items: center; gap: 18px; border-bottom: 1px solid var(--border); flex-wrap: wrap">
          <span class="dotline" style="font-size: 13px"><Icon name="users" :size="15" style="color: var(--muted-foreground)" /> <b>{{ selected.members.toLocaleString() }}</b>&nbsp;members</span>
          <span class="dotline" style="font-size: 13px"><Icon name="key-round" :size="15" style="color: var(--muted-foreground)" /> <b>{{ selected.perms.length }}</b>&nbsp;of {{ ALL_PERMS.length }} permissions</span>
          <span class="dotline" style="font-size: 13px"><Icon name="globe" :size="15" style="color: var(--muted-foreground)" /> {{ selected.scope === 'system' ? 'Platform-wide' : 'Per organization' }}</span>
          <Badge v-if="editing" variant="info" style="margin-left: auto">Editing — toggle permissions below</Badge>
        </div>

        <div style="padding: 24px">
          <div v-for="g in PERMISSION_GROUPS" :key="g.id" class="perm-group">
            <div class="perm-group__head">
              <Icon :name="groupIcon(g.id)" :size="15" style="color: var(--muted-foreground)" />
              <span style="flex: 1">{{ g.label }}</span>
              <span style="font-size: 12px; font-weight: 500; color: var(--muted-foreground)">{{ grantedCount(g.perms) }}/{{ g.perms.length }}</span>
            </div>
            <div v-for="p in g.perms" :key="p.id" class="perm-row">
              <div class="perm-row__main">
                <div class="perm-row__label">{{ p.label }}</div>
                <div class="perm-row__desc">{{ p.desc }}</div>
              </div>
              <span class="mono" style="font-size: 11.5px; color: var(--muted-foreground)">{{ p.id }}</span>
              <Switch
                v-if="editing && !selected.system"
                :checked="selected.perms.includes(p.id)"
                size="sm"
                @change="togglePerm(p.id)"
              />
              <template v-else>
                <Icon v-if="selected.perms.includes(p.id)" name="check-circle" :size="17" class="perm-check" />
                <Icon v-else name="minus" :size="17" class="perm-x" />
              </template>
            </div>
          </div>
        </div>
      </Card>
    </div>

    <!-- create role -->
    <Dialog
      :open="creating"
      :width="620"
      :title="`Create ${scope === 'system' ? 'system' : 'organization'} role`"
      :desc="scope === 'system' ? 'Applies to console operators across all tenants.' : 'A reusable role template available inside every organization.'"
      @close="creating = false"
    >
      <div style="display: flex; flex-direction: column; gap: 16px">
        <div style="display: grid; grid-template-columns: 1fr auto; gap: 14px; align-items: end">
          <Field label="Role name" required :error="nameErr">
            <Input placeholder="e.g. Compliance Reviewer" v-model="name" :error="nameErr" autofocus />
          </Field>
          <Field label="Color">
            <div style="display: flex; gap: 6px">
              <button
                v-for="c in SWATCHES.slice(0, 6)"
                :key="c"
                :aria-label="c"
                :style="{
                  width: '26px',
                  height: '26px',
                  borderRadius: '7px',
                  background: c,
                  border: color === c ? '2px solid var(--foreground)' : '2px solid transparent',
                  boxShadow: color === c ? '0 0 0 2px var(--background) inset' : 'none',
                }"
                @click="color = c"
              />
            </div>
          </Field>
        </div>
        <Field label="Description">
          <Textarea placeholder="What can people with this role do?" v-model="desc" rows="2" />
        </Field>
        <div>
          <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px">
            <span class="field__label">Permissions</span>
            <span style="font-size: 12px; color: var(--muted-foreground)">{{ perms.length }} of {{ ALL_PERMS.length }} selected</span>
          </div>
          <div style="max-height: 280px; overflow: auto; border: 1px solid var(--border); border-radius: 8px">
            <div v-for="(g, gi) in PERMISSION_GROUPS" :key="g.id" :style="{ borderTop: gi ? '1px solid var(--border)' : 'none' }">
              <button
                style="width: 100%; display: flex; align-items: center; gap: 9px; padding: 9px 14px; background: var(--subtle); text-align: left"
                @click="toggleGroup(g)"
              >
                <span
                  :style="{
                    width: '16px',
                    height: '16px',
                    borderRadius: '4px',
                    border: '1.5px solid ' + (groupState(g).all ? 'var(--primary)' : 'var(--border-strong)'),
                    background: groupState(g).all ? 'var(--primary)' : groupState(g).some ? 'var(--accent)' : 'transparent',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    flexShrink: 0,
                  }"
                >
                  <Icon v-if="groupState(g).all" name="check" :size="12" style="color: var(--primary-foreground)" />
                  <Icon v-else-if="groupState(g).some" name="minus" :size="12" style="color: var(--foreground)" />
                </span>
                <span style="font-weight: 600; font-size: 13px; flex: 1">{{ g.label }}</span>
                <span style="font-size: 11.5px; color: var(--muted-foreground)">{{ groupState(g).count }}/{{ g.perms.length }}</span>
              </button>
              <label
                v-for="p in g.perms"
                :key="p.id"
                style="display: flex; align-items: center; gap: 10px; padding: 8px 14px 8px 38px; cursor: pointer; border-top: 1px solid var(--border)"
              >
                <input type="checkbox" :checked="perms.includes(p.id)" style="accent-color: var(--primary); width: 15px; height: 15px" @change="toggleCreatePerm(p.id)" />
                <span style="flex: 1; font-size: 13px">{{ p.label }}</span>
                <span class="mono" style="font-size: 11px; color: var(--muted-foreground)">{{ p.id }}</span>
              </label>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <Button variant="ghost" @click="creating = false">Cancel</Button>
        <Button variant="primary" icon="shield-check" @click="submitCreate">Create role · {{ perms.length }}</Button>
      </template>
    </Dialog>
  </div>
</template>
