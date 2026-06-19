<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { refDebounced } from '@vueuse/core'
import Icon from '@/components/ui/Icon.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Avatar from '@/components/ui/Avatar.vue'
import Input from '@/components/ui/Input.vue'
import Textarea from '@/components/ui/Textarea.vue'
import Field from '@/components/ui/Field.vue'
import SearchBox from '@/components/ui/SearchBox.vue'
import Dialog from '@/components/ui/Dialog.vue'
import Sheet from '@/components/ui/Sheet.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import { useToast } from '@/composables/useToast'
import { ApiError } from '@/lib/api'
import {
  useCreateOrganization,
  useOrganizations,
  type Organization,
  type OrgListParams,
} from '@/composables/useOrganizations'

const toast = useToast()
const route = useRoute()

const PAGE_SIZE = 20

/* ── List ──────────────────────────────────────────────────── */
const q = ref('')
const debouncedQ = refDebounced(q, 300)
const page = ref(1)

// Reset to the first page whenever the search term changes.
watch(debouncedQ, () => (page.value = 1))

const params = computed<OrgListParams>(() => ({
  page: page.value,
  page_size: PAGE_SIZE,
  q: debouncedQ.value.trim() || undefined,
  sort_by: 'created_at',
  sort_dir: 'desc',
}))

const { data, isLoading, isError, error, isFetching, refetch } = useOrganizations(params)

const orgs = computed(() => data.value?.items ?? [])
const total = computed(() => data.value?.total ?? 0)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / PAGE_SIZE)))
const rangeFrom = computed(() => (total.value === 0 ? 0 : (page.value - 1) * PAGE_SIZE + 1))
const rangeTo = computed(() => Math.min(page.value * PAGE_SIZE, total.value))

function fmtDate(s: string) {
  return new Date(s).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}

/* ── Detail drawer ─────────────────────────────────────────── */
const selected = ref<Organization | null>(null)

/* ── Create-organization form ──────────────────────────────── */
const creating = ref(false)
const f = reactive({ name: '', slug: '', description: '' })
const dirty = reactive({ name: false, slug: false })
const errors = reactive({ name: '', slug: '' })
// Tracks whether the user has hand-edited the slug, so we stop auto-deriving it.
const slugTouched = ref(false)

const createMut = useCreateOrganization()

const RESERVED_SLUGS = new Set(['admin', 'auth', 'auth-admin', 'traefik', 'api', 'app', 'www'])

function slugify(s: string) {
  return s
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 40)
}

function validateName() {
  return f.name.trim() ? '' : 'Organization name is required'
}
function validateSlug() {
  const s = f.slug
  if (!/^[a-z0-9-]{2,40}$/.test(s)) return '2–40 chars: lowercase letters, digits, hyphens'
  if (s.startsWith('-') || s.endsWith('-')) return 'No leading or trailing hyphen'
  if (RESERVED_SLUGS.has(s)) return 'This slug is reserved'
  return ''
}

// Validate on every keystroke only after a field has been blurred once.
watch(
  () => f.name,
  (v) => {
    if (dirty.name) errors.name = validateName()
    if (!slugTouched.value) {
      f.slug = slugify(v)
      if (dirty.slug) errors.slug = validateSlug()
    }
  },
)
watch(
  () => f.slug,
  () => {
    if (dirty.slug) errors.slug = validateSlug()
  },
)
function markNameDirty() {
  dirty.name = true
  errors.name = validateName()
}
function onSlugInput() {
  slugTouched.value = true
}
function markSlugDirty() {
  dirty.slug = true
  errors.slug = validateSlug()
}

// Open via the button or a ?create=1 deep link (command palette etc.).
watch(
  () => route.query.create,
  (v) => {
    if (v) creating.value = true
  },
  { immediate: true },
)
watch(creating, (open) => {
  if (open) {
    f.name = ''
    f.slug = ''
    f.description = ''
    dirty.name = false
    dirty.slug = false
    errors.name = ''
    errors.slug = ''
    slugTouched.value = false
  }
})

async function submit() {
  markNameDirty()
  markSlugDirty()
  if (errors.name || errors.slug) return
  try {
    const org = await createMut.mutateAsync({
      name: f.name.trim(),
      slug: f.slug,
      description: f.description.trim(),
    })
    creating.value = false
    toast({
      title: 'Organization created',
      desc: `${org.name} — provisioning in the identity provider…`,
    })
    // The org lands unsynced; surface it in the drawer so the pending state is visible.
    setTimeout(() => (selected.value = org), 150)
  } catch (e) {
    toast({
      title: 'Could not create organization',
      desc: e instanceof ApiError ? e.message : 'Unexpected error',
      variant: 'error',
    })
  }
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1 class="page-title">Organizations</h1>
        <p class="page-sub">{{ total }} organization{{ total === 1 ? '' : 's' }} on the platform.</p>
      </div>
      <div class="page-actions">
        <Button variant="primary" icon="plus" @click="creating = true">New organization</Button>
      </div>
    </div>

    <Card>
      <div style="padding: 14px; border-bottom: 1px solid var(--border)">
        <div class="toolbar">
          <div style="flex: 1 1 240px; min-width: 200px; display: flex">
            <SearchBox v-model="q" placeholder="Search by name or description…" />
          </div>
          <span v-if="isFetching && !isLoading" style="font-size: 12px; color: var(--muted-foreground)">Updating…</span>
        </div>
      </div>

      <!-- Loading (first load) -->
      <div v-if="isLoading" style="padding: 48px; text-align: center; color: var(--muted-foreground)">
        <Icon name="refresh-cw" :size="20" /> Loading organizations…
      </div>

      <!-- Error -->
      <EmptyState
        v-else-if="isError"
        icon="alert-triangle"
        title="Couldn’t load organizations"
        :desc="error instanceof ApiError ? error.message : 'Request failed. Check your connection and try again.'"
      >
        <template #action>
          <Button variant="outline" size="sm" icon="refresh-cw" @click="refetch()">Retry</Button>
        </template>
      </EmptyState>

      <!-- Empty -->
      <EmptyState
        v-else-if="orgs.length === 0"
        icon="building-2"
        :title="debouncedQ ? 'No matching organizations' : 'No organizations yet'"
        :desc="debouncedQ ? 'Try a different search term.' : 'Create the first organization to get started.'"
      >
        <template #action>
          <Button v-if="!debouncedQ" variant="primary" size="sm" icon="plus" @click="creating = true">
            New organization
          </Button>
          <Button v-else variant="outline" size="sm" @click="q = ''">Clear search</Button>
        </template>
      </EmptyState>

      <!-- Table -->
      <div v-else class="table-wrap">
        <table class="table table--rows-clickable">
          <thead>
            <tr>
              <th>Organization</th>
              <th>Identity (Logto)</th>
              <th>Status</th>
              <th>Created</th>
              <th style="width: 44px"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="o in orgs" :key="o.id" @click="selected = o">
              <td>
                <div class="idcell">
                  <Avatar :name="o.name" size="sm" square />
                  <div class="idcell__main">
                    <div class="idcell__name">
                      {{ o.name }}
                      <Badge v-if="!o.slug" variant="violet">Master</Badge>
                    </div>
                    <div class="idcell__sub mono">{{ o.slug || 'apex domain' }}</div>
                  </div>
                </div>
              </td>
              <td>
                <span v-if="o.external_id" class="mono cell-muted">{{ o.external_id }}</span>
                <span v-else class="cell-muted">—</span>
              </td>
              <td>
                <Badge v-if="o.synced" variant="success" dot>Synced</Badge>
                <Badge v-else variant="warning" dot>Pending</Badge>
              </td>
              <td><span class="cell-muted tnum">{{ fmtDate(o.created_at) }}</span></td>
              <td @click.stop>
                <button class="btn btn--ghost btn--icon btn--sm row-actions" @click="selected = o">
                  <Icon name="eye" :size="16" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div
        v-if="!isLoading && !isError && orgs.length"
        style="padding: 12px 16px; border-top: 1px solid var(--border); display: flex; justify-content: space-between; align-items: center"
      >
        <span style="font-size: 12.5px; color: var(--muted-foreground)">
          Showing {{ rangeFrom }}–{{ rangeTo }} of {{ total }}
        </span>
        <div style="display: flex; gap: 4px; align-items: center">
          <Button
            variant="outline"
            size="sm"
            icon="chevron-left"
            :icon-size="14"
            :disabled="page <= 1"
            @click="page--"
          >
            Prev
          </Button>
          <span style="font-size: 12.5px; color: var(--muted-foreground); padding: 0 6px">
            Page {{ page }} / {{ totalPages }}
          </span>
          <Button
            variant="outline"
            size="sm"
            icon-right="chevron-right"
            :icon-size="14"
            :disabled="page >= totalPages"
            @click="page++"
          >
            Next
          </Button>
        </div>
      </div>
    </Card>

    <!-- Detail drawer -->
    <Sheet :open="!!selected" :width="480" @close="selected = null">
      <template #title>
        <span v-if="selected" style="display: flex; align-items: center; gap: 11px">
          <Avatar :name="selected.name" size="md" square /> {{ selected.name }}
        </span>
      </template>
      <template #desc>
        <span v-if="selected" class="mono">{{ selected.id }}</span>
      </template>
      <template v-if="selected" #footer>
        <Button variant="outline" @click="selected = null">Close</Button>
      </template>

      <div v-if="selected" style="display: flex; flex-direction: column; gap: 22px">
        <div style="display: flex; gap: 8px; flex-wrap: wrap">
          <Badge v-if="selected.synced" variant="success" dot>Synced</Badge>
          <Badge v-else variant="warning" dot>Pending sync</Badge>
        </div>

        <div>
          <div style="font-weight: 600; margin-bottom: 12px; font-size: 13.5px">Details</div>
          <dl class="dl">
            <dt>Name</dt><dd>{{ selected.name }}</dd>
            <dt>Slug</dt><dd class="mono">{{ selected.slug || 'apex domain (master)' }}</dd>
            <dt>Description</dt><dd>{{ selected.description || '—' }}</dd>
            <dt>Organization ID</dt><dd class="mono">{{ selected.id }}</dd>
            <dt>Logto ID</dt><dd class="mono">{{ selected.external_id || '— (not yet provisioned)' }}</dd>
            <dt>Created</dt><dd>{{ fmtDate(selected.created_at) }}</dd>
            <dt>Updated</dt><dd>{{ fmtDate(selected.updated_at) }}</dd>
          </dl>
        </div>

        <div
          v-if="!selected.synced"
          style="display: flex; gap: 9px; align-items: flex-start; font-size: 12.5px; background: var(--info-bg); color: var(--info-fg); padding: 11px 13px; border-radius: 8px; border: 1px solid var(--info-bd)"
        >
          <Icon name="info" :size="15" style="flex-shrink: 0; margin-top: 1px" />
          This organization is being provisioned in the identity provider. The Logto ID appears once sync completes.
        </div>
      </div>
    </Sheet>

    <!-- Create-organization form -->
    <Dialog
      :open="creating"
      :width="520"
      title="Create organization"
      desc="Core is the source of truth — the organization is provisioned in Logto right after."
      @close="creating = false"
    >
      <div style="display: flex; flex-direction: column; gap: 16px">
        <Field label="Organization name" required :error="errors.name">
          <Input
            icon="building-2"
            placeholder="Acme Corporation"
            v-model="f.name"
            :error="errors.name"
            autofocus
            @blur="markNameDirty"
          />
        </Field>
        <Field label="URL slug" required :error="errors.slug" hint="Tenant subdomain — auto-filled from the name, editable.">
          <Input
            icon="globe"
            placeholder="acme"
            :model-value="f.slug"
            :error="errors.slug"
            @update:model-value="f.slug = String($event); onSlugInput()"
            @blur="markSlugDirty"
          />
        </Field>
        <Field label="Description" hint="Optional — a short note about this organization.">
          <Textarea v-model="f.description" placeholder="What this organization is for…" />
        </Field>
      </div>

      <template #footer>
        <Button variant="ghost" :disabled="createMut.isPending.value" @click="creating = false">Cancel</Button>
        <Button
          variant="primary"
          icon-right="check"
          :disabled="createMut.isPending.value"
          @click="submit"
        >
          {{ createMut.isPending.value ? 'Creating…' : 'Create organization' }}
        </Button>
      </template>
    </Dialog>
  </div>
</template>
