<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
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
import Sheet from '@/components/ui/Sheet.vue'
import Tag from '@/components/ui/Tag.vue'
import Progress from '@/components/ui/Progress.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import { useNav } from '@/composables/useNav'
import { useToast } from '@/composables/useToast'
import { ORGS, PLANS, PLAN_PRICING, REGIONS, STATUS, fmtMoney, type Org } from '@/lib/data'

const nav = useNav()
const toast = useToast()
const route = useRoute()

const PLAN_OPTS = [{ value: '', label: 'All plans' }, ...Object.keys(PLANS).map((p) => ({ value: p, label: PLANS[p].label }))]
const STATUS_OPTS = [{ value: '', label: 'All statuses' }, ...['active', 'trial', 'past_due', 'suspended'].map((s) => ({ value: s, label: STATUS[s].label }))]
const REGION_OPTS = [{ value: '', label: 'All regions' }, ...Object.keys(REGIONS).map((r) => ({ value: r, label: REGIONS[r] }))]

const orgs = ref<Org[]>([...ORGS])
const q = ref('')
const plan = ref('')
const status = ref('')
const region = ref('')
const selected = ref<Org | null>(null)
const creating = ref(false)

watch(
  () => route.query,
  (query) => {
    if (query.create) creating.value = true
    if (query.open) {
      const o = ORGS.find((x) => x.id === query.open)
      if (o) selected.value = o
    }
  },
  { immediate: true },
)

const filtered = computed(() =>
  orgs.value.filter(
    (o) =>
      (!q.value ||
        o.name.toLowerCase().includes(q.value.toLowerCase()) ||
        o.slug.includes(q.value.toLowerCase()) ||
        o.owner.toLowerCase().includes(q.value.toLowerCase())) &&
      (!plan.value || o.plan === plan.value) &&
      (!status.value || o.status === status.value) &&
      (!region.value || o.region === region.value),
  ),
)

const activeFilters = computed(() =>
  [
    plan.value && { k: 'plan', label: 'Plan: ' + PLANS[plan.value].label, clear: () => (plan.value = '') },
    status.value && { k: 'status', label: 'Status: ' + STATUS[status.value].label, clear: () => (status.value = '') },
    region.value && { k: 'region', label: 'Region: ' + REGIONS[region.value], clear: () => (region.value = '') },
  ].filter(Boolean) as { k: string; label: string; clear: () => void }[],
)

function clearFilters() {
  q.value = ''
  plan.value = ''
  status.value = ''
  region.value = ''
}
function seatPct(o: Org) {
  return Math.round((o.seatsUsed / o.seats) * 100)
}

/* ── Create-organization flow ──────────────────────────────── */
const STEPS = [
  { id: 0, label: 'Details' },
  { id: 1, label: 'Plan & seats' },
  { id: 2, label: 'Owner' },
  { id: 3, label: 'Review' },
]
const slugify = (s: string) => s.toLowerCase().trim().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')

const step = ref(0)
const touched = ref(false)
const f = reactive({ name: '', slug: '', region: 'us-east', industry: '', plan: 'growth', seats: 25, ownerName: '', ownerEmail: '' })

watch(creating, (open) => {
  if (open) {
    step.value = 0
    touched.value = false
    Object.assign(f, { name: '', slug: '', region: 'us-east', industry: '', plan: 'growth', seats: 25, ownerName: '', ownerEmail: '' })
  }
})

const emailValid = computed(() => /^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(f.ownerEmail))
const nameErr = computed(() => (touched.value && !f.name.trim() ? 'Organization name is required' : null))
const emailErr = computed(() => (touched.value && step.value === 2 && !emailValid.value ? 'Enter a valid email' : null))
const canNext = computed(() =>
  step.value === 0 ? !!f.name.trim() : step.value === 2 ? emailValid.value && !!f.ownerName.trim() : true,
)

function setName(v: string) {
  f.name = v
  f.slug = slugify(v)
}

function next() {
  touched.value = true
  if (!canNext.value) return
  if (step.value < 3) {
    step.value++
    touched.value = false
  } else {
    handleCreate()
  }
}

function handleCreate() {
  const id = 'org_' + Math.random().toString(16).slice(2, 6)
  const newOrg: Org = {
    id,
    name: f.name,
    slug: f.slug,
    plan: f.plan,
    status: f.plan === 'free' ? 'active' : 'trial',
    seats: f.seats,
    seatsUsed: 1,
    mrr: 0,
    region: f.region,
    owner: f.ownerName,
    created: '2026-06-08',
    industry: f.industry || '—',
  }
  orgs.value = [newOrg, ...orgs.value]
  creating.value = false
  toast({ title: 'Organization created', desc: `${f.name} · invite sent to ${f.ownerEmail}` })
  setTimeout(() => (selected.value = newOrg), 200)
}

function suspend(o: Org) {
  toast({ title: 'Organization suspended', desc: o.name, variant: 'error' })
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1 class="page-title">Organizations</h1>
        <p class="page-sub">{{ orgs.length }} tenant organizations across {{ Object.keys(REGIONS).length }} regions.</p>
      </div>
      <div class="page-actions">
        <Button variant="outline" icon="download">Export CSV</Button>
        <Button variant="primary" icon="plus" @click="creating = true">New organization</Button>
      </div>
    </div>

    <Card>
      <div style="padding: 14px; display: flex; flex-direction: column; gap: 12px; border-bottom: 1px solid var(--border)">
        <div class="toolbar">
          <div style="flex: 1 1 240px; min-width: 200px; display: flex">
            <SearchBox v-model="q" placeholder="Search by name, slug or owner…" />
          </div>
          <Select v-model="plan" :options="PLAN_OPTS" filter />
          <Select v-model="status" :options="STATUS_OPTS" filter />
          <Select v-model="region" :options="REGION_OPTS" filter />
        </div>
        <div v-if="activeFilters.length" style="display: flex; gap: 8px; align-items: center; flex-wrap: wrap">
          <span style="font-size: 12px; color: var(--muted-foreground)">Filters:</span>
          <Tag v-for="af in activeFilters" :key="af.k" @remove="af.clear()">{{ af.label }}</Tag>
          <button style="font-size: 12px; color: var(--muted-foreground); text-decoration: underline" @click="clearFilters">
            Clear all
          </button>
        </div>
      </div>

      <EmptyState
        v-if="filtered.length === 0"
        icon="building-2"
        title="No organizations found"
        desc="Try adjusting your search or filters."
      >
        <template #action>
          <Button variant="outline" size="sm" @click="clearFilters">Clear filters</Button>
        </template>
      </EmptyState>

      <div v-else class="table-wrap">
        <table class="table table--rows-clickable">
          <thead>
            <tr>
              <th>Organization</th><th>Plan</th><th>Status</th><th>Seats</th>
              <th class="th-right">MRR</th><th>Owner</th><th>Created</th><th style="width: 44px"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="o in filtered" :key="o.id" @click="selected = o">
              <td>
                <div class="idcell">
                  <Avatar :name="o.name" size="sm" square />
                  <div class="idcell__main">
                    <div class="idcell__name">{{ o.name }}</div>
                    <div class="idcell__sub mono">{{ o.slug }}</div>
                  </div>
                </div>
              </td>
              <td><Badge :variant="PLANS[o.plan].variant">{{ PLANS[o.plan].label }}</Badge></td>
              <td><Badge :variant="STATUS[o.status].variant" dot>{{ STATUS[o.status].label }}</Badge></td>
              <td>
                <div class="meter">
                  <div class="meter__track"><Progress :value="seatPct(o)" :color="seatPct(o) > 90 ? 'var(--warning)' : undefined" /></div>
                  <span class="meter__val tnum">{{ o.seatsUsed }}/{{ o.seats }}</span>
                </div>
              </td>
              <td class="td-right cell-strong tnum">{{ fmtMoney(o.mrr) }}</td>
              <td><span class="cell-muted">{{ o.owner }}</span></td>
              <td><span class="cell-muted tnum">{{ o.created }}</span></td>
              <td @click.stop>
                <Dropdown align="end">
                  <template #trigger>
                    <button class="btn btn--ghost btn--icon btn--sm row-actions"><Icon name="more-horizontal" :size="16" /></button>
                  </template>
                  <MenuItem icon="eye" @click="selected = o">View details</MenuItem>
                  <MenuItem icon="pencil">Edit</MenuItem>
                  <MenuItem icon="users" @click="nav('users')">Members</MenuItem>
                  <MenuItem icon="credit-card" @click="nav('billing')">Billing</MenuItem>
                  <MenuSep />
                  <MenuItem icon="lock" danger @click="suspend(o)">Suspend</MenuItem>
                </Dropdown>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div style="padding: 12px 16px; border-top: 1px solid var(--border); display: flex; justify-content: space-between; align-items: center">
        <span style="font-size: 12.5px; color: var(--muted-foreground)">Showing {{ filtered.length }} of {{ orgs.length }} organizations</span>
        <div style="display: flex; gap: 4px">
          <Button variant="outline" size="sm" disabled icon="chevron-left" :icon-size="14">Prev</Button>
          <Button variant="outline" size="sm" icon-right="chevron-right" :icon-size="14">Next</Button>
        </div>
      </div>
    </Card>

    <!-- Detail drawer -->
    <Sheet :open="!!selected" :width="520" @close="selected = null">
      <template #title>
        <span v-if="selected" style="display: flex; align-items: center; gap: 11px">
          <Avatar :name="selected.name" size="md" square /> {{ selected.name }}
        </span>
      </template>
      <template #desc>
        <span v-if="selected" class="mono">{{ selected.id }} · {{ selected.slug }}</span>
      </template>
      <template v-if="selected" #footer>
        <Button variant="outline" @click="selected = null">Close</Button>
        <Button
          variant="primary"
          icon="external-link"
          @click="toast({ title: 'Opening tenant', desc: selected!.name, variant: 'info' })"
        >
          Open tenant
        </Button>
      </template>

      <div v-if="selected" style="display: flex; flex-direction: column; gap: 22px">
        <div style="display: flex; gap: 8px; flex-wrap: wrap">
          <Badge :variant="PLANS[selected.plan].variant">{{ PLANS[selected.plan].label }}</Badge>
          <Badge :variant="STATUS[selected.status].variant" dot>{{ STATUS[selected.status].label }}</Badge>
          <Badge variant="outline" icon="globe">{{ REGIONS[selected.region] }}</Badge>
          <Badge variant="outline">{{ selected.industry }}</Badge>
        </div>

        <div class="grid-3" style="grid-template-columns: 1fr 1fr; gap: 12px">
          <Card pad>
            <div class="kpi__label">MRR</div>
            <div style="font-size: 22px; font-weight: 700; margin-top: 4px" class="tnum">{{ fmtMoney(selected.mrr) }}</div>
          </Card>
          <Card pad>
            <div class="kpi__label">Seats used</div>
            <div style="font-size: 22px; font-weight: 700; margin-top: 4px" class="tnum">
              {{ selected.seatsUsed }}<span style="font-size: 14px; color: var(--muted-foreground); font-weight: 500">/{{ selected.seats }}</span>
            </div>
            <div style="margin-top: 8px">
              <Progress :value="seatPct(selected)" :color="seatPct(selected) > 90 ? 'var(--warning)' : undefined" />
            </div>
          </Card>
        </div>

        <div>
          <div style="font-weight: 600; margin-bottom: 12px; font-size: 13.5px">Details</div>
          <dl class="dl">
            <dt>Owner</dt>
            <dd style="display: flex; align-items: center; gap: 8px"><Avatar :name="selected.owner" size="xs" /> {{ selected.owner }}</dd>
            <dt>Organization ID</dt><dd class="mono">{{ selected.id }}</dd>
            <dt>Slug</dt><dd class="mono">{{ selected.slug }}</dd>
            <dt>Region</dt><dd>{{ REGIONS[selected.region] }}</dd>
            <dt>Industry</dt><dd>{{ selected.industry }}</dd>
            <dt>Created</dt><dd>{{ selected.created }}</dd>
          </dl>
        </div>

        <div>
          <div style="font-weight: 600; margin-bottom: 10px; font-size: 13.5px">Quick actions</div>
          <div style="display: flex; flex-wrap: wrap; gap: 9px">
            <Button variant="outline" size="sm" icon="users" :icon-size="14" @click="selected = null; nav('users')">Members</Button>
            <Button variant="outline" size="sm" icon="credit-card" :icon-size="14" @click="selected = null; nav('billing')">Billing</Button>
            <Button variant="outline" size="sm" icon="pencil" :icon-size="14">Edit</Button>
            <Button
              variant="danger-outline"
              size="sm"
              icon="lock"
              :icon-size="14"
              @click="toast({ title: 'Organization suspended', desc: selected!.name, variant: 'error' })"
            >
              Suspend
            </Button>
          </div>
        </div>
      </div>
    </Sheet>

    <!-- Create-organization flow -->
    <Dialog :open="creating" :width="580" title="Create organization" desc="Provision a new tenant on the platform." @close="creating = false">
      <div class="stepper" style="margin-bottom: 24px">
        <div
          v-for="(s, i) in STEPS"
          :key="s.id"
          :class="['step', step === i ? 'step--active' : step > i ? 'step--done' : '']"
          :style="{ flex: i < STEPS.length - 1 ? 1 : 'none' }"
        >
          <span class="step__dot"><Icon v-if="step > i" name="check" :size="13" /><template v-else>{{ i + 1 }}</template></span>
          <span class="step__label">{{ s.label }}</span>
          <span v-if="i < STEPS.length - 1" class="step__bar" />
        </div>
      </div>

      <div v-if="step === 0" style="display: flex; flex-direction: column; gap: 16px">
        <Field label="Organization name" required :error="nameErr">
          <Input icon="building-2" placeholder="Acme Corporation" :model-value="f.name" :error="nameErr" autofocus @update:model-value="setName(String($event))" />
        </Field>
        <Field label="URL slug" hint="Used in the tenant URL — auto-generated from the name.">
          <Input icon="globe" :model-value="f.slug" placeholder="acme" @update:model-value="f.slug = slugify(String($event))" />
        </Field>
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 14px">
          <Field label="Region">
            <Select v-model="f.region" :options="Object.keys(REGIONS).map((r) => ({ value: r, label: REGIONS[r] }))" />
          </Field>
          <Field label="Industry"><Input v-model="f.industry" placeholder="e.g. Software" /></Field>
        </div>
      </div>

      <div v-else-if="step === 1" style="display: flex; flex-direction: column; gap: 14px">
        <div style="display: flex; flex-direction: column; gap: 10px">
          <div
            v-for="p in PLAN_PRICING"
            :key="p.plan"
            :class="['radio-card', f.plan === p.plan && 'radio-card--active']"
            @click="f.plan = p.plan"
          >
            <span class="radio-card__dot" />
            <div style="flex: 1">
              <div style="display: flex; justify-content: space-between; align-items: center">
                <span style="font-weight: 600; text-transform: capitalize">{{ p.plan }}</span>
                <span style="font-weight: 600" class="tnum">
                  {{ p.price === null ? 'Custom' : p.price === 0 ? 'Free' : `$${p.price}` }}
                  <span style="font-size: 11.5px; color: var(--muted-foreground); font-weight: 500">{{ p.price ? ' /seat/mo' : '' }}</span>
                </span>
              </div>
              <div style="font-size: 12.5px; color: var(--muted-foreground); margin-top: 2px">{{ p.features.slice(0, 3).join(' · ') }}</div>
            </div>
          </div>
        </div>
        <Field label="Initial seats" hint="You can change this at any time from billing.">
          <Input type="number" min="1" icon="users" :model-value="f.seats" @update:model-value="f.seats = Number($event) || 0" />
        </Field>
      </div>

      <div v-else-if="step === 2" style="display: flex; flex-direction: column; gap: 16px">
        <div style="font-size: 13px; color: var(--muted-foreground)">The owner receives an email invitation and full control of the organization.</div>
        <Field label="Owner name" required :error="touched && !f.ownerName.trim() ? 'Required' : null">
          <Input icon="user" placeholder="Jane Doe" v-model="f.ownerName" autofocus />
        </Field>
        <Field label="Owner email" required :error="emailErr">
          <Input icon="mail" type="email" placeholder="jane@acme.com" v-model="f.ownerEmail" :error="emailErr" />
        </Field>
      </div>

      <div v-else-if="step === 3" style="display: flex; flex-direction: column; gap: 18px">
        <div style="display: flex; align-items: center; gap: 12px">
          <Avatar :name="f.name || 'New Org'" size="lg" square />
          <div>
            <div style="font-weight: 700; font-size: 17px">{{ f.name || 'Untitled organization' }}</div>
            <div class="mono" style="font-size: 12.5px; color: var(--muted-foreground)">{{ f.slug || 'slug' }}</div>
          </div>
        </div>
        <Card pad style="background: var(--subtle)">
          <dl class="dl">
            <dt>Plan</dt><dd style="text-transform: capitalize">{{ f.plan }} · {{ f.seats }} seats</dd>
            <dt>Region</dt><dd>{{ REGIONS[f.region] }}</dd>
            <dt>Industry</dt><dd>{{ f.industry || '—' }}</dd>
            <dt>Owner</dt><dd>{{ f.ownerName }} <span class="cell-muted">· {{ f.ownerEmail }}</span></dd>
          </dl>
        </Card>
        <div style="display: flex; gap: 9px; align-items: flex-start; font-size: 12.5px; background: var(--info-bg); color: var(--info-fg); padding: 11px 13px; border-radius: 8px; border: 1px solid var(--info-bd)">
          <Icon name="info" :size="15" style="flex-shrink: 0; margin-top: 1px" />
          An invitation email will be sent to the owner immediately after creation.
        </div>
      </div>

      <template #footer>
        <Button variant="ghost" @click="step === 0 ? (creating = false) : step--">{{ step === 0 ? 'Cancel' : 'Back' }}</Button>
        <Button variant="primary" :icon-right="step < 3 ? 'arrow-right' : 'check'" @click="next">
          {{ step < 3 ? 'Continue' : 'Create organization' }}
        </Button>
      </template>
    </Dialog>
  </div>
</template>
